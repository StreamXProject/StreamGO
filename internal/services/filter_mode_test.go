package services

import (
	"context"
	"testing"

	"streamgo/internal/config"
)

func TestParseFilterMode(t *testing.T) {
	tests := []struct {
		input    any
		expected int
		name     string
	}{
		{0, FilterModeGroupOnly, "int 0"},
		{1, FilterModeAnyone, "int 1"},
		{2, FilterModeHybrid, "int 2"},
		{"0", FilterModeGroupOnly, "str 0"},
		{"group_only", FilterModeGroupOnly, "str group_only"},
		{"1", FilterModeAnyone, "str 1"},
		{"anyone", FilterModeAnyone, "str anyone"},
		{"any", FilterModeAnyone, "str any"},
		{"all", FilterModeAnyone, "str all"},
		{"2", FilterModeHybrid, "str 2"},
		{"hybrid", FilterModeHybrid, "str hybrid"},
		{"whitelist", FilterModeHybrid, "str whitelist"},
		{"unknown", FilterModeGroupOnly, "unknown fallback"},
		{nil, FilterModeGroupOnly, "nil fallback"},
	}

	for _, tt := range tests {
		got := ParseFilterMode(tt.input)
		if got != tt.expected {
			t.Errorf("[%s] ParseFilterMode(%v) = %d; want %d", tt.name, tt.input, got, tt.expected)
		}
	}
}

func TestFilterModeToString(t *testing.T) {
	if FilterModeToString(FilterModeGroupOnly) != "group_only" {
		t.Errorf("expected group_only")
	}
	if FilterModeToString(FilterModeAnyone) != "anyone" {
		t.Errorf("expected anyone")
	}
	if FilterModeToString(FilterModeHybrid) != "hybrid" {
		t.Errorf("expected hybrid")
	}
}

func TestAccessFilterLogic(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		ChannelID:     -1003961478268,
		DumpChannelID: -1004300252384,
		FilterMode:    FilterModeGroupOnly,
	}

	filter := &AccessFilter{
		cfg:          cfg,
		mode:         FilterModeGroupOnly,
		allowedChats: make(map[int64]bool),
		bannedChats:  make(map[int64]bool),
	}
	filter.allowedChats[cfg.ChannelID] = true
	filter.allowedChats[cfg.DumpChannelID] = true

	randomChat := int64(-1009999999999)
	allowedContributor := int64(-1008888888888)

	// Mode 0 (GROUP_ONLY)
	filter.mode = FilterModeGroupOnly
	if !filter.IsChatAllowed(ctx, cfg.ChannelID) {
		t.Errorf("Mode 0: ChannelID should be allowed")
	}
	if !filter.IsChatAllowed(ctx, cfg.DumpChannelID) {
		t.Errorf("Mode 0: DumpChannelID should be allowed")
	}
	if filter.IsChatAllowed(ctx, randomChat) {
		t.Errorf("Mode 0: Random chat should be rejected")
	}

	// Mode 1 (ANYONE)
	filter.mode = FilterModeAnyone
	if !filter.IsChatAllowed(ctx, randomChat) {
		t.Errorf("Mode 1: Random chat should be allowed")
	}

	// Strict Ban across all modes
	filter.bannedChats[randomChat] = true
	if filter.IsChatAllowed(ctx, randomChat) {
		t.Errorf("Mode 1: Banned chat must be rejected strictly")
	}

	// Mode 2 (HYBRID)
	filter.mode = FilterModeHybrid
	delete(filter.bannedChats, randomChat)

	if !filter.IsChatAllowed(ctx, cfg.ChannelID) {
		t.Errorf("Mode 2: ChannelID should be allowed")
	}
	if filter.IsChatAllowed(ctx, randomChat) {
		t.Errorf("Mode 2: Unlisted random chat should be rejected")
	}

	// Add contributor to allowed
	filter.allowedChats[allowedContributor] = true
	if !filter.IsChatAllowed(ctx, allowedContributor) {
		t.Errorf("Mode 2: Listed allowed contributor should be allowed")
	}

	// Ban contributor -> strict ban must reject
	filter.bannedChats[allowedContributor] = true
	if filter.IsChatAllowed(ctx, allowedContributor) {
		t.Errorf("Mode 2: Banned contributor must be rejected strictly")
	}
}
