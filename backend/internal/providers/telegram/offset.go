package telegram

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func LoadOffset(path string) (int64, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, errors.New("cannot read Telegram offset")
	}
	offset, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil || offset < 0 {
		return 0, errors.New("invalid Telegram offset")
	}
	return offset, nil
}

func SaveOffset(path string, offset int64) error {
	if offset < 0 {
		return errors.New("invalid Telegram offset")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return errors.New("cannot create Telegram offset directory")
	}
	tmp, err := os.CreateTemp(dir, ".telegram-offset-*")
	if err != nil {
		return errors.New("cannot create Telegram offset file")
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return errors.New("cannot secure Telegram offset file")
	}
	if _, err := tmp.WriteString(strconv.FormatInt(offset, 10) + "\n"); err != nil {
		tmp.Close()
		return errors.New("cannot write Telegram offset")
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return errors.New("cannot sync Telegram offset")
	}
	if err := tmp.Close(); err != nil {
		return errors.New("cannot close Telegram offset")
	}
	if err := os.Rename(tmpName, path); err != nil {
		return errors.New("cannot replace Telegram offset")
	}
	return nil
}
