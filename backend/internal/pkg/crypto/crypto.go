// Package crypto 提供 AES-256-GCM 加解密工具。
// 用于敏感字段（手机号）的加密存储和传输。
// 密文格式: base64(12字节随机nonce + 密文)。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	// ErrCipherTooShort 密文长度不足以包含 nonce。
	ErrCipherTooShort = errors.New("密文太短")
)

// AES256GCM 封装 AES-256-GCM 加解密操作。
// GCM 模式同时提供加密和完整性校验（AEAD），防止密文篡改。
type AES256GCM struct {
	key []byte // 32 字节密钥
}

// NewAES256GCM 创建加解密实例。
// key 不足 32 字节时自动补 0 填充（生产环境应确保恰好 32 字节）。
func NewAES256GCM(key string) (*AES256GCM, error) {
	k := []byte(key)
	if len(k) != 32 {
		padded := make([]byte, 32)
		copy(padded, k)
		k = padded
	}
	return &AES256GCM{key: k}, nil
}

// Encrypt 使用 AES-256-GCM 加密明文，返回 base64 编码的密文。
// 每次加密生成随机 nonce，相同明文产生不同密文。
func (a *AES256GCM) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	// GCM 随机 nonce，大小由 cipher 决定（通常 12 字节）
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	// Seal 将 nonce 拼在密文前: nonce + ciphertext + authTag
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密 base64 编码的 AES-256-GCM 密文。
func (a *AES256GCM) Decrypt(encodedCiphertext string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrCipherTooShort
	}
	// 拆分 nonce 和待解密数据
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
