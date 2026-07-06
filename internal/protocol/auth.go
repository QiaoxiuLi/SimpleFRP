package protocol

type AuthRequest struct {
	ClientID string `json:"client_id"`
	Nonce    string `json:"nonce"`
	Proof    string `json:"proof"`
}
