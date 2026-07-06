package frpwrap

import "errors"

var (
	ErrTunnelNotFound = errors.New("tunnel not found")
	ErrPortInUse      = errors.New("port is already in use")
)
