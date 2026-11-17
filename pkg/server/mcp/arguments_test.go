package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"

	mcpprotocol "github.com/mark3labs/mcp-go/mcp"

	"example.invalid/mcp-template-module-placeholder/pkg/core/config"
	"example.invalid/mcp-template-module-placeholder/pkg/toolset"
)

func TestToolCallArgumentBoundary(t *testing.T) {
	var calls atomic.Int32
	server, err := NewServer(Configuration{
		StaticConfig: &config.StaticConfig{LogLevel: "info"},
		Toolsets:     []toolset.Toolset{countingToolset{calls: &calls}},
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	defer server.Close()

	t.Run("array does not run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": []any{}})
		if calls.Load() != before {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before)
		}
		assertToolError(t, result, "expected an object")
	})

	t.Run("string does not run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": "hello"})
		if calls.Load() != before {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before)
		}
		assertToolError(t, result, "expected an object")
	})

	t.Run("number does not run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": 1})
		if calls.Load() != before {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before)
		}
		assertToolError(t, result, "expected an object")
	})

	t.Run("omitted arguments run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{})
		if calls.Load() != before+1 {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before+1)
		}
		assertToolText(t, result, "ran")
	})

	t.Run("null arguments run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": nil})
		if calls.Load() != before+1 {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before+1)
		}
		assertToolText(t, result, "ran")
	})

	t.Run("object arguments run the tool", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": map[string]any{"note": "ok"}})
		if calls.Load() != before+1 {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before+1)
		}
		assertToolText(t, result, "ran")
	})

	t.Run("tool failure stays a tool error", func(t *testing.T) {
		before := calls.Load()
		result := callMark(t, server, map[string]any{"arguments": map[string]any{"fail": true}})
		if calls.Load() != before+1 {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before+1)
		}
		assertToolError(t, result, "message is required")
	})

	t.Run("unknown tool stays a protocol error", func(t *testing.T) {
		before := calls.Load()
		response := decodeCall(t, server, "missing", map[string]any{"arguments": map[string]any{}})
		if calls.Load() != before {
			t.Fatalf("tool calls = %d, want %d", calls.Load(), before)
		}
		if response.Error == nil || response.Error.Code != mcpprotocol.INVALID_PARAMS {
			t.Fatalf("protocol error = %#v, want invalid params", response.Error)
		}
		if response.Result != nil {
			t.Fatalf("unknown tool returned a tool result: %s", response.Result)
		}
	})
}

func callMark(t *testing.T, server *Server, params map[string]any) toolResult {
	t.Helper()
	response := decodeCall(t, server, "mark", params)
	if response.Error != nil {
		t.Fatalf("protocol error = %#v, want a tool result", response.Error)
	}
	var result toolResult
	if err := json.Unmarshal(response.Result, &result); err != nil {
		t.Fatalf("unmarshal tool result: %v raw=%s", err, response.Result)
	}
	return result
}

func decodeCall(t *testing.T, server *Server, name string, params map[string]any) protocolResponse {
	t.Helper()
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": name,
		},
	}
	for key, value := range params {
		body["params"].(map[string]any)[key] = value
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	message := server.server.HandleMessage(context.Background(), payload)
	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	var response protocolResponse
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatalf("unmarshal response: %v raw=%s", err, encoded)
	}
	return response
}

func assertToolError(t *testing.T, result toolResult, want string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("isError = false, result = %#v", result)
	}
	if len(result.Content) != 1 || result.Content[0].Text != want {
		t.Fatalf("tool text = %#v, want %q", result.Content, want)
	}
}

func assertToolText(t *testing.T, result toolResult, prefix string) {
	t.Helper()
	if result.IsError {
		t.Fatalf("isError = true, result = %#v", result)
	}
	if len(result.Content) != 1 || len(result.Content[0].Text) < len(prefix) || result.Content[0].Text[:len(prefix)] != prefix {
		t.Fatalf("tool text = %#v, want prefix %q", result.Content, prefix)
	}
}

type protocolResponse struct {
	Error  *mcpprotocol.JSONRPCErrorDetails `json:"error"`
	Result json.RawMessage                  `json:"result"`
}

type toolResult struct {
	IsError bool `json:"isError"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type countingToolset struct {
	calls *atomic.Int32
}

func (t countingToolset) GetName() string        { return "counting" }
func (t countingToolset) GetDescription() string { return "counting toolset" }
func (t countingToolset) GetTools() []toolset.ServerTool {
	return []toolset.ServerTool{{
		Tool: mcpprotocol.NewTool("mark", mcpprotocol.WithDescription("record one call")),
		Handler: func(_ context.Context, params map[string]any) (string, error) {
			t.calls.Add(1)
			if fail, _ := params["fail"].(bool); fail {
				return "", errors.New("message is required")
			}
			return "ran", nil
		},
		Domain: "example",
	}}
}
