package util

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// FileSHA256Hex returns lowercase hex SHA-256 of a file.
func FileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
