package candidates

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const MaxResumeBytes int64 = 10 << 20

var (
	ErrResumeTooLarge = errors.New("resume too large")
	ErrResumeType     = errors.New("resume must be a PDF")
)

type Service struct {
	store Store
	files FileStore
}

func NewService(store Store, files FileStore) *Service { return &Service{store: store, files: files} }

func (s *Service) Store() Store { return s.store }

func (s *Service) UploadResume(ctx context.Context, candidateID, filename, contentType string, source io.Reader) (Resume, error) {
	contents, err := io.ReadAll(io.LimitReader(source, MaxResumeBytes+1))
	if err != nil {
		return Resume{}, err
	}
	if int64(len(contents)) > MaxResumeBytes {
		return Resume{}, ErrResumeTooLarge
	}
	if len(contents) < 5 || !bytes.Equal(contents[:5], []byte("%PDF-")) {
		return Resume{}, ErrResumeType
	}
	detected := http.DetectContentType(contents[:min(len(contents), 512)])
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType != "application/pdf" || detected != "application/pdf" {
		return Resume{}, ErrResumeType
	}
	key, err := newStorageKey()
	if err != nil {
		return Resume{}, err
	}
	if err := s.files.Put(ctx, key, bytes.NewReader(contents)); err != nil {
		return Resume{}, fmt.Errorf("store resume: %w", err)
	}
	resume := Resume{OriginalFilename: safeFilename(filename), ContentType: "application/pdf", SizeBytes: int64(len(contents)), Status: "ready", storageKey: key}
	created, _, err := s.store.ReplaceResume(ctx, candidateID, resume)
	if err != nil {
		_ = s.files.Delete(ctx, key)
		return Resume{}, err
	}
	return created, nil
}

func (s *Service) OpenOwnResume(ctx context.Context, candidateID string) (Resume, io.ReadCloser, error) {
	resume, err := s.store.GetActiveResume(ctx, candidateID)
	if err != nil {
		return Resume{}, nil, err
	}
	file, err := s.files.Open(ctx, resume.storageKey)
	return resume, file, err
}

func (s *Service) DeleteOwnResume(ctx context.Context, candidateID string) error {
	resume, canDelete, err := s.store.DeleteResume(ctx, candidateID)
	if err != nil {
		return err
	}
	if canDelete && resume != nil {
		return s.files.Delete(ctx, resume.storageKey)
	}
	return nil
}

func (s *Service) OpenEmployerResume(ctx context.Context, employerID, applicationID string) (Resume, io.ReadCloser, error) {
	resume, err := s.store.GetEmployerApplicationResume(ctx, employerID, applicationID)
	if err != nil {
		return Resume{}, nil, err
	}
	file, err := s.files.Open(ctx, resume.storageKey)
	return resume, file, err
}

func newStorageKey() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer) + ".pdf", nil
}

func safeFilename(value string) string {
	value = filepath.Base(strings.ReplaceAll(value, "\\", "/"))
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 || r == '"' || r == ';' {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	if value == "" {
		return "resume.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(value), ".pdf") {
		value += ".pdf"
	}
	if len(value) > 180 {
		value = value[:176] + ".pdf"
	}
	return value
}
