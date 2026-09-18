package services

import (
	"testing"
)

func TestParseRange(t *testing.T) {
	totalSize := int64(1000)

	tests := []struct {
		name        string
		rangeHeader string
		wantStart   int64
		wantEnd     int64
		wantLen     int64
		wantErr     bool
	}{
		{
			name:        "Empty header",
			rangeHeader: "",
			wantStart:   0,
			wantEnd:     0,
			wantLen:     0,
			wantErr:     false,
		},
		{
			name:        "Valid range 0-499",
			rangeHeader: "bytes=0-499",
			wantStart:   0,
			wantEnd:     499,
			wantLen:     500,
			wantErr:     false,
		},
		{
			name:        "Valid range 500-",
			rangeHeader: "bytes=500-",
			wantStart:   500,
			wantEnd:     999,
			wantLen:     500,
			wantErr:     false,
		},
		{
			name:        "Suffix range -200",
			rangeHeader: "bytes=-200",
			wantStart:   800,
			wantEnd:     999,
			wantLen:     200,
			wantErr:     false,
		},
		{
			name:        "Exceeds total size clamped",
			rangeHeader: "bytes=900-2000",
			wantStart:   900,
			wantEnd:     999,
			wantLen:     100,
			wantErr:     false,
		},
		{
			name:        "Invalid start > total",
			rangeHeader: "bytes=1500-",
			wantErr:     true,
		},
		{
			name:        "Invalid start > end",
			rangeHeader: "bytes=500-400",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRange(tt.rangeHeader, totalSize)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseRange(%q) error = %v, wantErr %v", tt.rangeHeader, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if tt.rangeHeader == "" {
				if got != nil {
					t.Fatalf("ParseRange(%q) expected nil, got %v", tt.rangeHeader, got)
				}
				return
			}
			if got.Start != tt.wantStart || got.End != tt.wantEnd || got.Length != tt.wantLen {
				t.Fatalf("ParseRange(%q) = {%d, %d, %d}, want {%d, %d, %d}",
					tt.rangeHeader, got.Start, got.End, got.Length, tt.wantStart, tt.wantEnd, tt.wantLen)
			}
		})
	}
}
