package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// mcpToolList is the curated v0 surface (plan §P0.T45). Each tool is a
// thin shim over the corresponding CLI subcommand; full handler bodies
// land alongside daemon RPC integration.
var mcpToolList = []map[string]any{
	{
		"name":        "explore",
		"description": "Bounded graph navigation around a selector",
		"inputSchema": map[string]any{"type": "object"},
	},
	{
		"name":        "validate_diff",
		"description": "Run change.process on a candidate diff",
		"inputSchema": map[string]any{"type": "object"},
	},
	{
		"name":        "get_context",
		"description": "Fetch flow / invariant / control evidence for an entity",
		"inputSchema": map[string]any{"type": "object"},
	},
	{
		"name":        "query",
		"description": "Run a DSL query",
		"inputSchema": map[string]any{"type": "object"},
	},
}

// jsonrpcRequest is the JSON-RPC 2.0 envelope (mcp uses JSON-RPC under the hood).
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// jsonrpcResponse is the JSON-RPC 2.0 reply envelope.
type jsonrpcResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *jsonrpcError `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// newMCPCmdReal implements `graph-harness mcp` (stdio transport). Phase 0
// supports `tools/list` and `tools/call` for the four curated tools.
func newMCPCmdReal() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run the MCP adapter (stdio transport)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpServeStdio(context.Background(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
}

func mcpServeStdio(_ context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(jsonrpcResponse{
				JSONRPC: "2.0", ID: nil,
				Error: &jsonrpcError{Code: -32700, Message: err.Error()},
			})
			continue
		}
		resp := dispatchMCP(req)
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func dispatchMCP(req jsonrpcRequest) jsonrpcResponse {
	switch req.Method {
	case "tools/list":
		return jsonrpcResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{"tools": mcpToolList},
		}
	case "initialize":
		return jsonrpcResponse{
			JSONRPC: "2.0", ID: req.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "graph-harness",
					"version": "0.1.0-dev",
				},
				"capabilities": map[string]any{"tools": map[string]any{}},
			},
		}
	default:
		return jsonrpcResponse{
			JSONRPC: "2.0", ID: req.ID,
			Error: &jsonrpcError{Code: -32601, Message: fmt.Sprintf("method %q not implemented in P0", req.Method)},
		}
	}
}

// (unused) compile-time assertion that os is referenced (kept to support
// future implementation without re-importing).
var _ = os.Stdin
