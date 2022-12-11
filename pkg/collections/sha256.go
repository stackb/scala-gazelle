package collections

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Sha256 computes the sha256 of the given reader
func Sha256(in io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, in); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// FileSha256 computes the sha256 hash of a file
func FileSha256(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return Sha256(f)
}
