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

	// Test re-encoding back to file_id
	reEncoded := EncodeFileID(decoded.MediaID, decoded.AccessHash, decoded.DCID, decoded.FileReference)
	if reEncoded != sampleFileID {
		t.Errorf("expected re-encoded file_id %q, got %q", sampleFileID, reEncoded)
	}
}

func TestEncodeFileUniqueID(t *testing.T) {
	// Alphaville Forever Young media_id
	uid1 := EncodeFileUniqueID(6149766536438490758)
	if uid1 != "AgADhiYAAhVcWFU" {
		t.Errorf("expected 'AgADhiYAAhVcWFU', got %q", uid1)
	}

	// Taylor Swift willow media_id
	uid2 := EncodeFileUniqueID(4999148345483069633)
	if uid2 != "AgADwQgAAu-KYEU" {
		t.Errorf("expected 'AgADwQgAAu-KYEU', got %q", uid2)
	}
}
