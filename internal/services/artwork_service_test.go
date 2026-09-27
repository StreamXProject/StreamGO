package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"testing"
)

func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func createTestMP4CovrChunk(imgBytes []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("ftypM4A ")
	buf.WriteString("moov")
	buf.WriteString("udta")
	buf.WriteString("meta")
	buf.WriteString("ilst")

	// covr atom
	covrStart := buf.Len()
	buf.Write([]byte{0, 0, 0, 0}) // placeholder for covr size
	buf.WriteString("covr")

	// data atom
	dataStart := buf.Len()
	buf.Write([]byte{0, 0, 0, 0}) // placeholder for data size
	buf.WriteString("data")
	buf.Write([]byte{0, 0, 0, 13}) // type 13 = jpeg
	buf.Write([]byte{0, 0, 0, 0})  // locale 0
	buf.Write(imgBytes)

	dataLen := uint32(buf.Len() - dataStart)
	binary.BigEndian.PutUint32(buf.Bytes()[dataStart:dataStart+4], dataLen)

	covrLen := uint32(buf.Len() - covrStart)
	binary.BigEndian.PutUint32(buf.Bytes()[covrStart:covrStart+4], covrLen)

	return buf.Bytes()
}

func TestArtworkService_ExtractMP4Cover(t *testing.T) {
	testImg := createTestJPEG(100, 100)
	chunk := createTestMP4CovrChunk(testImg)

	svc := NewArtworkService()
	extracted, mime, err := svc.ExtractArtwork(chunk, "alac")
	if err != nil {
		t.Fatalf("Failed to extract MP4 cover: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("Expected image/jpeg mime, got %s", mime)
	}
	if len(extracted) != len(testImg) {
		t.Fatalf("Extracted size %d != expected %d", len(extracted), len(testImg))
	}
}

func TestArtworkService_CompressToWebP(t *testing.T) {
	testImg := createTestJPEG(500, 500)
	svc := NewArtworkService()

	webpBytes, err := svc.CompressToWebP(context.Background(), testImg, 300, 80)
	if err != nil {
		t.Fatalf("CompressToWebP failed: %v", err)
	}
	if len(webpBytes) == 0 {
		t.Fatal("Expected non-empty webp output")
	}
	if !bytes.HasPrefix(webpBytes, []byte("RIFF")) || !bytes.Contains(webpBytes[:16], []byte("WEBP")) {
		t.Fatal("Output does not have valid WebP magic header")
	}
}

func TestArtworkService_ExtractID3Picture_PNG_Syncsafe(t *testing.T) {
	// 1. Construct PNG payload that contains \xff\xd8\xff in its body
	pngPayload := []byte("\x89PNG\r\n\x1a\n")
	pngPayload = append(pngPayload, []byte("fake_chunk_data_with_\xff\xd8\xff_embedded_in_stream")...)
	pngPayload = append(pngPayload, []byte("IEND\xae\x42\x60\x82")...)

	// APIC payload: [encoding: 1B] [mime: null-terminated] [picType: 1B] [desc: null-terminated] [image data]
	var apicPayload bytes.Buffer
	apicPayload.WriteByte(0) // ISO-8859-1
	apicPayload.WriteString("image/png\x00")
	apicPayload.WriteByte(3) // Cover (front)
	apicPayload.WriteString("Cover Art\x00")
	apicPayload.Write(pngPayload)

	payloadBytes := apicPayload.Bytes()
	frameLen := len(payloadBytes)

	// In ID3v2.4, frame size is syncsafe (7 bits per byte)
	syncsafeLen := []byte{
		byte((frameLen >> 21) & 0x7F),
		byte((frameLen >> 14) & 0x7F),
		byte((frameLen >> 7) & 0x7F),
		byte(frameLen & 0x7F),
	}

	var tag bytes.Buffer
	// ID3v2.4 Header (10 bytes)
	tag.WriteString("ID3")
	tag.WriteByte(4) // v2.4
	tag.WriteByte(0) // revision
	tag.WriteByte(0) // flags
	tag.Write([]byte{0, 0, 0x10, 0}) // tag size syncsafe

	// APIC Frame Header (10 bytes)
	tag.WriteString("APIC")
	tag.Write(syncsafeLen)
	tag.Write([]byte{0, 0}) // flags
	tag.Write(payloadBytes)

	// Trailing audio / frame data
	tag.WriteString("TRAILING_AUDIO_DATA_OR_NEXT_FRAME")

	extracted, mime, err := ExtractID3Picture(tag.Bytes())
	if err != nil {
		t.Fatalf("ExtractID3Picture failed: %v", err)
	}
	if mime != "image/png" {
		t.Fatalf("Expected image/png, got %s", mime)
	}
	if !bytes.Equal(extracted, pngPayload) {
		t.Fatalf("Extracted payload did not match expected PNG (len %d vs %d)", len(extracted), len(pngPayload))
	}
}

func TestArtworkService_ExtractID3Picture_JPEG_V23(t *testing.T) {
	testJPEG := createTestJPEG(50, 50)

	var apicPayload bytes.Buffer
	apicPayload.WriteByte(0)
	apicPayload.WriteString("image/jpeg\x00")
	apicPayload.WriteByte(3)
	apicPayload.WriteString("Cover\x00")
	apicPayload.Write(testJPEG)

	payloadBytes := apicPayload.Bytes()
	frameLen := uint32(len(payloadBytes))

	var tag bytes.Buffer
	// ID3v2.3 Header (10 bytes)
	tag.WriteString("ID3")
	tag.WriteByte(3) // v2.3
	tag.WriteByte(0)
	tag.WriteByte(0)
	tag.Write([]byte{0, 0, 0x10, 0})

	// APIC Frame Header (10 bytes) - standard 32-bit big endian length
	tag.WriteString("APIC")
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], frameLen)
	tag.Write(lenBuf[:])
	tag.Write([]byte{0, 0})
	tag.Write(payloadBytes)
	tag.WriteString("TRAILING_MP3_AUDIO_STREAM_BYTES")

	extracted, mime, err := ExtractID3Picture(tag.Bytes())
	if err != nil {
		t.Fatalf("ExtractID3Picture failed: %v", err)
	}
	if mime != "image/jpeg" {
		t.Fatalf("Expected image/jpeg, got %s", mime)
	}
	if !bytes.Equal(extracted, testJPEG) {
		t.Fatalf("Extracted JPEG did not match expected (len %d vs %d)", len(extracted), len(testJPEG))
	}
}

