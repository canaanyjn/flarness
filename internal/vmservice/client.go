package vmservice

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Client performs JSON-RPC calls against a Flutter VM service websocket.
type Client struct {
	debugURL string
}

// NewClient creates a VM service client for the provided websocket URL.
func NewClient(debugURL string) *Client {
	return &Client{debugURL: debugURL}
}

// CallExtension invokes a service extension on the main isolate.
func (c *Client) CallExtension(method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	isolateID, err := c.getMainIsolateID(conn)
	if err != nil {
		return nil, err
	}

	callParams := map[string]any{
		"isolateId": isolateID,
	}
	for key, value := range params {
		callParams[key] = value
	}

	raw, err := c.sendRPCWithTimeout(conn, method, callParams, timeout)
	if err != nil {
		return nil, NormalizeExtensionError(err, method)
	}
	return raw, nil
}

// DecodeExtensionResult decodes a service extension response, handling both
// the standard ServiceExtensionResponse.result JSON string envelope and raw JSON.
func DecodeExtensionResult(raw json.RawMessage, out any) error {
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Result != "" {
		if err := json.Unmarshal([]byte(envelope.Result), out); err != nil {
			return fmt.Errorf("decode extension result failed: %w", err)
		}
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode extension payload failed: %w", err)
	}
	return nil
}

// NormalizeExtensionError rewrites the common "Unknown method" response into
// a concrete app integration hint.
func NormalizeExtensionError(err error, method string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.Contains(strings.ToLower(message), "unknown method") && strings.Contains(message, method) {
		return fmt.Errorf("%s is unavailable; ensure flarness_plugin is added to the app and FlarnessPluginBinding.ensureInitialized() runs in debug mode", method)
	}
	return err
}

func (c *Client) connect() (*websocket.Conn, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 15*time.Second)
		},
	}
	conn, _, err := dialer.Dial(c.debugURL, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

var rpcRequestID int64

func nextRPCID() int64 {
	return atomic.AddInt64(&rpcRequestID, 1)
}

func (c *Client) sendRPC(conn *websocket.Conn, method string, params map[string]any) (json.RawMessage, error) {
	return c.sendRPCWithTimeout(conn, method, params, 30*time.Second)
}

func (c *Client) sendRPCWithTimeout(conn *websocket.Conn, method string, params map[string]any, timeout time.Duration) (json.RawMessage, error) {
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
			ID     any             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
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

func (c *Client) getMainIsolateID(conn *websocket.Conn) (string, error) {
	result, err := c.sendRPC(conn, "getVM", map[string]any{})
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
