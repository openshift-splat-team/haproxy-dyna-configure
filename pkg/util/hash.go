package util

import (
	"crypto/sha512"
	"encoding/base64"
)

// GenerateSHA512Hash generates a SHA512 hash of the given byte slice
func GenerateSHA512Hash(byteStr []byte) string {
	hasher := sha512.New()
	hasher.Write(byteStr)
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}
