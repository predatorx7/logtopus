package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// IssueAPIKey generates a specialized API key for the given clientID signed with the secret.
// Format: clientID.signature
func IssueAPIKey(clientID string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(clientID))
	signature := mac.Sum(nil)
	encodedSig := base64.RawURLEncoding.EncodeToString(signature)
	return fmt.Sprintf("%s.%s", clientID, encodedSig)
}

// VerifyAPIKey verifies the API key against the secret.
// Returns valid bool and the extracted clientID if valid.
func VerifyAPIKey(apiKey string, secret []byte) (bool, string, error) {
	parts := strings.Split(apiKey, ".")
	if len(parts) != 2 {
		return false, "", errors.New("invalid api key format")
	}

	clientID := parts[0]
	providedSig := parts[1]

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(clientID))
	expectedSig := mac.Sum(nil)
	expectedEncodedSig := base64.RawURLEncoding.EncodeToString(expectedSig)

	if hmac.Equal([]byte(providedSig), []byte(expectedEncodedSig)) {
		return true, clientID, nil
	}

	return false, "", errors.New("invalid signature")
}
