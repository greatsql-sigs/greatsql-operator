package utils

import (
	"encoding/base64"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-21 15:29:03
 * @file: encrypt.go
 * @description: encrypt util
 */

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
