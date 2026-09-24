package candidates

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var storageKeyPattern = regexp.MustCompile(`^[a-f0-9]{64}\.pdf$`)

type LocalFileStore struct{ root string }

func NewLocalFileStore(root string) (*LocalFileStore, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve resume storage: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create resume storage: %w", err)
	}
	return &LocalFileStore{root: absolute}, nil
}

func (s *LocalFileStore) Put(ctx context.Context, key string, source io.Reader) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.root, ".upload-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := io.Copy(temp, &contextReader{ctx: ctx, reader: source}); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (s *LocalFileStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *LocalFileStore) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalFileStore) path(key string) (string, error) {
	if !storageKeyPattern.MatchString(key) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.root, key), nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(buffer)
	}
}
