package services

import (
	"testing"
	"time"

	"streamgo/internal/models"
)

func TestPeriodBounds(t *testing.T) {
	tz := 0 // UTC

	// Weekly: 2026-W38
	start, end, label, err := PeriodBounds(models.RecapPeriodWeekly, "2026-W38", tz)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end <= start {
		t.Errorf("expected end > start, got start=%f end=%f", start, end)
	}
	if end-start != 7*24*3600 {
		t.Errorf("expected 7 days duration in seconds (604800), got %f", end-start)
	}
	if label == "" {
		t.Errorf("expected non-empty label")
	}

	// Weekly invalid
	_, _, _, err = PeriodBounds(models.RecapPeriodWeekly, "invalid-week", tz)
	if err == nil {
		t.Errorf("expected error on invalid weekly period")
	}

	// Monthly: 2026-09
	start, end, label, err = PeriodBounds(models.RecapPeriodMonthly, "2026-09", tz)
	if err != nil {
		t.Fatalf("unexpected error for monthly: %v", err)
	}
	if label != "September 2026" {
		t.Errorf("expected 'September 2026', got '%s'", label)
	}
	// September has 30 days = 30 * 86400 = 2592000 seconds
	if end-start != 30*86400 {
		t.Errorf("expected 30 days for September, got %f seconds", end-start)
	}

	// Monthly rollover: 2026-12 -> next year 2027-01
	startDec, endDec, _, err := PeriodBounds(models.RecapPeriodMonthly, "2026-12", tz)
	if err != nil {
		t.Fatalf("unexpected error for December: %v", err)
	}
	// December has 31 days
	if endDec-startDec != 31*86400 {
		t.Errorf("expected 31 days for December, got %f seconds", endDec-startDec)
	}

	// Yearly: 2026
	startY, endY, labelY, err := PeriodBounds(models.RecapPeriodYearly, "2026", tz)
	if err != nil {
		t.Fatalf("unexpected error for yearly: %v", err)
	}
	if labelY != "2026" {
		t.Errorf("expected '2026', got '%s'", labelY)
	}
	// 2026 is non-leap year = 365 days
	if endY-startY != 365*86400 {
		t.Errorf("expected 365 days for 2026, got %f seconds", endY-startY)
	}

	// Timezone offset test: tz = +330 (IST: UTC+5:30)
	startIST, endIST, _, err := PeriodBounds(models.RecapPeriodMonthly, "2026-09", 330)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Start in UTC epoch for Sept 1 00:00 IST is Aug 31 18:30 UTC
	expectedStart := float64(time.Date(2026, 8, 31, 18, 30, 0, 0, time.UTC).Unix())
	if startIST != expectedStart {
		t.Errorf("expected start %f, got %f", expectedStart, startIST)
	}
	if endIST-startIST != 30*86400 {
		t.Errorf("expected 30 days duration regardless of timezone, got %f", endIST-startIST)
	}
}

func TestPreviousPeriod(t *testing.T) {
	// Weekly
	prev := PreviousPeriod(models.RecapPeriodWeekly, "2026-W38")
	if prev != "2026-W37" {
		t.Errorf("expected 2026-W37, got %s", prev)
	}

	// Monthly
	prevM := PreviousPeriod(models.RecapPeriodMonthly, "2026-09")
	if prevM != "2026-08" {
		t.Errorf("expected 2026-08, got %s", prevM)
	}
	prevJan := PreviousPeriod(models.RecapPeriodMonthly, "2026-01")
	if prevJan != "2025-12" {
		t.Errorf("expected 2025-12, got %s", prevJan)
	}

	// Yearly
	prevY := PreviousPeriod(models.RecapPeriodYearly, "2026")
	if prevY != "2025" {
		t.Errorf("expected 2025, got %s", prevY)
	}
}

