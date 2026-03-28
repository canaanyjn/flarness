package inspector

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Inspector retrieves the Widget tree from a running Flutter application
// via the Dart VM Service Protocol.
type Inspector struct {
	mu       sync.Mutex
	debugURL string // ws:// URL for the VM Service
}

const inspectAttempts = 3

var inspectorRPCID int64

// WidgetNode represents a node in the Widget tree.
type WidgetNode struct {
	Widget      string            `json:"widget"`
	Properties  map[string]string `json:"properties,omitempty"`
	Children    []*WidgetNode     `json:"children,omitempty"`
	Description string            `json:"description,omitempty"`
}

// InspectResult holds the result of a widget tree inspection.
type InspectResult struct {
	Tree       *WidgetNode  `json:"tree,omitempty"`
	RenderTree string       `json:"render_tree,omitempty"`
	Summary    *TreeSummary `json:"summary"`
}

// TreeSummary provides a quick overview of the widget tree.
type TreeSummary struct {
	TotalWidgets int      `json:"total_widgets"`
	MaxDepth     int      `json:"max_depth"`
	TopWidgets   []string `json:"top_widgets"`
}

// NewInspector creates a new Inspector.
func NewInspector(debugURL string) *Inspector {
	return &Inspector{
		debugURL: debugURL,
	}
}

// Inspect retrieves the Widget tree from the running Flutter app.
func (ins *Inspector) Inspect() (*InspectResult, error) {
	ins.mu.Lock()
	defer ins.mu.Unlock()

	if ins.debugURL == "" {
		return nil, fmt.Errorf("no debug URL available — is the app running?")
	}

	var lastErr error
	for attempt := 1; attempt <= inspectAttempts; attempt++ {
		result, err := ins.inspectOnce()
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isTransientVMServiceError(err) || attempt == inspectAttempts {
			break
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return nil, lastErr
}

func (ins *Inspector) inspectOnce() (*InspectResult, error) {
	conn, err := ins.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to VM Service: %w", err)
	}
	defer conn.Close()

	// Step 1: Get the isolate ID.
	isolateID, err := ins.getMainIsolateID(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get isolate: %w", err)
	}

	// Step 2: Try to get the widget summary tree via the Flutter extension.
	tree, err := ins.getWidgetTree(conn, isolateID)
	if err != nil {
		// Fallback: get a simpler render tree description.
		desc, descErr := ins.getRenderTreeDescription(conn, isolateID)
		if descErr != nil {
			return nil, fmt.Errorf("failed to get widget tree: %w", err)
		}
		summary := &TreeSummary{
			TotalWidgets: 0,
			MaxDepth:     0,
			TopWidgets:   []string{},
		}
		return &InspectResult{
			RenderTree: desc,
			Summary:    summary,
		}, nil
	}

	// Step 3: Build summary.
	summary := buildSummary(tree)

	return &InspectResult{
		Tree:    tree,
		Summary: summary,
	}, nil
}

// connect establishes a WebSocket connection to the VM Service.
func (ins *Inspector) connect() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 15*time.Second)
		},
	}

	conn, _, err := dialer.Dial(ins.debugURL, nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// sendRPC sends a JSON-RPC request and returns the response.
func (ins *Inspector) sendRPC(conn *websocket.Conn, method string, params map[string]any) (json.RawMessage, error) {
	id := nextInspectorRPCID()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	// Read responses until we get the matching one (skip events).
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return nil, fmt.Errorf("read failed: %w", err)
		}

		var resp struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      any             `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
				Data    any    `json:"data"`
			} `json:"error"`
			Method string `json:"method"` // For events.
		}

		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		// Skip events (they have "method" but no "id").
		if resp.Method != "" && resp.ID == nil {
			continue
		}

		// Check for matching ID (handle both int and float64 from JSON).
		respID, ok := resp.ID.(float64)
		if !ok {
			continue
		}
		if int64(respID) != id {
			continue
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
		}

		return resp.Result, nil
	}
}

func nextInspectorRPCID() int64 {
	return atomic.AddInt64(&inspectorRPCID, 1)
}

func isTransientVMServiceError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "i/o timeout") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "eof") ||
		strings.Contains(message, "connection reset") ||
		strings.Contains(message, "broken pipe") ||
		strings.Contains(message, "use of closed network connection")
}

// getMainIsolateID gets the ID of the main isolate from the VM.
func (ins *Inspector) getMainIsolateID(conn *websocket.Conn) (string, error) {
	result, err := ins.sendRPC(conn, "getVM", map[string]any{})
	if err != nil {
		return "", err
	}

	var vm struct {
		Isolates []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"isolates"`
	}

	if err := json.Unmarshal(result, &vm); err != nil {
		return "", fmt.Errorf("failed to parse VM info: %w", err)
	}

	if len(vm.Isolates) == 0 {
		return "", fmt.Errorf("no isolates found")
	}

	// Prefer the main isolate.
	for _, iso := range vm.Isolates {
		if strings.Contains(strings.ToLower(iso.Name), "main") {
			return iso.ID, nil
		}
	}

	// Fallback to first isolate.
	return vm.Isolates[0].ID, nil
}

