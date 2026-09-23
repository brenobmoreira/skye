package web

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const tokenBytes = 32

func validToken(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == tokenBytes
}

func LoadOrCreateToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if tok := strings.TrimSpace(string(data)); err == nil && validToken(tok) {
		return tok, os.Chmod(path, 0o600)
	}
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(raw)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(tok+"\n"), 0o600); err != nil {
		return "", err
	}
	return tok, os.Chmod(path, 0o600)
}
