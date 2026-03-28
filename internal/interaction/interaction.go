package interaction

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

const interactionAttempts = 3

// Interactor performs UI interactions on a running Flutter app via VM Service.
type Interactor struct {
	mu       sync.Mutex
	debugURL string
}

// SemanticsNode represents a node in the Flutter Semantics Tree.
type SemanticsNode struct {
	ID       int              `json:"id"`
	Label    string           `json:"label,omitempty"`
	Value    string           `json:"value,omitempty"`
	Hint     string           `json:"hint,omitempty"`
	Rect     Rect             `json:"rect"`
	Actions  []string         `json:"actions,omitempty"`
	Flags    []string         `json:"flags,omitempty"`
	Children []*SemanticsNode `json:"children,omitempty"`
}

// Rect represents a rectangle in logical coordinates.
type Rect struct {
	Left   float64 `json:"left"`
	Top    float64 `json:"top"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Center returns the center point of the rect.
func (r Rect) Center() (float64, float64) {
	return r.Left + r.Width/2, r.Top + r.Height/2
}

// FinderType specifies how to locate a target widget.
type FinderType string

const (
	FindByText    FinderType = "text"
	FindByType    FinderType = "type"
	FindByTooltip FinderType = "tooltip"
)

// Finder describes how to locate a target element.
type Finder struct {
	By    FinderType `json:"by"`
	Value string     `json:"value"`
	Index int        `json:"index"`
}

// InteractionResult holds the result of a UI interaction.
type InteractionResult struct {
	Status  string `json:"status"`
	Action  string `json:"action"`
	Finder  string `json:"finder,omitempty"`
	Target  string `json:"target,omitempty"`
	Details string `json:"details,omitempty"`
}

type extensionPayload struct {
	Status          string `json:"status"`
	Action          string `json:"action,omitempty"`
	Text            string `json:"text,omitempty"`
	ObservedText    string `json:"observed_text,omitempty"`
	CurrentText     string `json:"current_text,omitempty"`
	Focused         bool   `json:"focused,omitempty"`
	FocusedEditable bool   `json:"focused_editable,omitempty"`
	Error           string `json:"error,omitempty"`
}

// NewInteractor creates a new Interactor.
func NewInteractor(debugURL string) *Interactor {
	return &Interactor{debugURL: debugURL}
}

// Tap taps on a widget found by the given finder.
func (it *Interactor) Tap(finder Finder) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		node, err := it.findNode(conn, isolateID, finder)
		if err != nil {
			return nil, err
		}

		if hasAction(node, "tap") {
			if err := it.callFlarnessSemanticsAction(conn, isolateID, node.ID, "tap", ""); err != nil {
				return nil, fmt.Errorf("semantics tap failed on %s: %w", nodeLabel(node), err)
			}
			return &InteractionResult{
				Status:  "ok",
				Action:  "tap",
				Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
				Target:  nodeLabel(node),
				Details: fmt.Sprintf("semantics tap on node #%d", node.ID),
			}, nil
		}

		cx, cy := node.Rect.Center()
		if err := it.callFlarnessTapAt(conn, isolateID, cx, cy); err != nil {
			return nil, fmt.Errorf("tap failed on %s at (%.0f, %.0f): %w", nodeLabel(node), cx, cy, err)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "tap",
			Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
			Target:  nodeLabel(node),
			Details: fmt.Sprintf("tapAt (%.0f, %.0f)", cx, cy),
		}, nil
	})
}

// TapAt taps at a logical coordinate.
func (it *Interactor) TapAt(x, y float64) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		if err := it.callFlarnessTapAt(conn, isolateID, x, y); err != nil {
			return nil, fmt.Errorf("tapAt (%.0f, %.0f) failed: %w", x, y, err)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "tap",
			Target:  fmt.Sprintf("(%.0f, %.0f)", x, y),
			Details: "tapAt service extension",
		}, nil
	})
}

// Type enters text into the currently focused text field.
func (it *Interactor) Type(text string, clear bool, appendMode bool) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		payload, err := it.callFlarnessType(conn, isolateID, text, clear, appendMode)
		if err != nil {
			return nil, fmt.Errorf("type failed: %w", err)
		}

		details := fmt.Sprintf("typed %q into focused field", text)
		if clear && text == "" {
			details = "cleared focused field"
		} else if appendMode {
			details = fmt.Sprintf("appended %q to focused field", text)
		}
		if payload.ObservedText != "" || payload.CurrentText != "" {
			details = fmt.Sprintf("%s (observed=%q current=%q focused=%t)", details, payload.ObservedText, payload.CurrentText, payload.Focused)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "type",
			Details: details,
		}, nil
	})
}

// SwipeOn swipes a widget found by the given finder in the specified direction.
func (it *Interactor) SwipeOn(finder Finder, dx, dy float64, durationMs int) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		node, err := it.findNode(conn, isolateID, finder)
		if err != nil {
			return nil, err
		}

		cx, cy := node.Rect.Center()
		if err := it.callFlarnessSwipe(conn, isolateID, cx, cy, cx+dx, cy+dy, durationMs); err != nil {
			return nil, fmt.Errorf("swipe failed on %s: %w", nodeLabel(node), err)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "swipe",
			Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
			Target:  nodeLabel(node),
			Details: fmt.Sprintf("swipe from (%.0f,%.0f) by (%.0f,%.0f)", cx, cy, dx, dy),
		}, nil
	})
}

// Scroll scrolls a scrollable widget found by the given finder.
func (it *Interactor) Scroll(finder Finder, dx, dy float64) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		node, err := it.findNode(conn, isolateID, finder)
		if err != nil {
			return nil, err
		}

		var action string
		if dy < 0 && hasAction(node, "scrollUp") {
			action = "scrollUp"
		} else if dy > 0 && hasAction(node, "scrollDown") {
			action = "scrollDown"
		} else if dx < 0 && hasAction(node, "scrollLeft") {
			action = "scrollLeft"
		} else if dx > 0 && hasAction(node, "scrollRight") {
			action = "scrollRight"
		}

		if action == "" {
			return nil, fmt.Errorf("scroll failed: no matching scroll action on %s (dx=%.0f, dy=%.0f, available actions: %v)",
				nodeLabel(node), dx, dy, node.Actions)
		}

		if err := it.callFlarnessSemanticsAction(conn, isolateID, node.ID, action, ""); err != nil {
			return nil, fmt.Errorf("scroll %s failed on %s: %w", action, nodeLabel(node), err)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "scroll",
			Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
			Target:  nodeLabel(node),
			Details: fmt.Sprintf("%s via semantics", action),
		}, nil
	})
}

// LongPress performs a long press on a widget found by the given finder.
func (it *Interactor) LongPress(finder Finder, durationMs int) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*InteractionResult, error) {
		node, err := it.findNode(conn, isolateID, finder)
		if err != nil {
			return nil, err
		}

		if !hasAction(node, "longPress") {
			return nil, fmt.Errorf("longPress failed: node %s does not support longPress (available actions: %v)",
				nodeLabel(node), node.Actions)
		}

		if err := it.callFlarnessSemanticsAction(conn, isolateID, node.ID, "longPress", ""); err != nil {
			return nil, fmt.Errorf("longPress failed on %s: %w", nodeLabel(node), err)
		}

		return &InteractionResult{
			Status:  "ok",
			Action:  "longPress",
			Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
			Target:  nodeLabel(node),
			Details: fmt.Sprintf("semantics longPress on node #%d (duration=%dms)", node.ID, durationMs),
		}, nil
	})
}

// WaitFor waits for a widget matching the finder to appear.
func (it *Interactor) WaitFor(finder Finder, timeout time.Duration) (*InteractionResult, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	startTime := time.Now()
	deadline := startTime.Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		node, err := withInteractionSession(it, func(conn *websocket.Conn, isolateID string) (*SemanticsNode, error) {
			return it.findNode(conn, isolateID, finder)
		})
		if err == nil && node != nil {
			return &InteractionResult{
				Status:  "ok",
				Action:  "waitFor",
				Finder:  fmt.Sprintf("%s=%q", finder.By, finder.Value),
				Target:  nodeLabel(node),
				Details: fmt.Sprintf("found after %s", time.Since(startTime).Round(time.Millisecond)),
			}, nil
		}

		time.Sleep(interval)
	}

	return nil, fmt.Errorf("timeout waiting for %s=%q (waited %s)", finder.By, finder.Value, timeout)
}

// GetSemanticsTree returns the full semantics tree.
func (it *Interactor) GetSemanticsTree() ([]*SemanticsNode, error) {
	it.mu.Lock()
	defer it.mu.Unlock()

	return withInteractionSession(it, func(conn *websocket.Conn, isolateID string) ([]*SemanticsNode, error) {
		return it.getSemanticsTree(conn, isolateID)
	})
}

func (it *Interactor) setup() (*websocket.Conn, string, error) {
	if it.debugURL == "" {
		return nil, "", fmt.Errorf("no debug URL available — is the app running?")
	}

	conn, err := it.connect()
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to VM Service: %w", err)
	}

	isolateID, err := it.getMainIsolateID(conn)
	if err != nil {
		conn.Close()
		return nil, "", fmt.Errorf("failed to get isolate: %w", err)
	}

	_ = it.ensureSemantics(conn, isolateID)
	return conn, isolateID, nil
}

func withInteractionSession[T any](it *Interactor, fn func(conn *websocket.Conn, isolateID string) (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 1; attempt <= interactionAttempts; attempt++ {
		conn, isolateID, err := it.setup()
		if err != nil {
			lastErr = err
			if !isTransientInteractionError(err) || attempt == interactionAttempts {
				return zero, err
			}
			time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
			continue
		}

		result, err := fn(conn, isolateID)
		conn.Close()
		if err == nil {
			return result, nil
		}

		lastErr = err
		if !isTransientInteractionError(err) || attempt == interactionAttempts {
			return zero, err
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return zero, lastErr
}

func (it *Interactor) connect() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 15*time.Second)
		},
	}
	conn, _, err := dialer.Dial(it.debugURL, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (it *Interactor) sendRPC(conn *websocket.Conn, method string, params map[string]any) (json.RawMessage, error) {
	return it.sendRPCWithTimeout(conn, method, params, 30*time.Second)
}

var rpcRequestID int64

func nextRPCID() int64 {
	return atomic.AddInt64(&rpcRequestID, 1)
}

func isTransientInteractionError(err error) bool {
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

func (it *Interactor) sendRPCWithTimeout(conn *websocket.Conn, method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	id := nextRPCID()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}

	if err := conn.WriteJSON(req); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(timeout))
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
			} `json:"error"`
			Method string `json:"method"`
		}

		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}
		if resp.Method != "" && resp.ID == nil {
			continue
		}

		respID, ok := resp.ID.(float64)
		if !ok || int64(respID) != id {
			continue
		}

		if resp.Error != nil {
			return nil, fmt.Errorf("RPC error: %s", resp.Error.Message)
		}

		return resp.Result, nil
	}
}

func (it *Interactor) getMainIsolateID(conn *websocket.Conn) (string, error) {
	result, err := it.sendRPC(conn, "getVM", map[string]any{})
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

	for _, iso := range vm.Isolates {
		if strings.Contains(strings.ToLower(iso.Name), "main") {
			return iso.ID, nil
		}
	}
	return vm.Isolates[0].ID, nil
}

func (it *Interactor) ensureSemantics(conn *websocket.Conn, isolateID string) error {
	_, err := it.sendRPCWithTimeout(conn, "ext.flutter.debugDumpSemanticsTreeInTraversalOrder", map[string]any{
		"isolateId": isolateID,
	}, 5*time.Second)
	return err
}

func (it *Interactor) getSemanticsTree(conn *websocket.Conn, isolateID string) ([]*SemanticsNode, error) {
	result, err := it.sendRPC(conn, "ext.flutter.debugDumpSemanticsTreeInTraversalOrder", map[string]any{
		"isolateId": isolateID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get semantics tree: %w", err)
	}

	var dumpResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &dumpResp); err != nil {
		return nil, fmt.Errorf("failed to parse semantics response: %w", err)
	}
	if dumpResp.Data == "" {
		return nil, fmt.Errorf("semantics tree is empty — ensure semantics are enabled in the app")
	}
	if reason := semanticsUnavailableReason(dumpResp.Data); reason != "" {
		return nil, fmt.Errorf(reason)
	}

	nodes := parseSemanticsTreeDump(dumpResp.Data)
	resolveGlobalRects(nodes, 0, 0)
	return nodes, nil
}

func (it *Interactor) findNode(conn *websocket.Conn, isolateID string, finder Finder) (*SemanticsNode, error) {
	nodes, err := it.getSemanticsTree(conn, isolateID)
	if err != nil {
		return nil, err
	}

	var exactMatches []*SemanticsNode
	var fuzzyMatches []*SemanticsNode
	var search func(nodes []*SemanticsNode)
	search = func(nodes []*SemanticsNode) {
		for _, n := range nodes {
			switch finderScore(n, finder) {
			case 2:
				exactMatches = append(exactMatches, n)
			case 1:
				fuzzyMatches = append(fuzzyMatches, n)
			}
			search(n.Children)
		}
	}
	search(nodes)

	matches := exactMatches
	if len(matches) == 0 {
		matches = fuzzyMatches
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no element found for %s=%q", finder.By, finder.Value)
	}

	idx := finder.Index
	if idx < 0 || idx >= len(matches) {
		idx = 0
	}
	return matches[idx], nil
}

func finderScore(node *SemanticsNode, finder Finder) int {
	switch finder.By {
	case FindByText:
		return textScore(node.Label, finder.Value, node.Value)
	case FindByTooltip:
		return textScore(node.Hint, finder.Value)
	case FindByType:
		for _, f := range node.Flags {
			if strings.Contains(strings.ToLower(f), strings.ToLower(finder.Value)) {
				return 1
			}
		}
		return 0
	default:
		return 0
	}
}

func textScore(candidates ...string) int {
	if len(candidates) < 2 {
		return 0
	}
	target := strings.ToLower(candidates[1])
	for idx, candidate := range candidates {
		if idx == 1 {
			continue
		}
		value := strings.ToLower(candidate)
		if value == target {
			return 2
		}
	}
	for idx, candidate := range candidates {
		if idx == 1 {
			continue
		}
		if strings.Contains(strings.ToLower(candidate), target) {
			return 1
		}
	}
	return 0
}

func (it *Interactor) callFlarnessSemanticsAction(conn *websocket.Conn, isolateID string, nodeID int, action string, args string) error {
	params := map[string]any{
		"isolateId": isolateID,
		"nodeId":    fmt.Sprintf("%d", nodeID),
		"action":    action,
	}
	if args != "" {
		params["args"] = args
	}

	_, err := it.sendRPCWithTimeout(conn, "ext.flarness.semanticsAction", params, 5*time.Second)
	return err
}

func (it *Interactor) callFlarnessTapAt(conn *websocket.Conn, isolateID string, x, y float64) error {
	params := map[string]any{
		"isolateId": isolateID,
		"x":         fmt.Sprintf("%.2f", x),
		"y":         fmt.Sprintf("%.2f", y),
	}

	_, err := it.sendRPCWithTimeout(conn, "ext.flarness.tapAt", params, 5*time.Second)
	return err
}

func (it *Interactor) callFlarnessType(conn *websocket.Conn, isolateID string, text string, clear bool, appendMode bool) (*extensionPayload, error) {
	params := map[string]any{
		"isolateId": isolateID,
		"text":      text,
	}
	if clear {
		params["clear"] = "true"
	}
	if appendMode {
		params["append"] = "true"
	}

	raw, err := it.sendRPCWithTimeout(conn, "ext.flarness.type", params, 5*time.Second)
	if err != nil {
		return nil, err
	}

	payload, err := decodeExtensionPayload(raw)
	if err != nil {
		return nil, err
	}
	if payload.Status == "error" {
		if payload.Error != "" {
			return nil, fmt.Errorf(payload.Error)
		}
		return nil, fmt.Errorf("service extension returned error")
	}
	return payload, nil
}

func decodeExtensionPayload(raw json.RawMessage) (*extensionPayload, error) {
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Result != "" {
		var payload extensionPayload
		if err := json.Unmarshal([]byte(envelope.Result), &payload); err != nil {
			return nil, fmt.Errorf("decode extension result failed: %w", err)
		}
		return &payload, nil
	}

	var payload extensionPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode extension payload failed: %w", err)
	}
	return &payload, nil
}

func (it *Interactor) callFlarnessSwipe(conn *websocket.Conn, isolateID string, x1, y1, x2, y2 float64, durationMs int) error {
	params := map[string]any{
		"isolateId": isolateID,
		"x1":        fmt.Sprintf("%.2f", x1),
		"y1":        fmt.Sprintf("%.2f", y1),
		"x2":        fmt.Sprintf("%.2f", x2),
		"y2":        fmt.Sprintf("%.2f", y2),
		"duration":  fmt.Sprintf("%d", durationMs),
	}

	_, err := it.sendRPCWithTimeout(conn, "ext.flarness.swipe", params, time.Duration(durationMs+5000)*time.Millisecond)
	return err
}

func semanticsUnavailableReason(dump string) string {
	lower := strings.ToLower(dump)

	if strings.Contains(lower, "semantics not generated for") ||
		strings.Contains(lower, "only generates semantics when asked to do so by the platform") ||
		strings.Contains(lower, "turning on an assistive technology") {
		return "semantics is not generated by the platform yet; enable TalkBack/VoiceOver on device, then retry"
	}

	return ""
}

func resolveGlobalRects(nodes []*SemanticsNode, parentLeft, parentTop float64) {
	for _, n := range nodes {
		n.Rect.Left += parentLeft
		n.Rect.Top += parentTop
		resolveGlobalRects(n.Children, n.Rect.Left, n.Rect.Top)
	}
}

func parseSemanticsTreeDump(dump string) []*SemanticsNode {
	lines := strings.Split(dump, "\n")
	var rootNodes []*SemanticsNode
	var currentNode *SemanticsNode
	type stackItem struct {
		node  *SemanticsNode
		level int
	}
	var stack []stackItem

	for _, line := range lines {
		if line == "" {
			continue
		}

		indent := 0
		for _, ch := range line {
			if ch == ' ' || ch == '│' || ch == '├' || ch == '└' || ch == '─' || ch == '\u2502' || ch == '\u251c' || ch == '\u2514' || ch == '\u2500' {
				indent++
			} else {
				break
			}
		}

		trimmed := strings.TrimLeft(line, " │├└─\u2502\u251c\u2514\u2500")
		trimmed = strings.TrimSpace(trimmed)

		if strings.HasPrefix(trimmed, "SemanticsNode#") {
			idStr := strings.TrimPrefix(trimmed, "SemanticsNode#")
			if spaceIdx := strings.IndexAny(idStr, " ("); spaceIdx > 0 {
				idStr = idStr[:spaceIdx]
			}
			id := 0
			fmt.Sscanf(idStr, "%d", &id)

			currentNode = &SemanticsNode{ID: id}
			level := indent / 2

			for len(stack) > 0 && stack[len(stack)-1].level >= level {
				stack = stack[:len(stack)-1]
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1].node
				parent.Children = append(parent.Children, currentNode)
			} else {
				rootNodes = append(rootNodes, currentNode)
			}

			stack = append(stack, stackItem{node: currentNode, level: level})
			continue
		}

		if currentNode == nil {
			continue
		}

		if strings.HasPrefix(trimmed, "Rect.fromLTWH(") || strings.HasPrefix(trimmed, "Rect.fromLTRB(") {
			parseRect(trimmed, currentNode)
		} else if strings.HasPrefix(trimmed, "label:") {
			currentNode.Label = parseQuotedValue(strings.TrimPrefix(trimmed, "label:"))
		} else if strings.HasPrefix(trimmed, "value:") {
			currentNode.Value = parseQuotedValue(strings.TrimPrefix(trimmed, "value:"))
		} else if strings.HasPrefix(trimmed, "hint:") {
			currentNode.Hint = parseQuotedValue(strings.TrimPrefix(trimmed, "hint:"))
		} else if strings.HasPrefix(trimmed, "actions:") {
			actStr := strings.TrimPrefix(trimmed, "actions:")
			actStr = strings.TrimSpace(actStr)
			for _, a := range strings.Split(actStr, ",") {
				a = strings.TrimSpace(a)
				if a != "" {
					currentNode.Actions = append(currentNode.Actions, a)
				}
			}
		} else if strings.HasPrefix(trimmed, "flags:") {
			flagStr := strings.TrimPrefix(trimmed, "flags:")
			flagStr = strings.TrimSpace(flagStr)
			for _, f := range strings.Split(flagStr, ",") {
				f = strings.TrimSpace(f)
				if f != "" {
					currentNode.Flags = append(currentNode.Flags, f)
				}
			}
		}
	}

	return rootNodes
}

func parseRect(s string, node *SemanticsNode) {
	if strings.HasPrefix(s, "Rect.fromLTWH(") {
		inner := strings.TrimPrefix(s, "Rect.fromLTWH(")
		inner = strings.TrimSuffix(inner, ")")
		var l, t, w, h float64
		fmt.Sscanf(inner, "%f, %f, %f, %f", &l, &t, &w, &h)
		node.Rect = Rect{Left: l, Top: t, Width: w, Height: h}
	} else if strings.HasPrefix(s, "Rect.fromLTRB(") {
		inner := strings.TrimPrefix(s, "Rect.fromLTRB(")
		inner = strings.TrimSuffix(inner, ")")
		var l, t, r, b float64
		fmt.Sscanf(inner, "%f, %f, %f, %f", &l, &t, &r, &b)
		node.Rect = Rect{Left: l, Top: t, Width: r - l, Height: b - t}
	}
}

func parseQuotedValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func hasAction(node *SemanticsNode, action string) bool {
	for _, a := range node.Actions {
		if strings.EqualFold(a, action) {
			return true
		}
	}
	return false
}

func nodeLabel(node *SemanticsNode) string {
	if node.Label != "" {
		return node.Label
	}
	if node.Value != "" {
		return node.Value
	}
	if node.Hint != "" {
		return node.Hint
	}
	return fmt.Sprintf("node#%d", node.ID)
}

func semanticsActionIndex(action string) int {
	actions := []string{
		"tap",
		"longPress",
		"scrollLeft",
		"scrollRight",
		"scrollUp",
		"scrollDown",
		"increase",
		"decrease",
		"showOnScreen",
		"moveCursorForwardByCharacter",
		"moveCursorBackwardByCharacter",
		"setSelection",
		"copy",
		"cut",
		"paste",
		"didGainAccessibilityFocus",
		"didLoseAccessibilityFocus",
		"customAction",
		"dismiss",
		"moveCursorForwardByWord",
		"moveCursorBackwardByWord",
		"setText",
	}
	for i, a := range actions {
		if strings.EqualFold(a, action) {
			return i
		}
	}
	return -1
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
