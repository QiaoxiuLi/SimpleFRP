package ux

import "fmt"

type Code string

const (
	InvalidPasswordFormat Code = "INVALID_PASSWORD_FORMAT"
	AuthFailed            Code = "AUTH_FAILED"
	ServerUnreachable     Code = "SERVER_UNREACHABLE"
	ServerPortInUse       Code = "SERVER_PORT_IN_USE"
	LocalPortInUse        Code = "LOCAL_PORT_IN_USE"
	InvalidPort           Code = "INVALID_PORT"
	PortReserved          Code = "PORT_RESERVED"
	TunnelNotFound        Code = "TUNNEL_NOT_FOUND"
	UnknownError          Code = "UNKNOWN_ERROR"
)

func Failure(title, reason, fix, detail string) string {
	out := fmt.Sprintf("%s\n\nReason:\n%s\n\nHow to fix:\n%s", title, reason, fix)
	if detail != "" {
		out += "\n\nTechnical detail:\n" + detail
	}
	return out
}
