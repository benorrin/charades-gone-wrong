package game

import (
	"math/rand"
	"time"
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

// GenerateID generates a unique ID using timestamp and random string
func GenerateID() string {
	return time.Now().Format("20060102150405") + "_" + generateRandomString(8)
}

// generateRandomString creates a random alphanumeric string of given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