// getWidgetTree gets the widget summary tree via Flutter's inspector extension.
func (ins *Inspector) getWidgetTree(conn *websocket.Conn, isolateID string) (*WidgetNode, error) {
	// First, get the root widget summary tree.
	result, err := ins.sendRPC(conn, "ext.flutter.inspector.getRootWidgetSummaryTree", map[string]any{
		"isolateId": isolateID,
	})
	if err != nil {
		// Try alternative method name.
		result, err = ins.sendRPC(conn, "ext.flutter.inspector.getRootWidgetTree", map[string]any{
			"isolateId": isolateID,
		})
		if err != nil {
			return nil, err
		}
	}

	// Parse the tree.
	return ins.parseWidgetTree(result)
}

// parseWidgetTree converts the raw JSON widget tree into our WidgetNode structure.
func (ins *Inspector) parseWidgetTree(raw json.RawMessage) (*WidgetNode, error) {
	raw = unwrapExtensionEnvelope(raw)

	var rawNode struct {
		Description       string            `json:"description"`
		Type              string            `json:"type"`
		Properties        []json.RawMessage `json:"properties"`
		Children          []json.RawMessage `json:"children"`
		WidgetRuntimeType string            `json:"widgetRuntimeType"`
		HasChildren       bool              `json:"hasChildren"`
	}

	if err := json.Unmarshal(raw, &rawNode); err != nil {
		return nil, fmt.Errorf("failed to parse widget node: %w", err)
	}
	if rawNode.Description == "" && rawNode.Type == "" && rawNode.WidgetRuntimeType == "" && len(rawNode.Children) == 0 {
		return nil, fmt.Errorf("empty widget tree payload")
	}

	node := &WidgetNode{
		Widget:      rawNode.WidgetRuntimeType,
		Description: rawNode.Description,
	}

	if node.Widget == "" {
		node.Widget = rawNode.Type
	}
	if node.Widget == "" {
		node.Widget = rawNode.Description
	}

	// Parse properties.
	if len(rawNode.Properties) > 0 {
		node.Properties = make(map[string]string)
		for _, propRaw := range rawNode.Properties {
			var prop struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Value       any    `json:"value"`
			}
			if err := json.Unmarshal(propRaw, &prop); err == nil {
				if prop.Name != "" {
					val := prop.Description
					if val == "" && prop.Value != nil {
						val = fmt.Sprintf("%v", prop.Value)
					}
					node.Properties[prop.Name] = val
				}
			}
		}
	}

	// Parse children recursively (with depth limit to avoid huge trees).
	for _, childRaw := range rawNode.Children {
		child, err := ins.parseWidgetTree(childRaw)
		if err == nil && child != nil {
			node.Children = append(node.Children, child)
		}
	}

	return node, nil
}

