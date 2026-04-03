package util

import (
	"crypto/rand"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// 生成32字节密钥
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{"简单文本", "hello world"},
		{"空字符串", ""},
		{"JSON数据", `{"key": "value", "number": 123}`},
		{"特殊字符", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"中文文本", "这是一段中文测试文本"},
		{"长文本", "This is a very long text that should be encrypted and decrypted correctly without any issues. It contains multiple sentences and various characters."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := Encrypt(tt.plaintext, key)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if ciphertext == tt.plaintext {
				t.Error("Encrypt() returned plaintext")
			}

			decrypted, err := Decrypt(ciphertext, key)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := "same plaintext"
	ciphertext1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ciphertext2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() should produce different ciphertext for same plaintext (due to random nonce)")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	plaintext := "secret message"
	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("Decrypt() should fail with wrong key")
	}
}

func TestEncryptWithInvalidKey(t *testing.T) {
	tests := []struct {
		name string
		key  []byte
	}{
		{"空密钥", []byte{}},
		{"16字节密钥", make([]byte, 16)},
		{"24字节密钥", make([]byte, 24)},
		{"31字节密钥", make([]byte, 31)},
		{"33字节密钥", make([]byte, 33)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Encrypt("test", tt.key)
			if err != ErrInvalidKey {
				t.Errorf("Encrypt() error = %v, want %v", err, ErrInvalidKey)
			}
		})
	}
}

func TestDecryptWithInvalidKey(t *testing.T) {
	// 首先用有效密钥加密
	validKey := make([]byte, 32)
	rand.Read(validKey)
	ciphertext, _ := Encrypt("test", validKey)

	tests := []struct {
		name string
		key  []byte
	}{
		{"空密钥", []byte{}},
		{"16字节密钥", make([]byte, 16)},
		{"24字节密钥", make([]byte, 24)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(ciphertext, tt.key)
			if err != ErrInvalidKey {
				t.Errorf("Decrypt() error = %v, want %v", err, ErrInvalidKey)
			}
		})
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"无效Base64", "not-valid-base64!!!"},
		{"空字符串", ""},
		{"太短", "YWJj"}, // "abc" in base64, too short to be valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext, key)
			if err == nil {
				t.Error("Decrypt() should fail with invalid ciphertext")
			}
		})
	}
}

func TestDecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	plaintext := "original message"
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	// 修改密文（篡改）
	if len(ciphertext) > 10 {
		tampered := ciphertext[:len(ciphertext)-5] + "XXXXX"
		_, err = Decrypt(tampered, key)
		if err == nil {
			t.Error("Decrypt() should fail with tampered ciphertext")
		}
	}
}
