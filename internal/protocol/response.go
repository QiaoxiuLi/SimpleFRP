package protocol

type Response struct {
	OK         bool   `json:"ok"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	TunnelID   int    `json:"tunnel_id,omitempty"`
	LocalPort  int    `json:"local_port,omitempty"`
	PublicPort int    `json:"public_port,omitempty"`
}
