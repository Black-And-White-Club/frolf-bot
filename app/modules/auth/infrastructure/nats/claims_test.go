package authnats

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAuthorizationResponsePayload_JSON(t *testing.T) {
	audience := "test-audience"
	issuerAccount := "test-issuer-account"
	userJWT := "test-jwt"
	errMsg := "test-error"

	subject := "test-subject"
	claims := NewAuthorizationResponseClaims(audience, subject, issuerAccount, userJWT, errMsg)

	data, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("failed to marshal claims: %v", err)
	}

	jsonStr := string(data)

	// Check that "account" is present (replacing issuer_account)
	if !strings.Contains(jsonStr, `"account":"test-issuer-account"`) {
		t.Errorf("expected JSON to contain 'account' field with value 'test-issuer-account', got: %s", jsonStr)
	}

	// Check that "issuer_account" is NOT present in the NATS payload part
	// Note: NATS standard claims might use issuer_account in some places, but our change was for AuthorizationResponsePayload
	// We should be careful. The AuthorizationResponsePayload is inside "nats" field.
	// Let's verify the structure more robustly.

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	natsPayload, ok := decoded["nats"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'nats' field to be a map, got %T", decoded["nats"])
	}

	if _, ok := natsPayload["account"]; !ok {
		t.Error("expected 'nats' payload to have 'account' field")
	}
	if _, ok := natsPayload["issuer_account"]; ok {
		t.Error("expected 'nats' payload to NOT have 'issuer_account' field")
	}
}
