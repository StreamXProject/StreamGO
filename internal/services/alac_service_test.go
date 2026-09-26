package services

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"streamgo/internal/config"
	"streamgo/internal/models"
)

func TestIsALACTrack(t *testing.T) {
	svc := NewALACService(nil, nil, "")

	tests := []struct {
		name     string
		track    *models.Track
		expected bool
	}{
		{
			name:     "Nil track",
			track:    nil,
			expected: false,
		},
		{
			name: "Audio Type ALAC",
			track: &models.Track{
				Audio: models.AudioMeta{Type: "ALAC"},
			},
			expected: true,
		},
		{
			name: "Audio Genre ALAC",
			track: &models.Track{
				Audio: models.AudioMeta{Type: "m4a", Genre: "Apple Lossless (ALAC)"},
			},
			expected: true,
		},
		{
			name: "Telegram MimeType ALAC",
			track: &models.Track{
				Telegram: models.TelegramMeta{MimeType: "audio/alac"},
			},
			expected: true,
		},
		{
			name: "Telegram FileName with alac",
			track: &models.Track{
				Telegram: models.TelegramMeta{FileName: "track_alac_lossless.m4a"},
			},
			expected: true,
		},
		{
			name: "Standard MP3",
			track: &models.Track{
				Audio:    models.AudioMeta{Type: "mp3"},
				Telegram: models.TelegramMeta{MimeType: "audio/mpeg", FileName: "song.mp3"},
			},
			expected: false,
		},
		{
			name: "M4A with 24-bit bit depth",
			track: &models.Track{
				Audio: models.AudioMeta{
					Type:     "m4a",
					BitDepth: func(v int32) *int32 { return &v }(24),
				},
				Telegram: models.TelegramMeta{FileName: "song.m4a", MimeType: "audio/mp4"},
			},
			expected: true,
		},
		{
			name: "Standard FLAC",
			track: &models.Track{
				Audio:    models.AudioMeta{Type: "flac"},
				Telegram: models.TelegramMeta{MimeType: "audio/flac", FileName: "song.flac"},
			},
			expected: false,
		},
		{
			name: "M4A with BitDepth (ALAC lossless)",
			track: &models.Track{
				Audio: models.AudioMeta{
					Type:     "m4a",
					BitDepth: func() *int32 { v := int32(16); return &v }(),
				},
				Telegram: models.TelegramMeta{MimeType: "audio/mp4"},
			},
			expected: true,
		},
		{
			name: "M4A with Lossless Bitrate (> 450 kbps)",
			track: &models.Track{
				Audio: models.AudioMeta{
					Type:        "m4a",
					BitrateKbps: func() *int32 { v := int32(761); return &v }(),
				},
				Telegram: models.TelegramMeta{MimeType: "audio/mp4"},
			},
			expected: true,
		},
		{
			name: "M4A with Calculated High Bitrate from FileSize and Duration",
			track: &models.Track{
				Audio: models.AudioMeta{
					Type:        "m4a",
					DurationSec: 155,
				},
				Telegram: models.TelegramMeta{
					MimeType: "audio/mp4",
					FileSize: 15178406,
				},
			},
			expected: true,
		},
		{
			name: "M4A Standard Lossy AAC (256 kbps, no bit depth)",
			track: &models.Track{
				Audio: models.AudioMeta{
					Type:        "m4a",
					BitrateKbps: func() *int32 { v := int32(256); return &v }(),
					DurationSec: 200,
				},
				Telegram: models.TelegramMeta{
					MimeType: "audio/mp4",
					FileSize: 6400000,
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.IsALACTrack(tt.track)
			if got != tt.expected {
				t.Errorf("IsALACTrack() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldDecodeALAC(t *testing.T) {
	svc := NewALACService(nil, nil, "")

	alacTrack := &models.Track{
		Audio:    models.AudioMeta{Type: "alac"},
		Telegram: models.TelegramMeta{FileName: "song.m4a"},
	}

	mp3Track := &models.Track{
		Audio:    models.AudioMeta{Type: "mp3"},
		Telegram: models.TelegramMeta{FileName: "song.mp3"},
	}

	tests := []struct {
		name     string
		track    *models.Track
		url      string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "Non-ALAC track default",
			track:    mp3Track,
			url:      "/tracks/123/stream",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"},
			expected: false,
		},
		{
			name:     "Non-ALAC track with explicit format=flac",
			track:    mp3Track,
			url:      "/tracks/123/stream?format=flac",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"},
			expected: true,
		},
		{
			name:     "ALAC track from Chrome on Windows",
			track:    alacTrack,
			url:      "/tracks/123/stream",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"},
			expected: true,
		},
		{
			name:     "ALAC track from Firefox on Linux",
			track:    alacTrack,
			url:      "/tracks/123/stream",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0"},
			expected: true,
		},
		{
			name:     "ALAC track with format=raw override",
			track:    alacTrack,
			url:      "/tracks/123/stream?format=raw",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"},
			expected: false,
		},
		{
			name:     "ALAC track with decode=0 override",
			track:    alacTrack,
			url:      "/tracks/123/stream?decode=0",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0"},
			expected: false,
		},
		{
			name:     "ALAC track from Native client header",
			track:    alacTrack,
			url:      "/tracks/123/stream",
			headers:  map[string]string{"X-StreamX-Client": "android-native", "User-Agent": "StreamX-ExoPlayer"},
			expected: false,
		},
		{
			name:     "ALAC track with explicit format=flac from web player",
			track:    alacTrack,
			url:      "/tracks/123/stream?format=flac",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"},
			expected: true,
		},
		{
			name:     "ALAC track from native Safari on macOS without format param",
			track:    alacTrack,
			url:      "/tracks/123/stream",
			headers:  map[string]string{"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := svc.ShouldDecodeALAC(req, tt.track)
			if got != tt.expected {
				t.Errorf("ShouldDecodeALAC() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestALACCacheLRUPrune(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "alac_lru_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := &config.Config{
		AlacCacheMaxBytes: 500, // Very small byte limit for test
		AlacCacheMaxFiles: 3,   // Max 3 files
	}

	svc := NewALACService(cfg, nil, tempDir)

	// Create 5 dummy files with varying modification times and sizes (each 200 bytes)
	fileNames := []string{"track1.flac", "track2.flac", "track3.flac", "track4.flac", "track5.flac"}
	baseTime := time.Now().Add(-10 * time.Hour)

	for i, name := range fileNames {
		p := filepath.Join(svc.CacheDir(), name)
		data := make([]byte, 200)
		if err := os.WriteFile(p, data, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		// File 0 is oldest, File 4 is newest
		fileTime := baseTime.Add(time.Duration(i) * time.Hour)
		_ = os.Chtimes(p, fileTime, fileTime)
	}

	// Before prune: 5 files, 1000 bytes total
	entriesBefore, _ := os.ReadDir(svc.CacheDir())
	if len(entriesBefore) != 5 {
		t.Fatalf("expected 5 files before prune, got %d", len(entriesBefore))
	}

	// Trigger synchronous prune
	svc.PruneCacheIfNeeded()

	entriesAfter, _ := os.ReadDir(svc.CacheDir())
	// MaxBytes is 500 bytes and MaxFiles is 3. Each file is 200 bytes, so 2 files = 400 bytes <= 500 bytes.
	if len(entriesAfter) > 3 {
		t.Errorf("expected at most 3 files after prune, got %d", len(entriesAfter))
	}

	// The oldest files (track1, track2, track3) should have been pruned first.
	// track5 and track4 should be preserved!
	if _, err := os.Stat(filepath.Join(svc.CacheDir(), "track5.flac")); os.IsNotExist(err) {
		t.Errorf("expected newest file track5.flac to be preserved, but it was deleted")
	}
	if _, err := os.Stat(filepath.Join(svc.CacheDir(), "track4.flac")); os.IsNotExist(err) {
		t.Errorf("expected recent file track4.flac to be preserved, but it was deleted")
	}
	if _, err := os.Stat(filepath.Join(svc.CacheDir(), "track1.flac")); !os.IsNotExist(err) {
		t.Errorf("expected oldest file track1.flac to be pruned, but it still exists")
	}
}

