package site_test

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/voidmind-io/voidllm/internal/site"
)

func TestSaveUploadedLogo_PNG(t *testing.T) {
	dir := t.TempDir()
	header := multipartFileHeader(t, "logo.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})

	publicPath, err := site.SaveUploadedLogo(dir, header)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if publicPath != site.LogoUploadURLPrefix+"logo.png" {
		t.Fatalf("public path = %q", publicPath)
	}
	if _, err := os.Stat(filepath.Join(site.LogoDir(dir), "logo.png")); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}

func TestSaveUploadedLogo_ReplacesPrevious(t *testing.T) {
	dir := t.TempDir()
	first, err := site.SaveUploadedLogo(dir, multipartFileHeader(t, "logo.png", "image/png", []byte{1, 2, 3}))
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	second, err := site.SaveUploadedLogo(dir, multipartFileHeader(t, "logo.jpg", "image/jpeg", []byte{4, 5, 6}))
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if second == first {
		t.Fatalf("expected different public path")
	}
	if _, err := os.Stat(filepath.Join(site.LogoDir(dir), "logo.png")); !os.IsNotExist(err) {
		t.Fatalf("old png should be removed")
	}
}

func TestSaveUploadedLogo_RejectsLargeFile(t *testing.T) {
	dir := t.TempDir()
	payload := make([]byte, (2<<20)+1)
	_, err := site.SaveUploadedLogo(dir, multipartFileHeader(t, "logo.png", "image/png", payload))
	if err == nil {
		t.Fatal("expected size error")
	}
}

func multipartFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": []string{`form-data; name="logo"; filename="` + filename + `"`},
		"Content-Type":        []string{contentType},
	})
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	t.Cleanup(func() { _ = form.RemoveAll() })
	files := form.File["logo"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	files[0].Size = int64(len(data))
	return files[0]
}