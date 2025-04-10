package mysqlpassword

import (
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// allowedSpecialChars is the set of special characters allowed in the password.
const allowedSpecialChars = "#!~%^*_+-=?{}()<>|.,;"

// validChars contains all valid characters for the password.
var validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" + allowedSpecialChars

// forbiddenFirstChars are not allowed to appear as the first character.
var forbiddenFirstChars = "-_;"

// validatePassword checks whether the generated password meets the MySQL user password requirements.
func validatePassword(g *GomegaWithT, pwd string, expectedLength int) {
	// Check length.
	g.Expect(len(pwd)).To(Equal(expectedLength), "password should be of the requested length")
	// Must contain at least one lowercase letter.
	g.Expect(regexp.MustCompile("[a-z]").MatchString(pwd)).To(BeTrue(), "password must contain at least one lowercase character")
	// Must contain at least one uppercase letter.
	g.Expect(regexp.MustCompile("[A-Z]").MatchString(pwd)).To(BeTrue(), "password must contain at least one uppercase character")
	// Must contain at least one digit.
	g.Expect(regexp.MustCompile("[0-9]").MatchString(pwd)).To(BeTrue(), "password must contain at least one digit")
	// Must contain at least one allowed special character.
	specialPattern := "[" + regexp.QuoteMeta(allowedSpecialChars) + "]"
	g.Expect(regexp.MustCompile(specialPattern).MatchString(pwd)).To(BeTrue(), "password must contain at least one allowed special character")
	// Ensure the password contains only allowed characters.
	for _, ch := range pwd {
		g.Expect(strings.ContainsRune(validChars, ch)).To(BeTrue(), "password contains invalid character: %q", ch)
	}
	// The first character must not be forbidden.
	g.Expect(strings.ContainsAny(string(pwd[0]), forbiddenFirstChars)).To(BeFalse(), "password should not start with a forbidden character")
}

func TestGeneratePassword(t *testing.T) {
	g := NewWithT(t)

	t.Run("invalid length (less than 8) returns error", func(t *testing.T) {
		// Expect an error if the requested length is below 8.
		_, err := generatePassword(7)
		g.Expect(err).ToNot(BeNil(), "expected an error when password length is below 8")
	})

	t.Run("valid password generation meets requirements", func(t *testing.T) {
		reqLength := 12
		pwd, err := generatePassword(reqLength)
		g.Expect(err).To(BeNil(), "did not expect an error for a valid password length")
		validatePassword(g, pwd, reqLength)
	})

	t.Run("generated passwords are random and valid (multiple iterations)", func(t *testing.T) {
		reqLength := 10
		iterations := 10
		for i := 0; i < iterations; i++ {
			pwd, err := generatePassword(reqLength)
			g.Expect(err).To(BeNil(), "did not expect an error for valid password generation")
			validatePassword(g, pwd, reqLength)
		}
	})
}
