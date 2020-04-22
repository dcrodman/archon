package encryption

import (
	"reflect"
	"testing"
)

//func TestPSOCrypt(t *testing.T) {
//	tests := []struct {
//		name    string
//		cryptFn func() *PSOCrypt
//	}{
//		{"BB Crypt", NewBBCrypt},
//		{"PC Crypt", NewPCCrypt},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			crypt := tt.cryptFn()
//			testData := []byte("test data with padding _")
//
//			encBuffer := make([]byte, len(testData))
//			copy(encBuffer, testData)
//			crypt.Encrypt(encBuffer, uint32(len(encBuffer)))
//
//			if reflect.DeepEqual(encBuffer, testData) {
//				t.Fatalf("expected Encrypt() to have encrypted data")
//			}
//
//			decBuffer := make([]byte, len(testData))
//			copy(decBuffer, encBuffer)
//			crypt.Decrypt(decBuffer, uint32(len(decBuffer)))
//
//			if !reflect.DeepEqual(decBuffer, testData) {
//				t.Fatalf("expected Decrypt() to have decrypted to the original string")
//			}
//
//			buffer2 := make([]byte, len(testData))
//			copy(buffer2, testData)
//			tt.cryptFn().Encrypt(buffer2, uint32(len(buffer2)))
//
//			if reflect.DeepEqual(buffer2, encBuffer) {
//				t.Fatalf("expected new cipher to have used a different vector")
//			}
//		})
//	}
//}

func TestPSOCrypt_BB(t *testing.T) {
	crypt := NewBBCrypt()
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
	NewBBCrypt().Encrypt(buffer2, uint32(len(buffer2)))

	if reflect.DeepEqual(buffer2, encBuffer) {
		t.Fatalf("expected new cipher to have used a different vector")
	}
}

func TestPSOCrypt_PC(t *testing.T) {
	vector := createKey(4)

	// PCCrypt was presumably designed specifically for client/server interaction
	// and attempting to encrypt and subsequently decrypt the same block of code
	// will not yield the original string. To test the functionality, two crypt
	// instances are used to mimic how the client and server use this cipher.
	clientCipher, _ := newPCCipher(vector)
	clientCrypt := &PSOCrypt{Vector: vector, cipher: clientCipher}

	serverCipher, _ := newPCCipher(vector)
	serverCrypt := &PSOCrypt{Vector: vector, cipher: serverCipher}

	testData := []byte("test data with padding _")

	encBuffer := make([]byte, len(testData))
	copy(encBuffer, testData)
	clientCrypt.Encrypt(encBuffer, uint32(len(encBuffer)))

	if reflect.DeepEqual(encBuffer, testData) {
		t.Fatalf("expected Encrypt() to have encrypted data")
	}

	decBuffer := make([]byte, len(testData))
	copy(decBuffer, encBuffer)
	serverCrypt.Decrypt(decBuffer, uint32(len(decBuffer)))

	if !reflect.DeepEqual(decBuffer, testData) {
		t.Fatalf("expected Decrypt() to have decrypted to the original string")
	}

	buffer2 := make([]byte, len(testData))
	copy(buffer2, testData)
	NewPCCrypt().Encrypt(buffer2, uint32(len(buffer2)))

	if reflect.DeepEqual(buffer2, encBuffer) {
		t.Fatalf("expected new cipher to have used a different vector")
	}
}
