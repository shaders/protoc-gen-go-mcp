package main

import (
	"log"
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata"
	"github.com/shaders/protoc-gen-go-mcp/pkg/testdata/gen/go/testdata/testdatamcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 1. Create MCP Server
	mcpServer := server.NewMCPServer(
		"Your Service MCP Server",
		"1.0.0",
	)

	// 2. Connect to your gRPC service
	//nolint:staticcheck
	conn, err := grpc.Dial("localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("Failed to connect to gRPC:", err)
	}
	defer conn.Close()

	// 3. Create gRPC client and register with MCP
	grpcClient := testdata.NewTestServiceClient(conn)
	testdatamcp.ForwardToTestServiceClient(mcpServer, grpcClient)

	// 4. Serve MCP over HTTP
	mcpHandler := server.NewStreamableHTTPServer(mcpServer)
	http.Handle("/mcp", mcpHandler)

	log.Println("MCP server running on http://localhost:8080/mcp")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
