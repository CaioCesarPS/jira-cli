package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/caiocesarps/jira-cli/internal/config"
)

func TestAttachFiles(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "screenshot.png")
	f2 := filepath.Join(dir, "clip.mp4")
	if err := os.WriteFile(f1, []byte("png-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("mp4-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotPath, gotToken, gotContentType string
	var gotFilenames []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.Header.Get("X-Atlassian-Token")
		gotContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		for _, fh := range r.MultipartForm.File["file"] {
			gotFilenames = append(gotFilenames, fh.Filename)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"10001","filename":"screenshot.png","size":9},{"id":"10002","filename":"clip.mp4","size":9}]`))
	}))
	defer server.Close()

	client := NewClient(&config.Profile{BaseURL: server.URL, Email: "a@b.com", APIToken: "tok"})

	attachments, err := client.AttachFiles("PROJ-123", []string{f1, f2})
	if err != nil {
		t.Fatalf("AttachFiles returned error: %v", err)
	}

	if gotPath != "/rest/api/3/issue/PROJ-123/attachments" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if gotToken != "no-check" {
		t.Errorf("expected X-Atlassian-Token: no-check, got %q", gotToken)
	}
	if len(gotContentType) == 0 || gotContentType[:19] != "multipart/form-data" {
		t.Errorf("expected multipart/form-data content type, got %q", gotContentType)
	}
	if len(gotFilenames) != 2 || gotFilenames[0] != "screenshot.png" || gotFilenames[1] != "clip.mp4" {
		t.Errorf("unexpected uploaded filenames: %v", gotFilenames)
	}

	if len(attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(attachments))
	}
	if attachments[0].ID != "10001" || attachments[0].Filename != "screenshot.png" {
		t.Errorf("unexpected attachment[0]: %+v", attachments[0])
	}
	if attachments[1].ID != "10002" || attachments[1].Filename != "clip.mp4" {
		t.Errorf("unexpected attachment[1]: %+v", attachments[1])
	}
}

func TestAttachFilesMissingFile(t *testing.T) {
	client := NewClient(&config.Profile{BaseURL: "http://example.invalid", Email: "a@b.com", APIToken: "tok"})

	_, err := client.AttachFiles("PROJ-123", []string{"/nonexistent/path/does-not-exist.png"})
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
