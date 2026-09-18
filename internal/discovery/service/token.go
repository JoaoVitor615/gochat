package service

import (
	"crypto/rand"
	"fmt"
)

const tokenAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateInviteCode() (string, error) {
	const tokenLength = 9

	raw := make([]byte, tokenLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}

	token := make([]byte, tokenLength)
	for i, b := range raw {
		token[i] = tokenAlphabet[int(b)%len(tokenAlphabet)]
	}

	return fmt.Sprintf("%s-%s-%s", token[:3], token[3:6], token[6:]), nil
}