func TestArtworkService_IsArtworkTruncated_ID3(t *testing.T) {
	svc := NewArtworkService()

	// Truncated: ID3 tag header declared 10,000 bytes, but slice only has 50 bytes
	var truncatedID3 bytes.Buffer
	truncatedID3.WriteString("ID3")
	truncatedID3.WriteByte(3) // v2.3
	truncatedID3.WriteByte(0)
	truncatedID3.WriteByte(0)
	// Syncsafe size 10000: 0x00, 0x00, 0x4e, 0x10
	truncatedID3.Write([]byte{0, 0, 0x4e, 0x10})
	truncatedID3.WriteString(strings.Repeat("A", 40))

	if !svc.IsArtworkTruncated(truncatedID3.Bytes(), "mp3") {
		t.Fatal("Expected IsArtworkTruncated to return true for truncated ID3 tag")
	}

	// Not truncated: complete ID3 tag with 10 bytes content and isLast
	var completeID3 bytes.Buffer
	completeID3.WriteString("ID3")
	completeID3.WriteByte(3)
	completeID3.WriteByte(0)
	completeID3.WriteByte(0)
	completeID3.Write([]byte{0, 0, 0, 10}) // size 10
	completeID3.WriteString("0123456789")

	if svc.IsArtworkTruncated(completeID3.Bytes(), "mp3") {
		t.Fatal("Expected IsArtworkTruncated to return false for complete ID3 tag with no APIC")
	}
}

