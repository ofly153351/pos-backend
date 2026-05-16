package warehouse

import (
	"crypto/rand"
	"encoding/hex"
)

func newID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "generated-id"
	}
	return hex.EncodeToString(buf)
}
