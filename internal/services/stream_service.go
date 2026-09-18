package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/tg"

	"streamgo/internal/logger"
	"streamgo/internal/models"
	"streamgo/internal/repository"
	"streamgo/internal/telegram"
)

var logStream = logger.New("stream")

var rangeRegex = regexp.MustCompile(`^bytes=(\d*)-(\d*)$`)

// StreamService coordinates audio range parsing and streaming.
type StreamService struct {
	repo      repository.TrackRepository
	tgService *telegram.Service
	mediaDir  string
}

// NewStreamService creates a new StreamService.
func NewStreamService(repo repository.TrackRepository, tg *telegram.Service) *StreamService {
	mediaDir := "stream_media"
	_ = os.MkdirAll(mediaDir, 0755)
	_ = os.MkdirAll(filepath.Join(mediaDir, "alac_cache"), 0755)

	return &StreamService{
		repo:      repo,
		tgService: tg,
		mediaDir:  mediaDir,
	}
}

// ByteRange defines start and end byte offsets.
type ByteRange struct {
	Start  int64
	End    int64
	Length int64
}

// ParseRange parses an HTTP Range header string into byte boundaries.
func ParseRange(rangeHeader string, totalSize int64) (*ByteRange, error) {
	rangeHeader = strings.TrimSpace(rangeHeader)
	if rangeHeader == "" {
		return nil, nil
	}

	matches := rangeRegex.FindStringSubmatch(rangeHeader)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid range format: %s", rangeHeader)
	}

	startStr := matches[1]
	endStr := matches[2]

	var start, end int64

	if startStr == "" && endStr == "" {
		return nil, fmt.Errorf("empty range boundaries")
	}

	if startStr == "" {
		n, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid range offset: %s", endStr)
		}
		start = totalSize - n
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
	} else {
		var err error
		start, err = strconv.ParseInt(startStr, 10, 64)
		if err != nil || start < 0 {
			return nil, fmt.Errorf("invalid range start: %s", startStr)
		}

		if endStr == "" {
			end = totalSize - 1
		} else {
			end, err = strconv.ParseInt(endStr, 10, 64)
			if err != nil || end < start {
				return nil, fmt.Errorf("invalid range end: %s", endStr)
			}
			if end >= totalSize {
				end = totalSize - 1
			}
		}
	}

	if start >= totalSize {
		return nil, fmt.Errorf("range start %d out of bounds (total: %d)", start, totalSize)
	}

	return &ByteRange{
		Start:  start,
		End:    end,
		Length: end - start + 1,
	}, nil
}

