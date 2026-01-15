package auth

import (
	"encoding/base64"
	"testing"
)

func TestIssueAndVerifyAPIKey(t *testing.T) {
	secret := []byte("my-secret-key")
	clientID := "test-client"

	// 1. Issue Key
	apiKey := IssueAPIKey(clientID, secret)

	// 2. Verify Valid Key
	valid, extractedID, err := VerifyAPIKey(apiKey, secret)
	if !valid || err != nil {
		t.Fatalf("Expected key to be valid, got valid=%v, err=%v", valid, err)
	}
	if extractedID != clientID {
		t.Errorf("Expected clientID %s, got %s", clientID, extractedID)
	}

	// 3. Verify Invalid Key (Wrong Secret)
	wrongSecret := []byte("wrong-secret")
	valid, _, err = VerifyAPIKey(apiKey, wrongSecret)
	if valid || err == nil {
		t.Error("Expected failure with wrong secret, got success")
	}

	// 4. Verify Malformed Key
	valid, _, err = VerifyAPIKey("just-some-string", secret)
	if valid || err == nil {
		t.Error("Expected failure with malformed key, got success")
	}

	// 5. Verify Tampered Signature
	tamperedKey := apiKey + "tampered"
	valid, _, err = VerifyAPIKey(tamperedKey, secret)
	if valid || err == nil {
		t.Error("Expected failure with tampered key, got success")
	}

	// 6. Verify forged signature
	// Manually create a signature that looks correct base64 but is wrong value
	forged := clientID + "." + base64.RawURLEncoding.EncodeToString([]byte("fake-sig"))
	valid, _, err = VerifyAPIKey(forged, secret)
	if valid || err == nil {
		t.Error("Expected failure with forged key, got success")
	}
}
