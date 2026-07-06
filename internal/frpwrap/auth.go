package frpwrap

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/simplefrp/simplefrp/internal/protocol"
)

const authSkew = 5 * time.Minute

func signMessage(key string, msg protocol.Message) protocol.Message {
	msg.AuthKey = ""
	msg.AuthTimestamp = time.Now().Unix()
	msg.AuthProof = authProof(key, msg)
	return msg
}

func verifyMessage(key string, msg protocol.Message) bool {
	if key == "" || msg.AuthProof == "" || msg.AuthTimestamp == 0 {
		return false
	}
	sentAt := time.Unix(msg.AuthTimestamp, 0)
	if time.Since(sentAt) > authSkew || time.Until(sentAt) > authSkew {
		return false
	}
	return hmac.Equal([]byte(msg.AuthProof), []byte(authProof(key, msg)))
}

func authProof(key string, msg protocol.Message) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%d|%d|%d|%d|%d",
		msg.Type,
		msg.ClientID,
		msg.RequestID,
		msg.TunnelID,
		msg.LocalPort,
		msg.PublicPort,
		msg.RequestedPublicPort,
		msg.AuthTimestamp,
	)
	return hex.EncodeToString(mac.Sum(nil))
}
