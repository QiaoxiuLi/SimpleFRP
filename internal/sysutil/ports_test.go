package sysutil

import (
	"net"
	"testing"
)

func TestAvailablePortRejectsExistingLoopbackAndWildcardListeners(t *testing.T) {
	for _, address := range []string{"127.0.0.1:0", "0.0.0.0:0", "[::1]:0", "[::]:0"} {
		t.Run(address, func(t *testing.T) {
			ln, err := net.Listen("tcp", address)
			if err != nil {
				if address == "[::1]:0" || address == "[::]:0" {
					t.Skip("IPv6 loopback unavailable")
				}
				t.Fatal(err)
			}
			port := ln.Addr().(*net.TCPAddr).Port
			if IsPortAvailable(port) {
				ln.Close()
				t.Fatal("occupied port was reported available")
			}
			ln.Close()
			if !IsPortAvailable(port) {
				t.Fatal("released port was reported occupied")
			}
		})
	}
}
