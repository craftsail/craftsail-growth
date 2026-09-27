// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"io"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestIsLoopback(t *testing.T) {
	if !isLoopback("127.0.0.1") || !isLoopback("localhost") || !isLoopback("") || !isLoopback("::1") {
		t.Fatal("loopback should be true")
	}
	if isLoopback("0.0.0.0") || isLoopback("10.0.0.1") {
		t.Fatal("public bind is not loopback")
	}
}

func TestRelayLoopbackV6ReachesV4(t *testing.T) {
	backend, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go func() {
		conn, err := backend.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte("board"))
	}()
	port := backend.Addr().(*net.TCPAddr).Port
	stop, err := relayLoopbackV6(port)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	conn, err := net.DialTimeout("tcp6", net.JoinHostPort("::1", strconv.Itoa(port)), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	buf, err := io.ReadAll(conn)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != "board" {
		t.Fatalf("got %q", buf)
	}
}
