package protocol

import "github.com/simplefrp/simplefrp/internal/config"

type Response struct {
	OK         bool            `json:"ok"`
	Code       string          `json:"code,omitempty"`
	Message    string          `json:"message,omitempty"`
	ClientID   string          `json:"client_id,omitempty"`
	ClientKey  string          `json:"client_key,omitempty"`
	TunnelID   int             `json:"tunnel_id,omitempty"`
	LocalPort  int             `json:"local_port,omitempty"`
	PublicPort int             `json:"public_port,omitempty"`
	Tunnels    []config.Tunnel `json:"tunnels,omitempty"`
	Status     *Status         `json:"status,omitempty"`
}
