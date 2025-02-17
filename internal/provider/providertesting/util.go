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

	// append a special character because there are questionable requirements on password strength.
	return str[:31] + "_"
}
