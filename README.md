# `protoc-gen-go-mcp`

[![Test](https://github.com/shaders/protoc-gen-go-mcp/actions/workflows/test.yml/badge.svg)](https://github.com/shaders/protoc-gen-go-mcp/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shaders/protoc-gen-go-mcp)](https://goreportcard.com/report/github.com/shaders/protoc-gen-go-mcp)
[![codecov](https://codecov.io/gh/redpanda-data/protoc-gen-go-mcp/branch/main/graph/badge.svg)](https://codecov.io/gh/redpanda-data/protoc-gen-go-mcp)

**`protoc-gen-go-mcp`** is a [Protocol Buffers](https://protobuf.dev) compiler plugin that generates [Model Context Protocol (MCP)](https://modelcontextprotocol.io) servers for your `gRPC` APIs.

It generates `*.pb.mcp.go` files for each protobuf service, enabling you to delegate handlers directly to gRPC servers or clients. Under the hood, MCP uses JSON Schema for tool inputs—`protoc-gen-go-mcp` auto-generates these schemas from your method input descriptors.

> ⚠️ Currently supports [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) as the MCP server runtime. Future support is planned for official Go SDKs and additional runtimes.

## ✨ Features

- 🚀 **Auto-generates MCP handlers** from your `.proto` services
- 🧠 **AI-Friendly Schemas** - Clean JSON schemas using modern JSON Schema 2020-12 specification
- 📚 **JSON Schema $defs Support** - Message types defined once in `$defs` and referenced via `$ref`
- 🔄 **Recursive Structure Support** - Handles circular and recursive message references with cycle detection
- 🔀 **Advanced OneOf Support** - Handles protobuf oneOf with discriminated unions and automatic transformation
- 💬 **Field Comments as Descriptions** - Preserves protobuf comments in tool schemas (including nested messages)
- 📦 **JSON Schema Generation** for method inputs with proper validation
- 🔄 **Flexible Integration** - Wire up to gRPC servers or clients
- 🧩 **Easy [`buf`](https://buf.build) Integration**
- ⚡  **Well-Known Types** - Proper handling of Google protobuf well-known types
- 🎯 **Gemini Compliant** - Tool names follow Google's restrictions
- 🛡️ **Production Ready** - Robust error handling, no panics, safe type assertions  
  

## 🔧 Usage

### Generate code

Add entry to your `buf.gen.yaml`:
```
...
plugins:
  - local:
      - go
      - run
      - github.com/shaders/protoc-gen-go-mcp/cmd/protoc-gen-go-mcp@latest
    out: ./gen/go
    opt: paths=source_relative
```

You need to generate the standard `*.pb.go` files as well. `protoc-gen-go-mcp` by defaults uses a separate subfolder `{$servicename}mcp`, and imports the `*pb.go` files - similar to connectrpc-go.

After running `buf generate`, you will see a new folder for each package with protobuf Service definitions:

```
tree pkg/testdata/gen/
gen
└── go
    └── testdata
        ├── test_service.pb.go
        ├── testdataconnect/
        │   └── test_service.connect.go
        └── testdatamcp/
            └── test_service.pb.mcp.go
```

### Advanced Schema Generation

#### JSON Schema Structure

`protoc-gen-go-mcp` generates modern JSON schemas with `$defs` for better organization:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "user": {"$ref": "#/$defs/User"},
    "settings": {"$ref": "#/$defs/Settings"}
  },
  "required": ["user"],
  "$defs": {
    "User": {
      "type": "object",
      "properties": {
        "name": {"type": "string", "description": "User's full name"},
        "email": {"type": "string"}
      },
      "required": ["name", "email"]
    },
    "Settings": {
      "type": "object",
      "properties": {
        "theme": {"type": "string"}
      },
      "required": []
    }
  }
}
```

#### OneOf Support with Discriminated Unions

`protoc-gen-go-mcp` generates AI-friendly schemas for protobuf oneOf fields using discriminated unions with `object_type` field:

```protobuf
// Proto definition
message Item {
  oneof item_type {
    Product product = 1;
    Service service = 2;
  }
}
```

Generates schema:
```json
{
  "item_typeOneOfType": {
    "oneOf": [
      {
        "type": "object",
        "properties": {
          "object_type": {"const": "product", "type": "string"},
          "price": {"type": "number"}
        },
        "required": ["object_type"]
      },
      {
        "type": "object",
        "properties": {
          "object_type": {"const": "service", "type": "string"},
          "duration": {"type": "string"}
        },
        "required": ["object_type"]
      }
    ]
  }
}
```

#### Recursive Structure Support

Handles complex recursive structures without stack overflow:

```protobuf
message FilterExpression {
  message Operation {
    repeated FilterExpression operands = 1;  // Recursive reference
  }
  oneof kind {
    Operation operation = 1;
    string value = 2;
  }
}
```

### Wiring up with gRPC client

It is also possible to directly forward MCP tool calls to gRPC clients. Follows gRPC-Gateway pattern.
Connect to gRPC server, then:

```go
testdatamcp.ForwardToTestServiceClient(mcpServer, myGrpcClient)
```

This directly connects the MCP handler to the gRPC client, requiring zero boilerplate.
Each RPC method in your protobuf service becomes an MCP tool.

### Extra properties

It's possible to add extra properties to MCP tools, that are not in the proto. These are written into context.


```go
// Enable URL override with custom field name and description
option := runtime.WithExtraProperties(
    runtime.ExtraProperty{
        Name:        "base_url",
        Description: "Base URL for the API",
        Required:    true,
        ContextKey:  MyURLOverrideKey{},
    },
)

// Use with any generated function
testdatamcp.ForwardToTestServiceClient(mcpServer, client, option)
```


## 🧪 Development & Testing

### Quick Commands

```bash
# Run all tests
task test

# Build the binary
task build

# Install to GOPATH/bin
task install

# Update golden test files
task generate-golden


# View all available commands
task --list
```

### Manual Commands

```bash
# Run tests
go test ./...

# Update golden files
./tools/update-golden.sh
# Or manually for specific packages
go test ./pkg/generator -update-golden

# Build from source
go build -o protoc-gen-go-mcp ./cmd/protoc-gen-go-mcp

# Run integration tests (requires OPENAI_API_KEY)
# Either export OPENAI_API_KEY or add to .env file
export OPENAI_API_KEY="your-api-key"
task integrationtest
```

### Development Workflow

```bash
# Format code
task fmt

# Run linting
task lint

# Generate protobuf files for testdata
task generate
```

### Golden File Testing

The generator uses golden file testing to ensure output consistency. The test structure in `pkg/generator/testdata/` is organized as:

```
testdata/
├── *.proto          # Input proto files (just drop new ones here!)
├── buf.gen.yaml     # Generates into actual/
├── buf.gen.golden.yaml # Generates into golden/
├── actual/          # Current generated output (committed to track changes)
└── golden/          # Expected output (committed as test baseline)
```

**To add new tests:** Simply drop a `.proto` file in `pkg/testdata/proto/testdata/` and run the tests. The framework automatically:
1. Discovers all `.proto` files
2. Generates code using `task generate`
3. Compares with expected output
4. Creates missing golden files on first run

**To update golden files after generator changes:**
```bash
# Update all golden files
task generate-golden

# Or update specific package
go test ./pkg/generator -update-golden
```

## 🏗️ Recent Improvements

### v0.2.0 (Latest)
- **JSON Schema 2020-12**: Modern schema generation with `$defs` and `$ref`
- **Cycle Detection**: Prevents stack overflow with recursive message structures
- **Robust Error Handling**: Replaced panics with proper error propagation
- **Code Quality**: Eliminated code duplication, added safe type assertions
- **Performance**: Optimized memory allocation and string operations
- **Documentation**: Comprehensive documentation for all public APIs

## ⚠️ Limitations
- Tool name mangling for long RPC names: If the full RPC name exceeds 64 characters, the head of the tool name is mangled to fit.
- Streaming RPCs are not yet supported (unary only)
