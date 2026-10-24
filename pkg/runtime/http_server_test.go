package runtime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
)

func TestNewStreamableHTTPServerAcceptsSessionFromAnotherReplica(t *testing.T) {
	newReplica := func() *httptest.Server {
		return httptest.NewServer(NewStreamableHTTPServer(server.NewMCPServer("test", "1.0.0")))
	}
	first, second := newReplica(), newReplica()
	defer first.Close()
	defer second.Close()

	post := func(url, sessionID, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if sessionID != "" {
			req.Header.Set(server.HeaderKeySessionID, sessionID)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
		return resp
	}

	initResp := post(first.URL, "",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	if initResp.StatusCode != http.StatusOK {
		t.Fatalf("initialize: status %d", initResp.StatusCode)
	}
	if id := initResp.Header.Get(server.HeaderKeySessionID); id != "" {
		t.Fatalf("initialize issued session %q: a stateless server must not bind clients to a replica", id)
	}

	// Clients still carry ids issued by another replica (or by a pre-stateless release).
	resp := post(second.URL, "mcp-session-00000000-0000-0000-0000-000000000000", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tools/list with a foreign session id: status %d", resp.StatusCode)
	}
}

func TestNewStreamableHTTPServerKeepsCallerOptions(t *testing.T) {
	srv := httptest.NewServer(NewStreamableHTTPServer(server.NewMCPServer("test", "1.0.0"), server.WithSessionIdManager(&server.InsecureStatefulSessionIdManager{})))
	defer srv.Close()

	resp, err := http.Post(srv.URL, "application/json", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get(server.HeaderKeySessionID) == "" {
		t.Fatal("a caller session id manager must override the stateless default")
	}
}
