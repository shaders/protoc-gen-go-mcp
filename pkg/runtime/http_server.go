package runtime

import "github.com/mark3labs/mcp-go/server"

// NewStreamableHTTPServer is stateless: a stateful one rejects session ids issued by another replica.
// Opt back into sessions with server.WithSessionIdManager; WithStateLess(false) is a no-op.
func NewStreamableHTTPServer(mcpServer *server.MCPServer, opts ...server.StreamableHTTPOption) *server.StreamableHTTPServer {
	return server.NewStreamableHTTPServer(mcpServer, append([]server.StreamableHTTPOption{server.WithStateLess(true)}, opts...)...)
}