// StreamTrack handles streaming audio for a track ID with HTTP Range support.
func (s *StreamService) StreamTrack(w http.ResponseWriter, r *http.Request, trackID string) {
	ctx := r.Context()

	track, err := s.repo.GetByID(ctx, trackID)
	if err != nil {
		http.Error(w, "Failed to retrieve track", http.StatusInternalServerError)
		return
	}
	if track == nil {
		http.Error(w, "Track not found", http.StatusNotFound)
		return
	}

	// 1. Determine total size
	totalSize := track.Audio.FileSize
	if totalSize <= 0 {
		totalSize = track.Telegram.FileSize
	}
	if totalSize <= 0 {
		totalSize = 10 * 1024 * 1024
	}

	// 2. Determine mime type
	mimeType := track.Audio.MimeType
	if mimeType == "" {
		mimeType = track.Telegram.MimeType
	}
	if mimeType == "" {
		mimeType = "audio/mpeg"
	}

	// 3. Clean filename for Content-Disposition
	filename := track.Telegram.FileName
	if filename == "" {
		filename = fmt.Sprintf("%s.mp3", track.ID)
	}

	// Check if local cache file exists (e.g. decoded ALAC/FLAC)
	cacheFile := filepath.Join(s.mediaDir, "alac_cache", fmt.Sprintf("%s.flac", track.ID))
	if info, err := os.Stat(cacheFile); err == nil && info.Size() > 10240 {
		f, err := os.Open(cacheFile)
		if err == nil {
			defer f.Close()
			w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
			http.ServeContent(w, r, filename, info.ModTime(), f)
			return
		}
	}

	// 4. Handle Range Header
	rangeHeader := r.Header.Get("Range")
	var byteRange *ByteRange
	if rangeHeader != "" {
		var err error
		byteRange, err = ParseRange(rangeHeader, totalSize)
		if err != nil {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", totalSize))
			http.Error(w, "Requested Range Not Satisfiable", http.StatusRequestedRangeNotSatisfiable)
			return
		}
	}

	// 5. Setup Response Headers
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))

	status := http.StatusOK
	var contentLength int64 = totalSize

	if byteRange != nil {
		status = http.StatusPartialContent
		contentLength = byteRange.Length
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", byteRange.Start, byteRange.End, totalSize))
	}
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))

	// If HEAD request, emit headers and finish immediately
	if r.Method == http.MethodHead {
		w.WriteHeader(status)
		return
	}

	// 6. Asynchronously increment play count
	go func(tid string) {
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.repo.IncrementPlayCount(timeoutCtx, tid)
	}(track.ID)

	w.WriteHeader(status)

	// Check Telegram client connectivity
	if s.tgService == nil || !s.tgService.IsReady() {
		logStream.Warnf("Telegram client not ready for track %s", trackID)
		return
	}

	// Stream chunks directly from Telegram to HTTP Response
	s.streamFromTelegram(ctx, w, track, byteRange, totalSize)
}

type rangeWriter struct {
	w       io.Writer
	skip    int64
	remain  int64
	flusher http.Flusher
}

func (rw *rangeWriter) Write(p []byte) (int, error) {
	if rw.remain <= 0 {
		return 0, io.EOF
	}

	n := len(p)
	if rw.skip > 0 {
		if int64(n) <= rw.skip {
			rw.skip -= int64(n)
			return n, nil
		}
		p = p[rw.skip:]
		rw.skip = 0
	}

	toWrite := p
	if int64(len(toWrite)) > rw.remain {
		toWrite = toWrite[:rw.remain]
	}

	written, err := rw.w.Write(toWrite)
	rw.remain -= int64(written)

	if rw.flusher != nil {
		rw.flusher.Flush()
	}

	if err != nil {
		return written, err
	}
	if rw.remain <= 0 {
		return n, io.EOF
	}

	return n, nil
}

func (s *StreamService) streamFromTelegram(
	ctx context.Context,
	w http.ResponseWriter,
	track *models.Track,
	byteRange *ByteRange,
	totalSize int64,
) {
	var startOffset int64 = 0
	var bytesRemaining int64 = totalSize

	if byteRange != nil {
		startOffset = byteRange.Start
		bytesRemaining = byteRange.Length
	}

	logStream.Infof("streaming track %s: offset=%d, length=%d", track.ID, startOffset, bytesRemaining)

	fileID := track.Telegram.FileID
	if fileID == "" {
		logStream.Errorf("track %s has no telegram file_id", track.ID)
		return
	}

	decoded, err := telegram.DecodeFileID(fileID)
	if err != nil {
		logStream.Errorf("failed to decode file_id for track %s: %v", track.ID, err)
		return
	}

	location := &tg.InputDocumentFileLocation{
		ID:            decoded.MediaID,
		AccessHash:    decoded.AccessHash,
		FileReference: decoded.FileReference,
	}

	flusher, _ := w.(http.Flusher)
	rw := &rangeWriter{
		w:       w,
		skip:    startOffset,
		remain:  bytesRemaining,
		flusher: flusher,
	}

	downloader := s.tgService.Client.Downloader()
	_, err = downloader.Download(s.tgService.API, location).Stream(ctx, rw)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		logStream.Warnf("streaming ended with error for track %s: %v", track.ID, err)
	}
}
