package frpwrap

import "github.com/simplefrp/simplefrp/internal/sysutil"

type PortAllocator struct{}

func (PortAllocator) Next() (int, error) {
	return sysutil.RandomAvailablePort()
}
