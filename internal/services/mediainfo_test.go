package services

import (
	"testing"
)

func TestParseMediaInfoWillow(t *testing.T) {
	sampleOutput := `General
Complete name                            : /tmp/willow.part
Format                                   : FLAC
Format/Info                              : Free Lossless Audio Codec
File size                                : 72.5 MiB
Duration                                 : 3 min 34 s
Overall bit rate mode                    : Variable
Overall bit rate                         : 2 708 kb/s
Album                                    : evermore (deluxe version)
Track name                               : willow
Track name/Position                      : 1
Track name/Total                         : 17
Performer                                : Taylor Swift
Composer                                 : Taylor Swift / Aaron Dessner
Publisher                                : Taylor Swift
Genre                                    : Alternative & Indie, Pop/Rock, Rock
Recorded date                            : 2020-12-11
ISRC                                     : USUG12004457
Cover                                    : Yes
Cover type                               : Cover
Cover MIME                               : image/jpeg

Audio
Format                                   : FLAC
Format/Info                              : Free Lossless Audio Codec
Duration                                 : 3 min 34 s
Bit rate mode                            : Variable
Bit rate                                 : 72 kb/s
Channel(s)                               : 2 channels
Channel layout                           : L R
Sampling rate                            : 88.2 kHz
Bit depth                                : 24 bits
Compression mode                         : Lossless
Stream size                              : 72.5 MiB (100%)
Writing library                          : libFLAC 1.3.2 (UTC 2017-01-01)
`

	meta := ParseMediaInfo(sampleOutput, 0, 72455262)
	if meta == nil {
		t.Fatal("expected non-nil AudioMeta")
	}

	if meta.Title != "willow" {
		t.Errorf("expected title 'willow', got '%s'", meta.Title)
	}
	if meta.Album != "evermore (deluxe version)" {
		t.Errorf("expected album 'evermore (deluxe version)', got '%s'", meta.Album)
	}
	if meta.Artist != "Taylor Swift" {
		t.Errorf("expected artist 'Taylor Swift', got '%s'", meta.Artist)
	}
	if meta.Composer != "Taylor Swift / Aaron Dessner" {
		t.Errorf("expected composer 'Taylor Swift / Aaron Dessner', got '%s'", meta.Composer)
	}
	if meta.Label != "Taylor Swift" {
		t.Errorf("expected label 'Taylor Swift', got '%s'", meta.Label)
	}
	if meta.Genre != "Alternative & Indie, Pop/Rock, Rock" {
		t.Errorf("expected genre 'Alternative & Indie, Pop/Rock, Rock', got '%s'", meta.Genre)
	}
	if meta.Year == nil || *meta.Year != 2020 {
		t.Errorf("expected year 2020, got %v", meta.Year)
	}
	if meta.DurationSec != 214 {
		t.Errorf("expected duration_sec 214, got %d", meta.DurationSec)
	}
	if meta.Type != "flac" {
		t.Errorf("expected type 'flac', got '%s'", meta.Type)
	}
	if meta.BitDepth == nil || *meta.BitDepth != 24 {
		t.Errorf("expected bit_depth 24, got %v", meta.BitDepth)
	}
	if meta.SamplingRateHz == nil || *meta.SamplingRateHz != 88200 {
		t.Errorf("expected sampling_rate_hz 88200, got %v", meta.SamplingRateHz)
	}
	if meta.AlbumID != "album_evermore_deluxe_version_2020" {
		t.Errorf("expected album_id 'album_evermore_deluxe_version_2020', got '%s'", meta.AlbumID)
	}
	if len(meta.Artists) != 1 || meta.Artists[0] != "Taylor Swift" {
		t.Errorf("expected artists ['Taylor Swift'], got %v", meta.Artists)
	}

	fp := BuildMetadataFingerprint(meta.Title, meta.Artist, meta.Album, meta.DurationSec)
	if fp != "willow|taylor swift|evermore deluxe version|214" {
		t.Errorf("expected fingerprint 'willow|taylor swift|evermore deluxe version|214', got '%s'", fp)
	}
}

func TestSplitArtists(t *testing.T) {
	artists := SplitArtists("Taylor Swift, Aaron Dessner feat. Bon Iver / Jack Antonoff & HAIM and Marcus Mumford")
	expected := []string{"Taylor Swift", "Aaron Dessner", "Bon Iver", "Jack Antonoff", "HAIM", "Marcus Mumford"}

	if len(artists) != len(expected) {
		t.Fatalf("expected %d artists, got %d: %v", len(expected), len(artists), artists)
	}
	for i, v := range expected {
		if artists[i] != v {
			t.Errorf("expected artist[%d] == %s, got %s", i, v, artists[i])
		}
	}
}

func TestNormalizeMimeType(t *testing.T) {
	tests := []struct {
		mime     string
		fileName string
		expected string
	}{
		{"audio/x-flac", "01 - willow.flac", "audio/flac"},
		{"audio/flac", "song.flac", "audio/flac"},
		{"audio/mp3", "track.mp3", "audio/mpeg"},
		{"audio/x-m4a", "track.m4a", "audio/mp4"},
		{"audio/x-wav", "track.wav", "audio/wav"},
	}

	for _, tc := range tests {
		got := NormalizeMimeType(tc.mime, tc.fileName)
		if got != tc.expected {
			t.Errorf("NormalizeMimeType(%s, %s) = %s, expected %s", tc.mime, tc.fileName, got, tc.expected)
		}
	}
}

func TestParseMediaInfoALAC(t *testing.T) {
	alacOutput := `General
Format                                   : MPEG-4
Format profile                           : Apple audio with iTunes info
Codec ID                                 : M4A (M4A /mp42/isom)
File size                                : 55.2 MiB
Duration                                 : 5 min 0 s
Overall bit rate mode                    : Variable
Overall bit rate                         : 1 544 kb/s
Album                                    : Hounds of Love (2018 Remaster)
Track name                               : Running Up That Hill (A Deal With God) [2018 Remaster]
Performer                                : Kate Bush
Composer                                 : Kate Bush
Genre                                    : Pop
Recorded date                            : 1985

Audio
Format                                   : ALAC
Format/Info                              : Apple Lossless Audio Codec
Codec ID                                 : alac
Codec ID/Info                            : Apple Lossless Audio Codec
Duration                                 : 5 min 0 s
Bit rate mode                            : Variable
Bit rate                                 : 1 536 kb/s
Channel(s)                               : 2 channels
Sampling rate                            : 44.1 kHz
Bit depth                                : 24 bits
Stream size                              : 55.2 MiB (100%)
`

	meta := ParseMediaInfo(alacOutput, 0, 57942992)
	if meta == nil {
		t.Fatal("expected non-nil AudioMeta")
	}
	if meta.Type != "alac" {
		t.Errorf("expected type 'alac', got '%s'", meta.Type)
	}
	if meta.BitDepth == nil || *meta.BitDepth != 24 {
		t.Errorf("expected bit depth 24, got %v", meta.BitDepth)
	}
	if meta.Title != "Running Up That Hill (A Deal With God) [2018 Remaster]" {
		t.Errorf("expected title 'Running Up That Hill (A Deal With God) [2018 Remaster]', got '%s'", meta.Title)
	}
}

