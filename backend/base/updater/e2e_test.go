//go:build linux

package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestEndToEndCheckDownloadInstall walks the full backend update flow against a
// real HTTP update server: check -> download + verify -> install (AppImage).
func TestEndToEndCheckDownloadInstall(t *testing.T) {
	payload := []byte("#!/bin/sh\necho installed\n")
	sum := sha256.Sum256(payload)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app_manager/updater":
			if r.Method != http.MethodPost {
				t.Errorf("check method = %s", r.Method)
			}
			var req checkRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode check request: %v", err)
			}
			if req.Version != "v0.3.0-beta" {
				t.Errorf("current version = %q", req.Version)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"msg":  "ok",
				"data": Release{
					Version: "v0.4.0",
					Summary: "test release",
					URL:     server.URL + "/download",
					Assets: []Asset{{
						OS:     "linux",
						Arch:   "x64",
						Kind:   "AppImage",
						Name:   "QuotaPanel-0.4.0-linux-amd64.AppImage",
						Size:   int64(len(payload)),
						SHA256: hex.EncodeToString(sum[:]),
						URLs:   []string{server.URL + "/QuotaPanel-0.4.0-linux-amd64.AppImage"},
					}},
				},
			})
		case "/QuotaPanel-0.4.0-linux-amd64.AppImage":
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dataDir := t.TempDir()
	manager := New(Options{
		Endpoint:       server.URL + "/app_manager/updater",
		CurrentVersion: "v0.3.0-beta",
		DataDir:        dataDir,
		Platform:       Platform{Name: "linux", Arch: "x64"},
		UserAgent:      "QuotaPanel-Test",
	})
	// The downloaded payload is a shell script, not a real AppImage; skip the
	// ELF dependency pre-flight which would try to extract it.
	oldCheck := installDependencyCheck
	installDependencyCheck = nil
	defer func() { installDependencyCheck = oldCheck }()
	oldRelaunch := relaunchAfterInstall
	relaunchAfterInstall = false
	defer func() { relaunchAfterInstall = oldRelaunch }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	state, err := manager.Check(ctx)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if state.State != StateAvailable || state.Asset == nil || state.Asset.Kind != "AppImage" {
		t.Fatalf("check state = %+v", state)
	}
	if state.Release == nil || state.Release.Version != "v0.4.0" {
		t.Fatalf("release = %+v", state.Release)
	}

	// A listener sees every state transition the UI would receive.
	var transitions []string
	manager.opts.OnEvent = func(ev Event) { transitions = append(transitions, ev.State.State) }

	state, err = manager.Download(ctx)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if state.State != StateReady || state.Progress != 100 {
		t.Fatalf("download state = %+v", state)
	}
	downloaded, err := os.ReadFile(state.DownloadedPath)
	if err != nil {
		t.Fatalf("read download: %v", err)
	}
	if string(downloaded) != string(payload) {
		t.Fatalf("downloaded payload = %q", downloaded)
	}
	if !containsString(transitions, StateDownloading) || !containsString(transitions, StateReady) {
		t.Fatalf("transitions = %v", transitions)
	}

	// Install replaces the existing AppImage in place.
	target := filepath.Join(t.TempDir(), "QuotaPanel.AppImage")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", target)
	result, state, err := manager.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !result.RestartRequired || state.State != StateRestarting {
		t.Fatalf("install result=%+v state=%+v", result, state)
	}
	installed, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(payload) {
		t.Fatalf("installed payload = %q", installed)
	}
	if _, err := os.Stat(target + ".old"); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
