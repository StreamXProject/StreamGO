package services

import (
	"context"
	"testing"

	"streamgo/internal/config"
)

func TestStripNoise(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Song Title (Official Video)", "Song Title"},
		{"Track Name [Remastered 2021]", "Track Name"},
		{"Another Track [Lyric Video]", "Another Track"},
		{"Track (Live at Tokyo Dome)", "Track"},
		{"Simple Song", "Simple Song"},
		{"Official Audio Track", "Track"},
	}

	for _, tc := range tests {
		result := StripNoise(tc.input)
		if result != tc.expected {
			t.Errorf("StripNoise(%q) = %q; expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestMetadataFingerprint(t *testing.T) {
	// Alphaville Forever Young test matching original source
	fpOriginal := MetadataFingerprint("Forever Young", "Alphaville", "Disc 1", 226)
	expectedOriginal := "forever young|alphaville|disc 1|226"
	if fpOriginal != expectedOriginal {
		t.Errorf("expected %q, got %q", expectedOriginal, fpOriginal)
	}

	fp1 := MetadataFingerprint("Nandemonaiya (movie ver.)", "RADWIMPS", "", 344.2)
	fp2 := MetadataFingerprint("Nandemonaiya", "Radwimps", "", 344.8)

	// Both should not be empty
	if fp1 == "" || fp2 == "" {
		t.Fatal("Fingerprints should not be empty")
	}

	// RADWIMPS and Radwimps should match lower-cased
	if NormalizeText("RADWIMPS") != NormalizeText("Radwimps") {
		t.Errorf("expected NormalizeText to match case-insensitively")
	}

	// Check 2-second duration bucketing within same rounding slice
	fpClose1 := MetadataFingerprint("Song", "Artist", "Album", 120.1)
	fpClose2 := MetadataFingerprint("Song", "Artist", "Album", 120.8)
	if fpClose1 != fpClose2 {
		t.Errorf("expected %q == %q within 2-second bucket", fpClose1, fpClose2)
	}
}

func TestAccessFilter(t *testing.T) {
	cfg := &config.Config{
		FilterMode:    FilterModeGroupOnly,
		ChannelID:     -1004300252384,
		DumpChannelID: -1009999999999,
	}

	filter := NewAccessFilter(cfg, nil)
	ctx := context.Background()

	// ChannelID should be allowed
	if !filter.IsChatAllowed(ctx, -1004300252384) {
		t.Errorf("expected channel %d to be allowed in GroupOnly mode", -1004300252384)
	}

	// DumpChannelID should be allowed
	if !filter.IsChatAllowed(ctx, -1009999999999) {
		t.Errorf("expected dump channel %d to be allowed in GroupOnly mode", -1009999999999)
	}

	// Random unauthorized chat should be blocked
	if filter.IsChatAllowed(ctx, -1001234567890) {
		t.Errorf("expected unauthorized chat to be blocked in GroupOnly mode")
	}

	// Switch to Anyone mode
	cfg.FilterMode = FilterModeAnyone
	if !filter.IsChatAllowed(ctx, -1001234567890) {
		t.Errorf("expected any chat to be allowed in FilterModeAnyone")
	}
}