// getRenderTreeDescription gets a text description of the render tree as fallback.
func (ins *Inspector) getRenderTreeDescription(conn *websocket.Conn, isolateID string) (string, error) {
	result, err := ins.sendRPC(conn, "ext.flutter.debugDumpRenderTree", map[string]any{
		"isolateId": isolateID,
	})
	if err != nil {
		return "", err
	}

	result = unwrapExtensionEnvelope(result)

	var resp struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		var text string
		if err := json.Unmarshal(result, &text); err == nil {
			return text, nil
		}
		return string(result), nil
	}
	if resp.Description == "" {
		var text string
		if err := json.Unmarshal(result, &text); err == nil {
			return text, nil
		}
	}

	return resp.Description, nil
}

func unwrapExtensionEnvelope(raw json.RawMessage) json.RawMessage {
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return raw
	}

	for _, candidate := range []json.RawMessage{envelope.Result, envelope.Data} {
		if len(candidate) == 0 || string(candidate) == "null" {
			continue
		}
		var text string
		if err := json.Unmarshal(candidate, &text); err == nil {
			return json.RawMessage(text)
		}
		return candidate
	}

	return raw
}

// buildSummary creates a TreeSummary from a WidgetNode tree.
func buildSummary(tree *WidgetNode) *TreeSummary {
	if tree == nil {
		return &TreeSummary{}
	}

	totalWidgets := 0
	maxDepth := 0
	widgetCounts := make(map[string]int)

	var walk func(node *WidgetNode, depth int)
	walk = func(node *WidgetNode, depth int) {
		if node == nil {
			return
		}
		totalWidgets++
		if depth > maxDepth {
			maxDepth = depth
		}
		if node.Widget != "" {
			widgetCounts[node.Widget]++
		}
		for _, child := range node.Children {
			walk(child, depth+1)
		}
	}
	walk(tree, 0)

	// Get top widgets by count.
	type widgetCount struct {
		name  string
		count int
	}
	var sorted []widgetCount
	for name, count := range widgetCounts {
		sorted = append(sorted, widgetCount{name, count})
	}
	// Simple insertion sort (small list).
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].count > sorted[j-1].count; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	topWidgets := make([]string, 0, 10)
	for i, wc := range sorted {
		if i >= 10 {
			break
		}
		topWidgets = append(topWidgets, fmt.Sprintf("%s(%d)", wc.name, wc.count))
	}

	return &TreeSummary{
		TotalWidgets: totalWidgets,
		MaxDepth:     maxDepth,
		TopWidgets:   topWidgets,
	}
}

// PruneTree returns a simplified version of the tree with max depth.
func PruneTree(tree *WidgetNode, maxDepth int) *WidgetNode {
	if tree == nil {
		return nil
	}
	return pruneNode(tree, 0, maxDepth)
}

func pruneNode(node *WidgetNode, currentDepth, maxDepth int) *WidgetNode {
	if node == nil {
		return nil
	}

	pruned := &WidgetNode{
		Widget:      node.Widget,
		Description: node.Description,
		Properties:  node.Properties,
	}

	if currentDepth >= maxDepth {
		if len(node.Children) > 0 {
			pruned.Description = fmt.Sprintf("(%d children omitted)", countDescendants(node))
		}
		return pruned
	}

	for _, child := range node.Children {
		prunedChild := pruneNode(child, currentDepth+1, maxDepth)
		if prunedChild != nil {
			pruned.Children = append(pruned.Children, prunedChild)
		}
	}

	return pruned
}

func countDescendants(node *WidgetNode) int {
	if node == nil {
		return 0
	}
	count := len(node.Children)
	for _, child := range node.Children {
		count += countDescendants(child)
	}
	return count
}
