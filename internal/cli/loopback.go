// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
)

// relayLoopbackV6 accepts [::1]:port and copies each connection to 127.0.0.1:port.
// Browsers open localhost on IPv6 first. The HTTP server stays on IPv4.
func relayLoopbackV6(port int) (func(), error) {
	ln, err := net.Listen("tcp6", net.JoinHostPort("::1", strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	var once sync.Once
	stop := func() { once.Do(func() { _ = ln.Close() }) }
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go bridgeLoopback(conn, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		}
	}()
	return stop, nil
}

func bridgeLoopback(client net.Conn, dial string) {
	defer client.Close()
	upstream, err := net.Dial("tcp4", dial)
	if err != nil {
		return
	}
	defer upstream.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(upstream, client)
		_ = upstream.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, upstream)
		_ = client.Close()
	}()
	wg.Wait()
}

func startLoopbackV6(port int) {
	if port <= 0 {
		return
	}
	if _, err := relayLoopbackV6(port); err != nil {
		log.Printf("ipv6 loopback :%d: %v", port, err)
	}
}

func loopbackPort(host string, port int) int {
	if !isLoopback(host) || host == "::1" {
		return 0
	}
	if port <= 0 {
		return 0
	}
	return port
}

func formatListenURL(host string, port int) string {
	if host == "" || host == "0.0.0.0" || host == "::" {
		return fmt.Sprintf("http://127.0.0.1:%d", port)
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}
