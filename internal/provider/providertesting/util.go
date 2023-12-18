package providertesting

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
)

func TestRandomPassword(t *testing.T) string {
	out := make([]byte, 32)

	if _, err := rand.Read(out); err != nil {
		t.Fatal(err)
	}

	str := base64.RawStdEncoding.EncodeToString(out)
	return str[:32]
}
