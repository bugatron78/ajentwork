package idgen

import (
	"crypto/rand"
	"fmt"
)

const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func NewItemID() (string, error) {
	return newID("W-", 8)
}

func NewEventID() (string, error) {
	return newID("EV-", 12)
}

func NewArtifactID() (string, error) {
	return newID("AR-", 10)
}

func newID(prefix string, length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate id entropy: %w", err)
	}

	result := make([]byte, 0, len(prefix)+length)
	result = append(result, prefix...)
	for _, b := range bytes {
		result = append(result, alphabet[int(b)%len(alphabet)])
	}
	return string(result), nil
}
