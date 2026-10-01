package frpwrap

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/simplefrp/simplefrp/internal/crypto"
	"github.com/simplefrp/simplefrp/internal/protocol"
	"time"
)

func signMessage(key string, msg protocol.Message) protocol.Message {
	msg.AuthTimestamp = time.Now().Unix()
	msg.Nonce, _ = crypto.RandomToken(16)
	msg.AuthProof = authProof(key, msg)
	return msg
}
func authProof(key string, msg protocol.Message) string {
	msg.AuthProof = ""
	b, _ := json.Marshal(msg)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(b)
	return hex.EncodeToString(mac.Sum(nil))
}
func verifyMessage(key string, msg protocol.Message) bool {
	age := time.Since(time.Unix(msg.AuthTimestamp, 0))
	return key != "" && len(msg.Nonce) >= 20 && age < time.Minute && age > -time.Minute && hmac.Equal([]byte(msg.AuthProof), []byte(authProof(key, msg)))
}