func TestPeriodIDFor(t *testing.T) {
	tm := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	wID := PeriodIDFor(models.RecapPeriodWeekly, tm)
	if wID != "2026-W39" {
		t.Errorf("expected 2026-W39, got %s", wID)
	}

	mID := PeriodIDFor(models.RecapPeriodMonthly, tm)
	if mID != "2026-09" {
		t.Errorf("expected 2026-09, got %s", mID)
	}

	yID := PeriodIDFor(models.RecapPeriodYearly, tm)
	if yID != "2026" {
		t.Errorf("expected 2026, got %s", yID)
	}
}

func TestIsPeriodAvailable(t *testing.T) {
	// Past period: 2020-01 should always be available
	if !IsPeriodAvailable(models.RecapPeriodMonthly, "2020-01", 0) {
		t.Errorf("expected past monthly period to be available")
	}

	if !IsPeriodAvailable(models.RecapPeriodWeekly, "2020-W01", 0) {
		t.Errorf("expected past weekly period to be available")
	}

	if !IsPeriodAvailable(models.RecapPeriodYearly, "2020", 0) {
		t.Errorf("expected past yearly period to be available")
	}

	// Future period (e.g. 2099-01) should NOT be available unless today is the unlock date
	if IsPeriodAvailable(models.RecapPeriodMonthly, "2099-01", 0) {
		t.Errorf("expected future monthly period to NOT be available")
	}

	// Invalid format returns false
	if IsPeriodAvailable(models.RecapPeriodWeekly, "invalid", 0) {
		t.Errorf("expected invalid period to return false")
	}
}

func TestClassifyEvent(t *testing.T) {
	playMs20 := 20_000 // 20s
	playMs45 := 45_000 // 45s
	duration := 180_000 // 3 min

	// 1. Under 30s, not completed -> skip
	evSkip := &models.ListeningEventDoc{
		PlayedMs:   &playMs20,
		Completed:  false,
		DurationMs: duration,
	}
	isPlay, completed, isSkip, played := classifyEvent(evSkip, duration)
	if isPlay || !isSkip || completed {
		t.Errorf("expected skip for 20s play, got isPlay=%v, isSkip=%v, completed=%v", isPlay, isSkip, completed)
	}
	if played != 20_000 {
		t.Errorf("expected playedMs 20000, got %d", played)
	}

	// 2. Over 30s -> play
	evPlay := &models.ListeningEventDoc{
		PlayedMs:   &playMs45,
		Completed:  false,
		DurationMs: duration,
	}
	isPlay, completed, isSkip, played = classifyEvent(evPlay, duration)
	if !isPlay || isSkip || completed {
		t.Errorf("expected play for 45s play, got isPlay=%v, isSkip=%v, completed=%v", isPlay, isSkip, completed)
	}
	if played != 45_000 {
		t.Errorf("expected playedMs 45000, got %d", played)
	}

	// 3. Under 30s but completed flag true -> play & completed
	evCompFlag := &models.ListeningEventDoc{
		PlayedMs:   &playMs20,
		Completed:  true,
		DurationMs: duration,
	}
	isPlay, completed, isSkip, _ = classifyEvent(evCompFlag, duration)
	if !isPlay || isSkip || !completed {
		t.Errorf("expected completed play when Completed=true, got isPlay=%v, isSkip=%v, completed=%v", isPlay, isSkip, completed)
	}

	// 4. Completed >= 80% duration
	playMs150 := 150_000 // > 80% of 180s
	ev80 := &models.ListeningEventDoc{
		PlayedMs:   &playMs150,
		Completed:  false,
		DurationMs: duration,
	}
	isPlay, completed, isSkip, _ = classifyEvent(ev80, duration)
	if !isPlay || isSkip || !completed {
		t.Errorf("expected completed play when played >= 80%% duration, got isPlay=%v, isSkip=%v, completed=%v", isPlay, isSkip, completed)
	}

	// 5. Legacy event with nil PlayedMs
	evLegacy := &models.ListeningEventDoc{
		PlayedMs:   nil,
		DurationMs: 200_000,
	}
	isPlay, completed, isSkip, played = classifyEvent(evLegacy, 200_000)
	if !isPlay || isSkip || !completed || played != 200_000 {
		t.Errorf("expected legacy event to assume full duration play, got isPlay=%v, isSkip=%v, completed=%v, played=%d", isPlay, isSkip, completed, played)
	}
}

