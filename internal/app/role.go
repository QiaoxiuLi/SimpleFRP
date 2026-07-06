package app

import (
	"os"
	"strings"

	"github.com/simplefrp/simplefrp/internal/sysutil"
)

type Role string

const (
	RoleServer Role = "server"
	RoleClient Role = "client"
)

func DetectRole() (Role, error) {
	if env := strings.TrimSpace(os.Getenv("SIMPLEFRP_ROLE")); env != "" {
		return Role(env), nil
	}
	b, err := os.ReadFile(sysutil.RolePath())
	if err != nil {
		return RoleClient, nil
	}
	role := Role(strings.TrimSpace(string(b)))
	if role != RoleServer && role != RoleClient {
		return RoleClient, nil
	}
	return role, nil
}

func WriteRole(role Role) error {
	if err := os.MkdirAll(sysutil.ConfigDir(), 0750); err != nil {
		return err
	}
	return os.WriteFile(sysutil.RolePath(), []byte(role+"\n"), 0640)
}
