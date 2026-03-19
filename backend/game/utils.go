package game

import (
	"math/rand"
	"unicode"
)

// GenerateJoinCode creates a 6-character alphanumeric join code
func GenerateJoinCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

// ValidateJoinCode checks if a join code format is valid
func ValidateJoinCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
