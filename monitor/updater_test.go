package monitor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	if !newerVersion("0.2.0", "0.1.1") {
		t.Error("0.2.0 should be newer than 0.1.1")
	}
	if newerVersion("0.1.1", "0.1.1") {
		t.Error("same version should not be newer")
	}
	if newerVersion("0.1.0", "0.1.1") {
		t.Error("0.1.0 should not be newer than 0.1.1")
	}
	if !newerVersion("1.0.0", "0.9.9") {
		t.Error("1.0.0 should be newer than 0.9.9")
	}
	if !newerVersion("0.1.2", "0.1.1") {
		t.Error("0.1.2 should be newer than 0.1.1")
	}
}

func TestUpdaterCheck_UpdateAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v0.2.0",
			"html_url": "https://github.com/test/repo/releases/tag/v0.2.0",
		})
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected update info, got nil")
	}
	if info.Version != "0.2.0" {
		t.Errorf("expected version 0.2.0, got %s", info.Version)
	}
	if info.DownloadURL != "https://github.com/test/repo/releases/tag/v0.2.0" {
		t.Errorf("unexpected download URL: %s", info.DownloadURL)
	}
}

func TestUpdaterCheck_UpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"tag_name": "v0.1.1",
			"html_url": "https://github.com/test/repo/releases/tag/v0.1.1",
		})
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Errorf("expected nil (up to date), got %+v", info)
	}
}

func TestUpdaterCheck_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	u := NewUpdater("0.1.1", srv.URL)
	info, err := u.Check()
	if err == nil {
		t.Error("expected error on 500 response")
	}
	if info != nil {
		t.Error("expected nil info on error")
	}
}
