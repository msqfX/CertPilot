package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
)

// AES-GCM 加密工具：用于私钥、SMTP 凭证、Token 等敏感字段的落库加密。
// 主密钥来自环境变量（hex 编码的 32 字节），生产环境建议接入 KMS。

var ErrBadKey = errors.New("master key must be 64 hex chars (32 bytes)")

func parseKey(masterKeyHex string) ([]byte, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil || len(key) != 32 {
		return nil, ErrBadKey
	}
	return key, nil
}

func Encrypt(masterKeyHex, plaintext string) (string, error) {
	key, err := parseKey(masterKeyHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(out), nil
}

func Decrypt(masterKeyHex, ciphertextHex string) (string, error) {
	key, err := parseKey(masterKeyHex)
	if err != nil {
		return "", err
	}
	raw, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, data := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	out, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