func TestArtworkService_IsArtworkTruncated_FLAC(t *testing.T) {
	svc := NewArtworkService()

	// Truncated FLAC: Block type 6 (PICTURE), length 5000, slice only has 20 bytes
	var truncatedFLAC bytes.Buffer
	truncatedFLAC.WriteString("fLaC")
	// Block header: byte 0 = 6 (PICTURE, not last), bytes 1-3 = 5000 (0x00, 0x13, 0x88)
	truncatedFLAC.Write([]byte{6, 0x00, 0x13, 0x88})
	truncatedFLAC.WriteString(strings.Repeat("X", 16))

	if !svc.IsArtworkTruncated(truncatedFLAC.Bytes(), "flac") {
		t.Fatal("Expected IsArtworkTruncated to return true for truncated FLAC picture block")
	}

	// Clean FLAC: Block type 0 (STREAMINFO, isLast = true), length 34 bytes, followed by data
	var completeFLAC bytes.Buffer
	completeFLAC.WriteString("fLaC")
	// Header: 0x80 | 0 = 0x80 (isLast = true, type = 0)
	completeFLAC.Write([]byte{0x80, 0x00, 0x00, 34})
	completeFLAC.Write(make([]byte, 34))

	if svc.IsArtworkTruncated(completeFLAC.Bytes(), "flac") {
		t.Fatal("Expected IsArtworkTruncated to return false for clean FLAC metadata with no picture")
	}
}

func TestArtworkService_IsArtworkTruncated_MP4(t *testing.T) {
	svc := NewArtworkService()

	// Truncated MP4 covr: covr atom present with data atom requiring 8000 bytes, but slice is small
	var truncatedMP4 bytes.Buffer
	truncatedMP4.WriteString("ftypM4A ")
	truncatedMP4.WriteString("moov")
	truncatedMP4.WriteString("covr")
	// 4 bytes length before data
	truncatedMP4.Write([]byte{0, 0, 0x20, 0}) // data atom length = 8192
	truncatedMP4.WriteString("data")
	truncatedMP4.Write([]byte{0, 0, 0, 13}) // jpeg
	truncatedMP4.Write([]byte{0, 0, 0, 0})
	truncatedMP4.WriteString("short data")

	if !svc.IsArtworkTruncated(truncatedMP4.Bytes(), "alac") {
		t.Fatal("Expected IsArtworkTruncated to return true for truncated MP4 covr atom")
	}

	// Complete MP4: covr atom is complete
	testImg := createTestJPEG(40, 40)
	completeMP4 := createTestMP4CovrChunk(testImg)
	if svc.IsArtworkTruncated(completeMP4, "alac") {
		t.Fatal("Expected IsArtworkTruncated to return false for complete MP4 covr")
	}
}

func TestArtworkService_DimensionsCap(t *testing.T) {
	testImg := createTestJPEG(1200, 1200)
	svc := NewArtworkService()

	// 1. Big cover max 1000x1000
	bigWebP, err := svc.CompressToWebP(context.Background(), testImg, 1000, 80)
	if err != nil {
		t.Fatalf("Failed to compress 1000x1000 master WebP: %v", err)
	}
	if len(bigWebP) == 0 {
		t.Fatal("Expected non-empty big cover WebP")
	}

	// 2. Preview cover max 200x200
	prevWebP, err := svc.CompressToWebP(context.Background(), testImg, 200, 75)
	if err != nil {
		t.Fatalf("Failed to compress 200x200 preview WebP: %v", err)
	}
	if len(prevWebP) == 0 {
		t.Fatal("Expected non-empty preview WebP")
	}

	// Preview WebP must be significantly smaller than big WebP
	if len(prevWebP) >= len(bigWebP) {
		t.Fatalf("Expected 200x200 preview (%d bytes) to be smaller than 1000x1000 master (%d bytes)", len(prevWebP), len(bigWebP))
	}
}
