package security

import (
	"os"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	// Setup 32-byte key
	key := "12345678901234567890123456789012"
	os.Setenv("TOKEN_ENCRYPTION_KEY", key)
	defer os.Unsetenv("TOKEN_ENCRYPTION_KEY")

	plaintext := "my-secret-token"

	// Test Encryption
	ciphertext, err := Encrypt(plaintext)
	assert.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	// Test Decryption
	decrypted, err := Decrypt(ciphertext)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)

	// Test with wrong key
	os.Setenv("TOKEN_ENCRYPTION_KEY", "wrong-key-length-123")
	_, err = Encrypt(plaintext)
	assert.Error(t, err)

	os.Setenv("TOKEN_ENCRYPTION_KEY", "another-valid-32-byte-key-!@#$%^&")
	_, err = Decrypt(ciphertext)
	assert.Error(t, err, "Should fail to decrypt with wrong key")
}
