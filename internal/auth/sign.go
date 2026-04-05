package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

func SignRequest(req *http.Request, priv ed25519.PrivateKey, keyID string) error {
	now := time.Now().Unix()
	targetURI := req.URL.String()

	sigBase := buildSignatureBase(req.Method, targetURI, now)

	sig := ed25519.Sign(priv, []byte(sigBase))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	input := fmt.Sprintf(`sig1=("@method" "@target-uri");created=%d;keyid="%s";alg="ed25519"`, now, keyID)
	req.Header.Set("Signature-Input", input)
	req.Header.Set("Signature", fmt.Sprintf("sig1=:%s:", sigB64))

	return nil
}

func buildSignatureBase(method, targetURI string, created int64) string {
	return fmt.Sprintf("\"@method\": %s\n\"@target-uri\": %s\n\"@signature-params\": (\"@method\" \"@target-uri\");created=%d", method, targetURI, created)
}
