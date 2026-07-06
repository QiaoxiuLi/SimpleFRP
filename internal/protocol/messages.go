package protocol

type Message struct {
	Type                string         `json:"type"`
	ClientID            string         `json:"client_id,omitempty"`
	AuthKey             string         `json:"auth_key,omitempty"`
	AuthTimestamp       int64          `json:"auth_timestamp,omitempty"`
	AuthProof           string         `json:"auth_proof,omitempty"`
	RequestID           string         `json:"request_id,omitempty"`
	TunnelID            int            `json:"tunnel_id,omitempty"`
	LocalPort           int            `json:"local_port,omitempty"`
	RequestedPublicPort int            `json:"requested_public_port,omitempty"`
	PublicPort          int            `json:"public_port,omitempty"`
	Tunnels             []TunnelStatus `json:"tunnels,omitempty"`
}

const (
	TypeAuth         = "auth"
	TypeDataConn     = "data_conn"
	TypeHeartbeat    = "heartbeat"
	TypeOpenStream   = "open_stream"
	TypeCreateTunnel = "create_tunnel"
	TypeDeleteTunnel = "delete_tunnel"
	TypeUpdatePublic = "update_public_port"
	TypeTraffic      = "traffic_updated"
)
