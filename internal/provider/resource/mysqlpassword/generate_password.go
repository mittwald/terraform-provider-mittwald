package mysqlpassword

import (
	crypto_rand "crypto/rand"
	"fmt"
	"math/big"
	math_rand "math/rand"
	"strings"
)

func generatePassword(length int) (string, error) {
	// Ensure the minimum length is 8
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters, %d given", length)
	}

	// Character sets
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	specialChars := "#!~%^*_+-=?{}()<>|.,;"

	// Ensure the password does not start with any of these characters
	disallowedStart := "-_;"

	// Combine all character sets
	allChars := lowercase + uppercase + digits + specialChars

	// Function to generate a random character from a given set
	randomChar := func(set string) (byte, error) {
		n := len(set)
		index, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, err
		}
		return set[index.Int64()], nil
	}

	// Generate the initial required characters
	password := []byte{}

	// Ensure at least one of each required character type is included
	char, err := randomChar(lowercase)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(uppercase)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(digits)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(specialChars)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	// Fill up the remaining characters randomly from all available characters
	for len(password) < length {
		char, err := randomChar(allChars)
		if err != nil {
			return "", err
		}
		password = append(password, char)
	}

	// Ensure the password doesn't start with disallowed characters
	for strings.ContainsAny(string(password[0]), disallowedStart) {
		password[0], _ = randomChar(allChars)
	}

	// Shuffle the password but preserve the first character
	shuffledPassword := shuffleBytes(password[1:])
	// Combine the first character with the shuffled rest
	finalPassword := append([]byte{password[0]}, shuffledPassword...)

	return string(finalPassword), nil
}

// Helper function to shuffle a slice of bytes.
func shuffleBytes(input []byte) []byte {
	n := len(input)
	output := make([]byte, n)
	perm := math_rand.Perm(n)

	for i, v := range perm {
		output[v] = input[i]
	}

	return output
}
