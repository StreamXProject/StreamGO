package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"

	"streamgo/internal/config"
	"streamgo/internal/logger"
)

var log = logger.New("telegram")

// Service encapsulates the gotd MTProto client lifecycle.
type Service struct {
	Config     *config.Config
	Client     *telegram.Client
	API        *tg.Client
	Downloader *downloader.Downloader

	mu         sync.RWMutex
	ready      bool
	self       *tg.User
	stopCancel context.CancelFunc
}

// New creates an unstarted Telegram service instance.
func New(cfg *config.Config) (*Service, error) {
	if cfg.ApiID <= 0 || cfg.ApiHash == "" {
		return nil, errors.New("API_ID and API_HASH are required to initialize Telegram client")
	}

	opts := telegram.Options{
		NoUpdates: false,
	}

	client := telegram.NewClient(cfg.ApiID, cfg.ApiHash, opts)

	return &Service{
		Config:     cfg,
		Client:     client,
		API:        client.API(),
		Downloader: downloader.NewDownloader(),
	}, nil
}

// Start launches the Telegram client loop in the background and authenticates.
func (s *Service) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.stopCancel = cancel

	readyChan := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		err := s.Client.Run(ctx, func(ctx context.Context) error {
			// Handle authentication
			authStatus, err := s.Client.Auth().Status(ctx)
			if err != nil {
				return fmt.Errorf("failed to check auth status: %w", err)
			}

			if !authStatus.Authorized {
				if s.Config.BotToken != "" {
					log.Info("authorizing with BOT_TOKEN...")
					if _, err := s.Client.Auth().Bot(ctx, s.Config.BotToken); err != nil {
						return fmt.Errorf("bot authentication failed: %w", err)
					}
					log.Info("bot authentication successful!")
				} else {
					log.Warn("client is not authorized and no BOT_TOKEN provided")
				}
			}

			// Get authenticated user info
			self, err := s.Client.Self(ctx)
			if err == nil && self != nil {
				s.mu.Lock()
				s.self = self
				s.ready = true
				s.mu.Unlock()
				log.Infof("connected as: %s (ID: %d, Bot: %v)", self.FirstName, self.ID, self.Bot)
			}

			close(readyChan)

			// Keep client running until context cancellation
			<-ctx.Done()
			return ctx.Err()
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			errChan <- err
		}
	}()

	// Wait for connection or timeout
	select {
	case <-readyChan:
		return nil
	case err := <-errChan:
		return err
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for telegram connection")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsReady returns true if the client is connected and authorized.
func (s *Service) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

// Self returns the authenticated Telegram user/bot profile.
func (s *Service) Self() *tg.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.self
}

// Stop terminates the Telegram background loop.
func (s *Service) Stop() {
	if s.stopCancel != nil {
		log.Info("stopping client...")
		s.stopCancel()
	}
}
