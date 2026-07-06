package crypto

import "encoding/base64"

func RandomPasswordKey(password, salt string) string {
	return base64.RawStdEncoding.EncodeToString(DeriveSessionKey(password, salt))
}
