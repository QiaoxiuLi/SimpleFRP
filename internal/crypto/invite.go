package crypto

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

type Invite struct {
	IP          string
	Port        int
	ServerID    string
	Key         string
	Fingerprint string
}

func EncodeInvite(v Invite) (string, error) {
	ip := net.ParseIP(v.IP)
	if ip == nil || v.Port < 1025 || v.Port > 65535 {
		return "", fmt.Errorf("invalid invitation address")
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	id, err := hex.DecodeString(v.ServerID)
	if err != nil || len(id) != 8 {
		return "", fmt.Errorf("invalid server identity")
	}
	key, err := base64.RawURLEncoding.DecodeString(v.Key)
	if err != nil || len(key) != 16 {
		return "", fmt.Errorf("invalid invitation key")
	}
	pin, err := hex.DecodeString(v.Fingerprint)
	if err != nil || len(pin) != 32 {
		return "", fmt.Errorf("invalid server fingerprint")
	}
	b := []byte{2, byte(len(ip))}
	b = append(b, ip...)
	b = binary.BigEndian.AppendUint16(b, uint16(v.Port))
	b = append(b, id...)
	b = append(b, key...)
	b = append(b, pin...)
	sum := sha256.Sum256(b)
	b = append(b, sum[:4]...)
	n := new(big.Int).SetBytes(b)
	radix := big.NewInt(62)
	rem := new(big.Int)
	var digits []byte
	for n.Sign() > 0 {
		n.QuoRem(n, radix, rem)
		digits = append(digits, alphabet[rem.Int64()])
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return "SF2" + string(digits), nil
}
func DecodeInvite(code string) (Invite, error) {
	fail := fmt.Errorf("invalid connection string; copy the full SF2 string from simplefrp run")
	var v Invite
	if len(code) < 60 || len(code) > 140 || !strings.HasPrefix(code, "SF2") {
		return v, fail
	}
	n := new(big.Int)
	radix := big.NewInt(62)
	for _, c := range code[3:] {
		digit := strings.IndexRune(alphabet, c)
		if digit < 0 {
			return v, fail
		}
		n.Mul(n, radix)
		n.Add(n, big.NewInt(int64(digit)))
	}
	b := n.Bytes()
	if len(b) < 2 || b[0] != 2 || (b[1] != 4 && b[1] != 16) || len(b) != int(b[1])+64 {
		return v, fail
	}
	sum := sha256.Sum256(b[:len(b)-4])
	if !bytes.Equal(sum[:4], b[len(b)-4:]) {
		return v, fail
	}
	pos := 2 + int(b[1])
	v.IP = net.IP(b[2:pos]).String()
	v.Port = int(binary.BigEndian.Uint16(b[pos : pos+2]))
	pos += 2
	v.ServerID = hex.EncodeToString(b[pos : pos+8])
	pos += 8
	v.Key = base64.RawURLEncoding.EncodeToString(b[pos : pos+16])
	pos += 16
	v.Fingerprint = hex.EncodeToString(b[pos : pos+32])
	canonical, err := EncodeInvite(v)
	if err != nil || canonical != code {
		return Invite{}, fail
	}
	return v, nil
}
func (v Invite) Address() string { return net.JoinHostPort(v.IP, strconv.Itoa(v.Port)) }
