package telegram

import (
	"testing"
)

func TestDecodeFileID(t *testing.T) {
	sampleFileID := "CQACAgEAAyEFAAMBAAFQpOAAA1JqrOHy9AzfcqpOSsZ1W9OvOkJZ4AACwQgAAu-KYEWQ5P1U3bsqzB4E"

	decoded, err := DecodeFileID(sampleFileID)
	if err != nil {
		t.Fatalf("DecodeFileID failed: %v", err)
	}

	if decoded.MediaID != 4999148345483069633 {
		t.Errorf("expected MediaID 4999148345483069633, got %d", decoded.MediaID)
	}
	if decoded.AccessHash != -3734966381662313328 {
		t.Errorf("expected AccessHash -3734966381662313328, got %d", decoded.AccessHash)
	}
	if len(decoded.FileReference) != 33 {
		t.Errorf("expected FileReference length 33, got %d", len(decoded.FileReference))
	}
	if decoded.DCID != 1 {
		t.Errorf("expected DCID 1, got %d", decoded.DCID)
	}
}
