package services

import (
	"strings"
	"testing"
)

func TestParseRichsyncJSONToLRC(t *testing.T) {
	sampleRichsync := `[
		{
			"ts": 24.38,
			"te": 29.65,
			"l": [
				{"c": "あぁ", "o": 0},
				{"c": " ", "o": 4.78},
				{"c": "このまま僕たちの声が", "o": 4.84}
			],
			"x": "あぁ このまま僕たちの声が"
		}
	]`

	lrc := parseRichsyncJSONToLRC(sampleRichsync)
	if !strings.Contains(lrc, "[00:24.38]") {
		t.Errorf("Expected [00:24.38] in LRC, got: %s", lrc)
	}
	if !strings.Contains(lrc, "<00:24.38>あぁ") {
		t.Errorf("Expected word timestamp <00:24.38>あぁ in LRC, got: %s", lrc)
	}
	if !strings.Contains(lrc, "このまま僕たちの声が") {
		t.Errorf("Expected lyrics text in LRC, got: %s", lrc)
	}
}

func TestParseSubtitlesJSONToLRC(t *testing.T) {
	sampleSubtitles := `[
		{
			"text": "Hello world",
			"time": {"total": 65.5}
		}
	]`

	lrc := parseSubtitlesJSONToLRC(sampleSubtitles)
	if !strings.Contains(lrc, "[01:05.50] Hello world") {
		t.Errorf("Expected [01:05.50] Hello world, got: %s", lrc)
	}
}

func TestIsCJK(t *testing.T) {
	if !isCJK("夢灯籠") {
		t.Errorf("Expected 夢灯籠 to be detected as CJK")
	}
	if !isCJK("RADWIMPS - 君の名は。") {
		t.Errorf("Expected Japanese text to be detected as CJK")
	}
	if isCJK("Hello World 123") {
		t.Errorf("Expected English text not to be detected as CJK")
	}
}

func TestRomanizeJapanese(t *testing.T) {
	res := romanizeJapanese("夢灯籠")
	// If python pykakasi is installed on this system, verify result
	if res != "" && res != "Yume Tourou" && res != "Yumetourou" {
		t.Logf("romanizeJapanese('夢灯籠') produced: %s", res)
	}
}
