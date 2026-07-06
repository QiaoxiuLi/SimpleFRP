package sysutil

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
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
	if ReservedPorts[port] {
		return fmt.Errorf("port is reserved")
	}
	return nil
}

func IsPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func RandomAvailablePort() (int, error) {
	for i := 0; i < 256; i++ {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			return 0, err
		}
		port := 20000 + int(binary.BigEndian.Uint64(b[:])%40001)
		if ValidatePort(port) == nil && IsPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}
