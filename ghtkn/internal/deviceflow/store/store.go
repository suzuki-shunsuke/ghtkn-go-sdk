package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Code struct {
	Code     string
	Expiry   time.Time
	ClientID string
}

type CodeFile struct {
	Code     string `json:"code"`
	ClientID string `json:"client_id"`
}

func New() *DeviceCodeStore {
	return &DeviceCodeStore{}
}

type DeviceCodeStore struct {
}

func (d *DeviceCodeStore) Remove(file string) error {
	return os.Remove(file)
}

func (d *DeviceCodeStore) Write(code *Code) (string, error) {
	dir, err := d.getDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	file := filepath.Join(dir, fmt.Sprintf("%d.json", code.Expiry.UnixNano()))
	b, err := json.Marshal(&CodeFile{
		Code:     code.Code,
		ClientID: code.ClientID,
	})
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(file, b, 0o600); err == nil {
		return file, nil
	}
	return file, nil
}

func (d *DeviceCodeStore) getDir() (string, error) {
	// ${XDG_CACHE_HOME:-$HOME/.cache}/ghtkn/device-codes
	cd, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cd, "ghtkn", "device-codes"), nil
}
