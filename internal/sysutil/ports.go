package sysutil

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"syscall"
)

var ReservedPorts = map[int]bool{
	20: true, 21: true, 22: true, 23: true, 25: true, 53: true, 67: true, 68: true, 69: true,
	80: true, 110: true, 123: true, 137: true, 138: true, 139: true, 143: true, 161: true,
	162: true, 389: true, 443: true, 445: true, 465: true, 587: true, 993: true, 995: true,
	1433: true, 1521: true, 1723: true, 2049: true, 2375: true, 2376: true, 3306: true,
	3389: true, 5432: true, 5900: true, 6379: true, 8080: true, 8443: true, 9200: true,
	9300: true, 11211: true, 27017: true,
}

func ValidatePort(port int) error {
	if port <= 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1025 and 65535")
	}
	return nil
}

func IsPortAvailable(port int) bool {
	// Windows treats a dual-stack wildcard differently from separate family probes.
	dual, err := net.Listen("tcp", net.JoinHostPort("", fmt.Sprint(port)))
	if err != nil {
		return false
	}
	dual.Close()
	// Wildcard and specific-address sockets can coexist on macOS. Check each
	// address separately so another application's listener is never overlooked.
	addresses := []string{"127.0.0.1", "0.0.0.0", "::1", "::"}
	interfaces, _ := net.InterfaceAddrs()
	for _, address := range interfaces {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && !ip.IsLinkLocalUnicast() {
			addresses = append(addresses, ip.String())
		}
	}
	seen := map[string]bool{}
	for _, address := range addresses {
		if seen[address] {
			continue
		}
		seen[address] = true
		network := "tcp6"
		if net.ParseIP(address).To4() != nil {
			network = "tcp4"
		}
		ln, err := net.Listen(network, net.JoinHostPort(address, fmt.Sprint(port)))
		if err != nil {
			if network == "tcp6" && (errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EADDRNOTAVAIL)) {
				continue
			}
			return false
		}
		ln.Close()
	}
	return true
}

func RandomAvailablePort() (int, error) {
	for i := 0; i < 256; i++ {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return 0, err
		}
		port := 20000 + int(binary.BigEndian.Uint64(b[:])%40001)
		if ValidatePort(port) == nil && !ReservedPorts[port] && IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}
