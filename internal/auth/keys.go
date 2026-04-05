package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

func GenerateKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generating ed25519 keypair: %w", err)
	}
	return pub, priv, nil
}

func KeyID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return hex.EncodeToString(h[:12])
}

func SaveKeys(dir string, pub ed25519.PublicKey, priv ed25519.PrivateKey) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating key directory: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: priv.Seed(),
	})
	if err := os.WriteFile(filepath.Join(dir, "key"), privPEM, 0600); err != nil {
		return fmt.Errorf("writing private key: %w", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pub,
	})
	if err := os.WriteFile(filepath.Join(dir, "key.pub"), pubPEM, 0644); err != nil {
		return fmt.Errorf("writing public key: %w", err)
	}

	return nil
}

func LoadPrivateKey(dir string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(filepath.Join(dir, "key"))
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "ED25519 PRIVATE KEY" {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	return ed25519.NewKeyFromSeed(block.Bytes), nil
}

func LoadPublicKey(dir string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(filepath.Join(dir, "key.pub"))
	if err != nil {
		return nil, fmt.Errorf("reading public key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "ED25519 PUBLIC KEY" {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	return ed25519.PublicKey(block.Bytes), nil
}
