package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

func GenerateJoinToken(addr string) (string, error) {
	b := make([]byte, 15)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	return fmt.Sprintf("OVWCH-%s-%s", addr, secret), nil
}

func ParseJoinToken(token string) (addr, secret string, err error) {
	const prefix = "OVWCH-"
	if !strings.HasPrefix(token, prefix) {
		return "", "", fmt.Errorf("invalid join token: missing OVWCH- prefix")
	}
	rest := token[len(prefix):]
	idx := strings.LastIndex(rest, "-")
	if idx < 1 {
		return "", "", fmt.Errorf("invalid join token format")
	}
	addr = rest[:idx]
	secret = rest[idx+1:]
	if addr == "" || secret == "" {
		return "", "", fmt.Errorf("invalid join token: empty address or secret")
	}
	return addr, secret, nil
}
