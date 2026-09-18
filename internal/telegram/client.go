package telegram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"

	"streamgo/internal/config"
	"streamgo/internal/logger"
)

var log = logger.New("telegram")

// ClientWorker represents a single authenticated MTProto client connection.
type ClientWorker struct {
	ID         int64
	FirstName  string
	Username   string
	Bot        bool
	Token      string
	Client     *telegram.Client
	API        *tg.Client
	Downloader *downloader.Downloader
	Workload   int64
	Ready      bool
}

// Service encapsulates the MTProto multi-client pool and lifecycle.
type Service struct {
	Config         *config.Config
	primaryWorker  *ClientWorker
	workers        []*ClientWorker
	mu             sync.RWMutex
	stopCancel     context.CancelFunc
	Dispatcher     tg.UpdateDispatcher
}

// New creates an unstarted Telegram multi-client pool.
func New(cfg *config.Config) (*Service, error) {
	if cfg.ApiID <= 0 || cfg.ApiHash == "" {
		return nil, errors.New("API_ID and API_HASH are required to initialize Telegram client")
	}

	dispatcher := tg.NewUpdateDispatcher()

	svc := &Service{
		Config:     cfg,
		workers:    make([]*ClientWorker, 0),
		Dispatcher: dispatcher,
	}

	// 1. Create Primary Worker
	primaryClient := telegram.NewClient(cfg.ApiID, cfg.ApiHash, telegram.Options{
		NoUpdates:     false,
		UpdateHandler: dispatcher,
	})
	svc.primaryWorker = &ClientWorker{
		Token:      cfg.BotToken,
		Client:     primaryClient,
		API:        primaryClient.API(),
		Downloader: downloader.NewDownloader(),
	}
	svc.workers = append(svc.workers, svc.primaryWorker)


	// 2. Create Secondary Multi-Client Workers
	if cfg.MultiClients {
		for i, tok := range cfg.MultiClientTokens {
			tok = strings.TrimSpace(tok)
			if tok == "" || tok == cfg.BotToken {
				continue
			}
			workerClient := telegram.NewClient(cfg.ApiID, cfg.ApiHash, telegram.Options{
				NoUpdates: true, // Secondary download workers don't need update processing
			})
			worker := &ClientWorker{
				Token:      tok,
				Client:     workerClient,
				API:        workerClient.API(),
				Downloader: downloader.NewDownloader(),
			}
			svc.workers = append(svc.workers, worker)
			log.Infof("Registered multi-client worker #%d", i+1)
		}
	}

	return svc, nil
}

// Start launches all MTProto client workers concurrently.
func (s *Service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.stopCancel = cancel

	readyChan := make(chan struct{})
	var once sync.Once

	for _, w := range s.workers {
		worker := w
		go func() {
			_ = worker.Client.Run(ctx, func(ctx context.Context) error {
				// Authenticate
				authStatus, err := worker.Client.Auth().Status(ctx)
				if err != nil {
					log.Warnf("Worker check auth failed: %v", err)
					return err
				}

				if !authStatus.Authorized && worker.Token != "" {
					log.Infof("Authorizing client with token (prefix: %s...)", worker.Token[:min(10, len(worker.Token))])
					if _, err := worker.Client.Auth().Bot(ctx, worker.Token); err != nil {
						log.Warnf("Worker bot auth failed: %v", err)
						return err
					}
					log.Info("Worker authentication successful!")
				}

				self, err := worker.Client.Self(ctx)
				if err == nil && self != nil {
					s.mu.Lock()
					worker.ID = self.ID
					worker.FirstName = self.FirstName
					worker.Username = self.Username
					worker.Bot = self.Bot
					worker.Ready = true
					s.mu.Unlock()
					log.Infof("Connected client: %s (@%s, ID: %d, Bot: %v)", self.FirstName, self.Username, self.ID, self.Bot)
				}

				if worker == s.primaryWorker {
					once.Do(func() {
						close(readyChan)
					})
				}

				<-ctx.Done()
				return ctx.Err()
			})
		}()
	}

	// Wait up to 25 seconds for primary client readiness
	select {
	case <-readyChan:
		log.Infof("Telegram service active with %d client(s) ready", s.ReadyWorkerCount())
		return nil
	case <-time.After(25 * time.Second):
		log.Warn("Telegram service started with delayed initialization")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReadyWorkerCount returns number of connected and ready workers.
func (s *Service) ReadyWorkerCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, w := range s.workers {
		if w.Ready {
			count++
		}
	}
	return count
}

// PrimaryWorker returns the main bot worker.
func (s *Service) PrimaryWorker() *ClientWorker {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.primaryWorker
}

// AcquireWorker selects the best ready worker with the lowest workload (load-balancing).
func (s *Service) AcquireWorker() *ClientWorker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *ClientWorker
	minLoad := int64(1<<62 - 1)

	for _, w := range s.workers {
		if !w.Ready {
			continue
		}
		load := atomic.LoadInt64(&w.Workload)
		if best == nil || load < minLoad {
			best = w
			minLoad = load
		}
	}

	if best == nil {
		best = s.primaryWorker
	}

	if best != nil {
		atomic.AddInt64(&best.Workload, 1)
	}
	return best
}