func TestComputeStats(t *testing.T) {
	svc := &RecapService{}

	playMs60 := 60_000
	playMs10 := 10_000

	baseTime := float64(time.Date(2026, 9, 21, 22, 0, 0, 0, time.UTC).Unix()) // 22:00 = 10 PM (night hour)

	events := []*models.ListeningEventDoc{
		{
			ID:         "ev1",
			TrackID:    "t1",
			PlayedAt:   baseTime,
			StartedAt:  baseTime - 60,
			PlayedMs:   &playMs60,
			DurationMs: 120_000,
			Completed:  false,
			Source:     "server",
		},
		{
			ID:         "ev2",
			TrackID:    "t1",
			PlayedAt:   baseTime + 300,
			StartedAt:  baseTime + 240,
			PlayedMs:   &playMs60,
			DurationMs: 120_000,
			Completed:  true,
			Source:     "flac",
		},
		{
			ID:         "ev3",
			TrackID:    "t2",
			PlayedAt:   baseTime + 600,
			StartedAt:  baseTime + 590,
			PlayedMs:   &playMs10, // skip
			DurationMs: 120_000,
			Completed:  false,
		},
	}

	tracks := map[string]*models.Track{
		"t1": {
			ID: "t1",
			Audio: models.AudioMeta{
				Title:       "Track One",
				Artist:      "Artist A",
				Album:       "Album A",
				DurationSec: 120,
				Type:        "flac",
			},
		},
		"t2": {
			ID: "t2",
			Audio: models.AudioMeta{
				Title:       "Track Two",
				Artist:      "Artist B",
				Album:       "Album B",
				DurationSec: 120,
			},
		},
	}

	seenBefore := map[string]bool{
		"t1": true, // repeat
	}

	stats := svc.ComputeStats(events, tracks, 0, seenBefore)

	if stats.TotalPlays != 2 {
		t.Errorf("expected 2 plays, got %d", stats.TotalPlays)
	}
	if stats.SkippedPlays != 1 {
		t.Errorf("expected 1 skip, got %d", stats.SkippedPlays)
	}
	if stats.CompletedPlays != 1 {
		t.Errorf("expected 1 completed play, got %d", stats.CompletedPlays)
	}
	if stats.TotalMinutes != 2 {
		t.Errorf("expected 2 total minutes (120s), got %d", stats.TotalMinutes)
	}
	if stats.UniqueTracks != 1 {
		t.Errorf("expected 1 unique track played, got %d", stats.UniqueTracks)
	}
	if stats.UniqueArtists != 1 {
		t.Errorf("expected 1 unique artist, got %d", stats.UniqueArtists)
	}
	if len(stats.TopTracks) != 1 || stats.TopTracks[0].Track.ID != "t1" {
		t.Errorf("expected t1 as top track")
	}
	if len(stats.TopArtists) != 1 || stats.TopArtists[0].Name != "Artist A" {
		t.Errorf("expected Artist A as top artist")
	}
	if stats.RepeatCount != 1 {
		t.Errorf("expected 1 repeat, got %d", stats.RepeatCount)
	}
	// Base time is 22:00, which is in nightHours (21-03)
	if stats.NightShare != 1.0 {
		t.Errorf("expected nightShare 1.0, got %f", stats.NightShare)
	}
}

func TestComputePersonality(t *testing.T) {
	// Night owl
	statsNight := &models.RecapStats{
		NightShare: 0.55,
		TotalPlays: 20,
	}
	traits := ComputePersonality(statsNight)
	foundNight := false
	for _, tr := range traits {
		if tr.ID == "night_owl" {
			foundNight = true
			break
		}
	}
	if !foundNight {
		t.Errorf("expected night_owl personality trait")
	}

	// Audiophile
	statsHiFi := &models.RecapStats{
		LosslessShare: 0.65,
		TotalPlays:    20,
	}
	traitsHiFi := ComputePersonality(statsHiFi)
	foundHiFi := false
	for _, tr := range traitsHiFi {
		if tr.ID == "audiophile" {
			foundHiFi = true
			break
		}
	}
	if !foundHiFi {
		t.Errorf("expected audiophile personality trait")
	}

	// Explorer
	statsExplorer := &models.RecapStats{
		DiscoveryCount: 15,
		UniqueTracks:   20,
		TotalPlays:     30,
	}
	traitsExp := ComputePersonality(statsExplorer)
	foundExp := false
	for _, tr := range traitsExp {
		if tr.ID == "explorer" {
			foundExp = true
			break
		}
	}
	if !foundExp {
		t.Errorf("expected explorer personality trait")
	}
}

