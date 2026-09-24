package candidates

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
)

func TestUploadResumeValidatesPDFAndStoresOpaqueKey(t *testing.T) {
	store := &storeStub{}
	files := &fileStoreStub{}
	service := NewService(store, files)
	resume, err := service.UploadResume(context.Background(), "candidate", "../../cv.pdf", "application/pdf", bytes.NewBufferString("%PDF-1.7\nvalid"))
	if err != nil {
		t.Fatal(err)
	}
	if resume.OriginalFilename != "cv.pdf" || resume.SizeBytes == 0 || !storageKeyPattern.MatchString(files.key) {
		t.Fatalf("unexpected resume: %+v key=%q", resume, files.key)
	}
}

func TestUploadResumeRejectsInvalidTypeAndOversize(t *testing.T) {
	service := NewService(&storeStub{}, &fileStoreStub{})
	if _, err := service.UploadResume(context.Background(), "candidate", "cv.pdf", "application/pdf", bytes.NewBufferString("not a pdf")); !errors.Is(err, ErrResumeType) {
		t.Fatalf("expected type error, got %v", err)
	}
	large := io.LimitReader(zeroReader{}, MaxResumeBytes+1)
	if _, err := service.UploadResume(context.Background(), "candidate", "cv.pdf", "application/pdf", large); !errors.Is(err, ErrResumeTooLarge) {
		t.Fatalf("expected size error, got %v", err)
	}
}

func TestUploadResumeRemovesNewFileWhenMetadataFails(t *testing.T) {
	store := &storeStub{err: errors.New("database failed")}
	files := &fileStoreStub{}
	service := NewService(store, files)
	_, err := service.UploadResume(context.Background(), "candidate", "cv.pdf", "application/pdf", bytes.NewBufferString("%PDF-1.7\nvalid"))
	if err == nil || !files.deleted {
		t.Fatalf("expected metadata failure cleanup, err=%v deleted=%v", err, files.deleted)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

type fileStoreStub struct {
	key     string
	deleted bool
}

func (s *fileStoreStub) Put(_ context.Context, key string, _ io.Reader) error {
	s.key = key
	return nil
}
func (s *fileStoreStub) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}
func (s *fileStoreStub) Delete(context.Context, string) error { s.deleted = true; return nil }

type storeStub struct{ err error }

func (s *storeStub) GetProfile(context.Context, string) (Profile, error) { return Profile{}, s.err }
func (s *storeStub) UpdateProfile(context.Context, string, Profile) (Profile, error) {
	return Profile{}, s.err
}
func (s *storeStub) GetActiveResume(context.Context, string) (Resume, error) { return Resume{}, s.err }
func (s *storeStub) ReplaceResume(_ context.Context, _ string, r Resume) (Resume, *Resume, error) {
	r.ID = "resume"
	return r, nil, s.err
}
func (s *storeStub) DeleteResume(context.Context, string) (*Resume, bool, error) {
	return nil, false, s.err
}
func (s *storeStub) SaveJob(context.Context, string, string) error             { return s.err }
func (s *storeStub) UnsaveJob(context.Context, string, string) error           { return s.err }
func (s *storeStub) ListSavedJobs(context.Context, string) ([]SavedJob, error) { return nil, s.err }
func (s *storeStub) CreateApplication(context.Context, string, string, string, string) (Application, error) {
	return Application{}, s.err
}
func (s *storeStub) ListCandidateApplications(context.Context, string) ([]Application, error) {
	return nil, s.err
}
func (s *storeStub) WithdrawApplication(context.Context, string, string) (Application, error) {
	return Application{}, s.err
}
func (s *storeStub) ListEmployerApplications(context.Context, string, string) ([]Application, error) {
	return nil, s.err
}
func (s *storeStub) UpdateEmployerApplication(context.Context, string, string, string) (Application, error) {
	return Application{}, s.err
}
func (s *storeStub) GetEmployerApplicationResume(context.Context, string, string) (Resume, error) {
	return Resume{}, s.err
}
