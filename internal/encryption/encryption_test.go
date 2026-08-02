package encryption

import (
	"reflect"
	"testing"
)

func TestPSOCrypt_BB(t *testing.T) {
	crypt := newBlowfishCipher(createKey(48))
	testData := []byte("test data with padding _")

	encBuffer := make([]byte, len(testData))
	copy(encBuffer, testData)
	crypt.Encrypt(encBuffer, uint32(len(encBuffer)))

	if reflect.DeepEqual(encBuffer, testData) {
		t.Fatalf("expected Encrypt() to have encrypted data")
	}

	decBuffer := make([]byte, len(testData))
	copy(decBuffer, encBuffer)
	crypt.Decrypt(decBuffer, uint32(len(decBuffer)))

	if !reflect.DeepEqual(decBuffer, testData) {
		t.Fatalf("expected Decrypt() to have decrypted to the original string")
	}

	buffer2 := make([]byte, len(testData))
	copy(buffer2, testData)
	newBlowfishCipher(createKey(48)).Encrypt(buffer2, uint32(len(buffer2)))

	if reflect.DeepEqual(buffer2, encBuffer) {
		t.Fatalf("expected new cipher to have used a different vector")
	}
}

func TestPSOCrypt_PC(t *testing.T) {
	// PCCrypt was presumably designed specifically for client/server interaction
	// and attempting to encrypt and subsequently decrypt the same block of code
	// will not yield the original string. To test the functionality, two crypt
	// instances are used to mimic how the client and server use this cipher.
	cryptSession := NewPCCryptoSession()

	testData := []byte("test data with padding _")

	encBuffer := make([]byte, len(testData))
	copy(encBuffer, testData)
	cryptSession.Encrypt(encBuffer, uint32(len(encBuffer)))

	if reflect.DeepEqual(encBuffer, testData) {
		t.Fatalf("expected Encrypt() to have encrypted data")
	}

	decBuffer := make([]byte, len(testData))
	copy(decBuffer, encBuffer)
	cryptSession.Decrypt(decBuffer, uint32(len(decBuffer)))

	if !reflect.DeepEqual(decBuffer, testData) {
		t.Fatalf("expected Decrypt() to have decrypted to the original string")
	}

	buffer2 := make([]byte, len(testData))
	copy(buffer2, testData)
	NewPCCryptoSession().Encrypt(buffer2, uint32(len(buffer2)))

	if reflect.DeepEqual(buffer2, encBuffer) {
		t.Fatalf("expected new cipher to have used a different vector")
	}
}