// AcquireWorkerForBot attempts to acquire a specific worker by Telegram bot user ID.
func (s *Service) AcquireWorkerForBot(botID string) *ClientWorker {
	botID = strings.TrimSpace(botID)
	if botID != "" {
		s.mu.RLock()
		for _, w := range s.workers {
			if w.Ready && strconv.FormatInt(w.ID, 10) == botID {
				s.mu.RUnlock()
				atomic.AddInt64(&w.Workload, 1)
				return w
			}
		}
		s.mu.RUnlock()
	}
	return s.AcquireWorker()
}

// ReleaseWorker decrements active download workload for a worker.
func (s *Service) ReleaseWorker(w *ClientWorker) {
	if w != nil {
		atomic.AddInt64(&w.Workload, -1)
	}
}

// Stop gracefully stops all Telegram client workers.
func (s *Service) Stop() {
	if s.stopCancel != nil {
		s.stopCancel()
	}
}

// IsReady returns true if at least the primary client is connected.
func (s *Service) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.primaryWorker != nil && s.primaryWorker.Ready
}

// Self returns primary bot user info.
func (s *Service) Self() *tg.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.primaryWorker != nil && s.primaryWorker.Ready {
		return &tg.User{
			ID:        s.primaryWorker.ID,
			FirstName: s.primaryWorker.FirstName,
			Username:  s.primaryWorker.Username,
			Bot:       s.primaryWorker.Bot,
		}
	}
	return nil
}

// StatusString returns summary of connected bot clients for /health.
func (s *Service) StatusString() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	readyWorkers := 0
	var names []string
	for _, w := range s.workers {
		if w.Ready {
			readyWorkers++
			if w.Username != "" {
				names = append(names, fmt.Sprintf("@%s", w.Username))
			} else if w.FirstName != "" {
				names = append(names, w.FirstName)
			}
		}
	}

	if readyWorkers == 0 {
		return "connecting"
	}
	return fmt.Sprintf("connected (%s)", strings.Join(names, ", "))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type partialWriter struct {
	w      io.Writer
	remain int64
}

func (pw *partialWriter) Write(p []byte) (int, error) {
	if pw.remain <= 0 {
		return 0, io.EOF
	}

	toWrite := p
	if int64(len(toWrite)) > pw.remain {
		toWrite = toWrite[:pw.remain]
	}

	n, err := pw.w.Write(toWrite)
	pw.remain -= int64(n)
	if err != nil {
		return n, err
	}
	if pw.remain <= 0 {
		return n, io.EOF
	}
	return n, nil
}

// DownloadPartial streams up to maxBytes of a document location into w.
func (s *Service) DownloadPartial(ctx context.Context, location *tg.InputDocumentFileLocation, maxBytes int64, w io.Writer) error {
	worker := s.AcquireWorker()
	defer s.ReleaseWorker(worker)

	pw := &partialWriter{
		w:      w,
		remain: maxBytes,
	}

	downloader := worker.Client.Downloader()
	_, err := downloader.Download(worker.API, location).Stream(ctx, pw)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// DownloadPartialByFileID decodes a Pyrogram/TDLib file_id and downloads up to maxBytes.
func (s *Service) DownloadPartialByFileID(ctx context.Context, fileID string, maxBytes int64, w io.Writer) error {
	decoded, err := DecodeFileID(fileID)
	if err != nil {
		return fmt.Errorf("failed to decode file_id: %w", err)
	}

	location := &tg.InputDocumentFileLocation{
		ID:            decoded.MediaID,
		AccessHash:    decoded.AccessHash,
		FileReference: decoded.FileReference,
	}

	return s.DownloadPartial(ctx, location, maxBytes, w)
}
