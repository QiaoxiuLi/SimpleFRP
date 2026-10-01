package protocol

import "github.com/simplefrp/simplefrp/internal/config"

type Message struct {
	Type          string          `json:"type"`
	Admin         bool            `json:"admin,omitempty"`
	ClientID      string          `json:"client_id,omitempty"`
	AuthTimestamp int64           `json:"auth_timestamp,omitempty"`
	Nonce         string          `json:"nonce,omitempty"`
	AuthProof     string          `json:"auth_proof,omitempty"`
	RequestID     string          `json:"request_id,omitempty"`
	TunnelID      int             `json:"tunnel_id,omitempty"`
	LocalPort     int             `json:"local_port,omitempty"`
	PublicPort    int             `json:"public_port,omitempty"`
	Kind          string          `json:"kind,omitempty"`
	Port          int             `json:"port,omitempty"`
	Address       string          `json:"address,omitempty"`
	Tunnels       []config.Tunnel `json:"tunnels,omitempty"`
	OK            bool            `json:"ok,omitempty"`
}

const (
	TypePair         = "pair"
	TypeUnpair       = "unpair"
	TypeAuth         = "auth"
	TypeDataConn     = "data_conn"
	TypeHeartbeat    = "heartbeat"
	TypeOpenStream   = "open_stream"
	TypeCreateTunnel = "create_tunnel"
	TypeDeleteTunnel = "delete_tunnel"
	TypeUpdate       = "update_tunnel"
	TypeConfigure    = "configure_client"
	TypeAck          = "ack"
	TypeStatus       = "status"
	TypePort         = "port"
	TypeMoved        = "server_moved"
)

type TunnelStatus struct {
	ID             int     `json:"id"`
	LocalPort      int     `json:"local_port"`
	PublicPort     int     `json:"public_port"`
	Status         string  `json:"status"`
	BytesPerSecond float64 `json:"bytes_per_second"`
	TotalBytes     int64   `json:"total_bytes"`
}
type Status struct {
	HeartbeatPort int            `json:"heartbeat_port"`
	DashboardPort int            `json:"dashboard_port,omitempty"`
	Tunnels       []TunnelStatus `json:"tunnels"`
	Online        bool           `json:"online"`
}
