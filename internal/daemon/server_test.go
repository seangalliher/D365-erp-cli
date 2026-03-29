package daemon

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"
)

func TestServer_PingIntegration(t *testing.T) {
	t.Parallel()

	srv, addr := startTestServer(t, echoHandler())

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Send ping
	req := &Request{ID: "test-1", Command: CmdPing}
	sendJSON(t, conn, req)

	resp := readJSON(t, conn)
	if !resp.Success {
		t.Errorf("expected ping to succeed, got error: %v", resp.Error)
	}

	var ping PingResponse
	if err := json.Unmarshal(resp.Data, &ping); err != nil {
		t.Fatalf("unmarshal ping: %v", err)
	}
	if ping.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", ping.Status)
	}

	_ = srv
}

func TestServer_ShutdownIntegration(t *testing.T) {
	t.Parallel()

	srv, addr := startTestServer(t, echoHandler())

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	req := &Request{ID: "test-2", Command: CmdShutdown}
	sendJSON(t, conn, req)

	resp := readJSON(t, conn)
	if !resp.Success {
		t.Errorf("expected shutdown to succeed")
	}

	// Give server time to stop
	time.Sleep(200 * time.Millisecond)
	_ = srv
}

func TestServer_CommandHandler(t *testing.T) {
	t.Parallel()

	handler := HandlerFunc(func(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
		result := map[string]string{"received": command}
		data, _ := json.Marshal(result)
		return data, nil
	})

	_, addr := startTestServer(t, handler)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	args, _ := json.Marshal(&FormOpenArgs{MenuItemName: "CustTable", MenuItemType: "Display"})
	req := &Request{ID: "test-3", Command: CmdFormOpen, Args: args}
	sendJSON(t, conn, req)

	resp := readJSON(t, conn)
	if !resp.Success {
		t.Errorf("expected success, got error: %v", resp.Error)
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result["received"] != CmdFormOpen {
		t.Errorf("expected received=%q, got %q", CmdFormOpen, result["received"])
	}
}

func TestServer_CommandError(t *testing.T) {
	t.Parallel()

	handler := HandlerFunc(func(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
		return nil, &json.SyntaxError{}
	})

	_, addr := startTestServer(t, handler)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	req := &Request{ID: "test-4", Command: "bad.command"}
	sendJSON(t, conn, req)

	resp := readJSON(t, conn)
	if resp.Success {
		t.Error("expected failure for bad command")
	}
	if resp.Error == nil {
		t.Fatal("expected error detail")
	}
	if resp.Error.Code != "COMMAND_ERROR" {
		t.Errorf("expected code COMMAND_ERROR, got %q", resp.Error.Code)
	}
}

func TestServer_InvalidJSON(t *testing.T) {
	t.Parallel()

	_, addr := startTestServer(t, echoHandler())

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Send invalid JSON
	_, _ = conn.Write([]byte("not json\n"))

	resp := readJSON(t, conn)
	if resp.Success {
		t.Error("expected failure for invalid JSON")
	}
	if resp.Error == nil {
		t.Fatal("expected error detail")
	}
	if resp.Error.Code != "INVALID_REQUEST" {
		t.Errorf("expected code INVALID_REQUEST, got %q", resp.Error.Code)
	}
}

func TestServer_MultipleRequests(t *testing.T) {
	t.Parallel()

	_, addr := startTestServer(t, echoHandler())

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Send multiple pings on the same connection
	for i := 0; i < 3; i++ {
		req := &Request{ID: "multi-" + json.Number(rune('0'+i)).String(), Command: CmdPing}
		sendJSON(t, conn, req)
		resp := readJSON(t, conn)
		if !resp.Success {
			t.Errorf("ping %d failed", i)
		}
	}
}

func TestServer_FormStateTracking(t *testing.T) {
	t.Parallel()

	handler := HandlerFunc(func(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
		result := map[string]string{"ok": "true"}
		data, _ := json.Marshal(result)
		return data, nil
	})

	_, addr := startTestServer(t, handler)

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Open form
	sendJSON(t, conn, &Request{ID: "t1", Command: CmdFormOpen})
	resp := readJSON(t, conn)
	if !resp.Success {
		t.Fatal("form open failed")
	}

	// Check ping shows form open
	sendJSON(t, conn, &Request{ID: "t2", Command: CmdPing})
	resp = readJSON(t, conn)
	var ping PingResponse
	_ = json.Unmarshal(resp.Data, &ping)
	if !ping.FormOpen {
		t.Error("expected form_open=true after form.open")
	}

	// Close form
	sendJSON(t, conn, &Request{ID: "t3", Command: CmdFormClose})
	resp = readJSON(t, conn)
	if !resp.Success {
		t.Fatal("form close failed")
	}

	// Check ping shows form closed
	sendJSON(t, conn, &Request{ID: "t4", Command: CmdPing})
	resp = readJSON(t, conn)
	_ = json.Unmarshal(resp.Data, &ping)
	if ping.FormOpen {
		t.Error("expected form_open=false after form.close")
	}
}

// Test helpers

func echoHandler() Handler {
	return HandlerFunc(func(ctx context.Context, command string, args json.RawMessage) (json.RawMessage, error) {
		result := map[string]string{"command": command}
		data, _ := json.Marshal(result)
		return data, nil
	})
}

func startTestServer(t *testing.T, handler Handler) (*Server, string) {
	t.Helper()

	// Use a random TCP port for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	addr := listener.Addr().String()

	srv := NewServer(handler, ServerConfig{
		IdleTimeout: 30 * time.Second,
	})
	srv.listener = listener

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Run accept loop
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go srv.handleConn(ctx, conn)
		}
	}()

	t.Cleanup(func() {
		cancel()
		_ = listener.Close()
	})

	return srv, addr
}

func sendJSON(t *testing.T, conn net.Conn, v interface{}) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(append(data, '\n')); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readJSON(t *testing.T, conn net.Conn) *Response {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 1024*64)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(buf[:n], &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, string(buf[:n]))
	}
	return &resp
}
