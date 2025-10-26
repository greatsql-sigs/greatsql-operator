package util

import (
	"crypto/rand"
	"encoding/base64"
	"math/big"
)

// 密码生成器
const alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Base64Encode base64 encode
func Base64Encode(data []byte) string {
	encode := base64.StdEncoding.EncodeToString(data)
	return encode
}

// Base64Decode base64 decode
func Base64Decode(data string) ([]byte, error) {
	decode, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	return decode, nil
}

// GeneratePassword 生成包含字母和数字的随机密码
func GeneratePassword(length int) string {
	if length <= 0 {
		return ""
	}
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(alphanum)))

	for i := range length {
		n, _ := rand.Int(rand.Reader, charsetLen)
		result[i] = alphanum[n.Int64()]
	}
	return string(result)
}