func TestComputeComparison(t *testing.T) {
	curr := &models.RecapStats{
		TotalMinutes: 120,
		TotalPlays:   30,
		UniqueArtists: 10,
		NightShare:   0.40,
	}

	prev := &models.RecapStats{
		TotalMinutes: 100,
		TotalPlays:   20,
		UniqueArtists: 8,
		NightShare:   0.30,
		TopArtists: []*models.ArtistStat{
			{Name: "Previous Top Artist"},
		},
	}

	comp := ComputeComparison("2026-08", curr, prev)
	if comp == nil {
		t.Fatalf("expected non-nil comparison")
	}

	if comp.Period != "2026-08" {
		t.Errorf("expected period 2026-08, got %s", comp.Period)
	}
	if comp.MinutesDeltaPct == nil || *comp.MinutesDeltaPct != 20 {
		t.Errorf("expected +20%% minutes delta, got %v", comp.MinutesDeltaPct)
	}
	if comp.PlaysDeltaPct == nil || *comp.PlaysDeltaPct != 50 {
		t.Errorf("expected +50%% plays delta, got %v", comp.PlaysDeltaPct)
	}
	if comp.ArtistsDelta != 2 {
		t.Errorf("expected artistsDelta +2, got %d", comp.ArtistsDelta)
	}
	if comp.NightShareDelta != 10 { // 40% - 30% = 10%
		t.Errorf("expected nightShareDelta +10, got %d", comp.NightShareDelta)
	}
	if comp.TopArtistPrev == nil || *comp.TopArtistPrev != "Previous Top Artist" {
		t.Errorf("expected Previous Top Artist, got %v", comp.TopArtistPrev)
	}
}

func TestPublicSummary(t *testing.T) {
	trackTitle := "Test Song"
	artistName := "Test Artist"
	snap := &models.RecapSnapshot{
		Type:   models.RecapPeriodMonthly,
		Period: "2026-09",
		Label:  "September 2026",
		Stats: &models.RecapStats{
			TotalMinutes:  150,
			TotalPlays:    40,
			UniqueArtists: 5,
			UniqueTracks:  12,
			TopArtists: []*models.ArtistStat{
				{Name: artistName, Plays: 25, Minutes: 90},
			},
			TopTracks: []*models.TrackStat{
				{
					Track: &models.BrowseItem{
						ID:     "t1",
						Title:  trackTitle,
						Artist: artistName,
					},
					Plays:   15,
					Minutes: 50,
				},
			},
		},
		Personality: []*models.RecapPersonality{
			{ID: "night_owl", Name: "Night Owl", Detail: "You listen at night"},
		},
	}

	summary := PublicSummary(snap)
	if summary.Type != models.RecapPeriodMonthly {
		t.Errorf("expected monthly, got %s", summary.Type)
	}
	if summary.Period != "2026-09" {
		t.Errorf("expected 2026-09, got %s", summary.Period)
	}
	if summary.TotalMinutes != 150 {
		t.Errorf("expected 150 minutes, got %d", summary.TotalMinutes)
	}
	if summary.TopArtist == nil || summary.TopArtist.Name != artistName {
		t.Errorf("expected top artist %s", artistName)
	}
	if summary.TopTrack == nil || *summary.TopTrack.Title != trackTitle {
		t.Errorf("expected top track %s", trackTitle)
	}
	if len(summary.Personality) != 1 || summary.Personality[0].ID != "night_owl" {
		t.Errorf("expected night_owl personality in public summary")
	}
}
