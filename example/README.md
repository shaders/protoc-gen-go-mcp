# MCP Server Example

This example demonstrates how to create an MCP (Model Context Protocol) server that exposes gRPC services as AI-accessible tools.

## What This Example Demonstrates

### Core Features
- **Automatic Tool Generation**: Converts protobuf service methods into MCP tools
- **AI-Friendly Schemas**: Generates clean JSON schemas that AI models can easily understand
- **OneOf Support**: Handles protobuf oneOf fields with discriminated unions and automatic transformation
- **Field Comments**: Preserves protobuf field comments as descriptions in tool schemas
- **Nested Message Comments**: Supports comments at all levels of message nesting

### Generated Tools
The example uses the `TestService` from testdata which generates these MCP tools:

1. **`testdata_TestService_CreateItem`**: Creates items with oneOf product/service variants
2. **`testdata_TestService_GetItem`**: Retrieves items by ID
3. **`testdata_TestService_ProcessWellKnownTypes`**: Handles Google well-known types

## Running the Example

1. **Start a gRPC server** on `localhost:9090` implementing `TestService`
2. **Run the MCP server**:
   ```bash
   go run main.go
   ```
3. **Connect AI tools** to `http://localhost:8080/mcp`

## Generated Schema Features

- **Clean oneOf structures** instead of complex anyOf/oneOf nesting
- **Type discriminators** with `const` values for clear identification
- **Field descriptions** from protobuf comments
- **Well-known type handling** (Timestamp, Struct, Any, etc.)
- **Nested message support** with recursive comment extraction

## Integration Pattern

This example shows the standard integration pattern:
1. Create MCP server with name/version
2. Connect to existing gRPC service
3. Register gRPC client using generated `ForwardToXXXClient` function
4. Serve MCP over HTTP for AI tool connections

Perfect for exposing existing gRPC APIs to AI assistants! =�
