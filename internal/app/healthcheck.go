package app

import (
	"net"
	"time"

	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type Health struct {
	DashboardPort bool
	ControlPort   bool
	Database      bool
}

func CheckServer() Health {
	return Health{
		DashboardPort: canDial("127.0.0.1:8387"),
		ControlPort:   canDial("127.0.0.1:8388"),
		Database:      sysutil.FileWritable(sysutil.DatabasePath()),
	}
}

func canDial(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
