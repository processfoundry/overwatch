package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/christianmscott/overwatch/pkg/spec"
)

const maxClockSkew = 5 * time.Minute

func VerifyRequest(r *http.Request, keys []spec.PublicKeyEntry) error {
	input := r.Header.Get("Signature-Input")
	sigHeader := r.Header.Get("Signature")
	if input == "" || sigHeader == "" {
		return fmt.Errorf("missing signature headers")
	}

	keyID, created, err := parseSignatureInput(input)
	if err != nil {
		return err
	}

	age := math.Abs(float64(time.Now().Unix() - created))
	if age > maxClockSkew.Seconds() {
		return fmt.Errorf("signature expired")
	}

	pubKey, err := findKey(keys, keyID)
	if err != nil {
		return err
	}

	sigBytes, err := parseSignatureValue(sigHeader)
	if err != nil {
		return err
	}

	targetURI := fmt.Sprintf("http://%s%s", r.Host, r.RequestURI)
	sigBase := buildSignatureBase(r.Method, targetURI, created)

	if !ed25519.Verify(pubKey, []byte(sigBase), sigBytes) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

func parseSignatureInput(input string) (keyID string, created int64, err error) {
	// sig1=("@method" "@target-uri");created=1712345678;keyid="abc123";alg="ed25519"
	input = strings.TrimPrefix(input, "sig1=")
	parts := strings.Split(input, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "created=") {
			created, err = strconv.ParseInt(strings.TrimPrefix(p, "created="), 10, 64)
			if err != nil {
				return "", 0, fmt.Errorf("invalid created timestamp: %w", err)
			}
		}
		if strings.HasPrefix(p, "keyid=") {
			keyID = strings.Trim(strings.TrimPrefix(p, "keyid="), `"`)
		}
	}
	if keyID == "" || created == 0 {
		return "", 0, fmt.Errorf("missing keyid or created in Signature-Input")
	}
	return keyID, created, nil
}

func parseSignatureValue(header string) ([]byte, error) {
	// sig1=:base64data:
	header = strings.TrimPrefix(header, "sig1=")
	header = strings.Trim(header, ":")
	return base64.StdEncoding.DecodeString(header)
}

func findKey(keys []spec.PublicKeyEntry, keyID string) (ed25519.PublicKey, error) {
	for _, k := range keys {
		if k.KeyID == keyID {
			raw, err := base64.StdEncoding.DecodeString(k.PublicKey)
			if err != nil {
				return nil, fmt.Errorf("decoding public key %s: %w", keyID, err)
			}
			return ed25519.PublicKey(raw), nil
		}
	}
	return nil, fmt.Errorf("unknown key: %s", keyID)
}
