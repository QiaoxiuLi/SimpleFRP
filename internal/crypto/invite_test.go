package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestInviteRoundTripAndCorruption(t *testing.T) {
	key, _ := RandomToken(16)
	pin := sha256.Sum256([]byte("fixture"))
	for _, ip := range []string{"203.0.113.8", "2001:db8::8"} {
		want := Invite{IP: ip, Port: 25001, ServerID: "0123456789abcdef", Key: key, Fingerprint: hex.EncodeToString(pin[:])}
		code, err := EncodeInvite(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeInvite(code)
		if err != nil || got != want {
			t.Fatal("invite round trip failed", err)
		}
		for _, bad := range []string{code[:len(code)-1], code + "0", strings.Replace(code, "SF2", "SF1", 1), code[:len(code)-1] + "!", "SF2" + strings.Repeat("0", 141)} {
			if _, err = DecodeInvite(bad); err == nil {
				t.Fatal("invalid invite accepted")
			}
		}
	}
}
