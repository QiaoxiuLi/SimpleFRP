package protocol

type TunnelStatus struct {
	TunnelID   int    `json:"tunnel_id"`
	LocalPort  int    `json:"local_port"`
	PublicPort int    `json:"public_port"`
	Status     string `json:"status"`
}
