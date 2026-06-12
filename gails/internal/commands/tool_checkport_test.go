package commands

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestIsPortOpen_Open(t *testing.T) {
	// Start a listener on a random local port, then assert isPortOpen
	// returns true for that port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	if !isPortOpen(host, port) {
		t.Errorf("isPortOpen(%s, %d) = false, want true (listener is active)", host, port)
	}
}

func TestIsPortOpen_Closed(t *testing.T) {
	// Find an unused port, close the listener, then assert isPortOpen
	// returns false.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	_ = ln.Close() // immediately free the port

	// Give the OS a moment to release the port.
	time.Sleep(50 * time.Millisecond)

	if isPortOpen(host, port) {
		t.Errorf("isPortOpen(%s, %d) = true, want false (port was closed)", host, port)
	}
}

func TestToolCheckPort_RequiresPort(t *testing.T) {
	err := ToolCheckPort(&ToolCheckPortOptions{Host: "127.0.0.1", Port: 0})
	if err == nil {
		t.Fatal("expected error for port=0, got nil")
	}
}

func TestToolCheckPort_OpenPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)

	if err := ToolCheckPort(&ToolCheckPortOptions{Host: "127.0.0.1", Port: port}); err != nil {
		t.Errorf("ToolCheckPort for open port: %v", err)
	}
}

func TestToolCheckPort_ClosedPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	port, _ := strconv.Atoi(portStr)
	_ = ln.Close()
	time.Sleep(50 * time.Millisecond)

	err = ToolCheckPort(&ToolCheckPortOptions{Host: "127.0.0.1", Port: port})
	if err == nil {
		t.Fatal("expected error for closed port, got nil")
	}
}
