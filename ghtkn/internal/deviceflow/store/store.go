package store

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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

func New() (*UserCodeStore, error) {
	ucd, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get user cache dir: %w", err)
	}
	return &UserCodeStore{
		dir: getDir(ucd),
	}, nil
}

type UserCodeStore struct {
	dir    string
	logger *log.Logger
}

func (d *UserCodeStore) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *UserCodeStore) Remove(file string) error {
	return os.Remove(file)
}

func (d *UserCodeStore) Write(code *Code) (string, error) {
	if err := os.MkdirAll(d.dir, 0o700); err != nil {
		return "", err
	}
	file := filepath.Join(d.dir, fmt.Sprintf("%d.json", code.Expiry.UnixNano()))
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

func getDir(cacheDir string) string {
	// ${XDG_CACHE_HOME:-$HOME/.cache}/ghtkn/device-codes
	return filepath.Join(cacheDir, "ghtkn", "device-codes")
}

func (d *UserCodeStore) Get(logger *slog.Logger) (string, error) {
	// get files
	files, err := os.ReadDir(d.dir)
	if err != nil {
		return "", err
	}
	// sort files by file name
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	now := time.Now().UnixNano()
	for _, file := range files {
		name := file.Name()
		expiryS, ok := strings.CutSuffix(name, ".json")
		if !ok {
			// Ignore non JSON files
			continue
		}
		expiry, err := strconv.ParseInt(expiryS, 10, 64)
		if err != nil {
			// Ignore invalid expiry
			continue
		}
		if expiry < now {
			// remove expired files
			if err := os.Remove(filepath.Join(d.dir, name)); err != nil {
				return "", err
			}
			continue
		}
		// get expiry
		// remove expired file
		// read oldest file
		// remove oldest file
		// output code

	}
	return "", nil
}
