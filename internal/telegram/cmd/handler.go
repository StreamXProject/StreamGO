package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/entity"
	tgHtml "github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/tg"

	"streamgo/internal/config"
	"streamgo/internal/database"
	"streamgo/internal/models"
	"streamgo/internal/telegram"
)

// AuthHelper defines authentication methods required by bot commands.
type AuthHelper interface {
	AuthenticateOrRegisterTgUser(ctx context.Context, tgUserID int64, name, username, photoURL, inviteCode string) (*models.User, string, error)
	ConfirmBotSession(ctx context.Context, sessionID, token string, user *models.User) error
}

// AccessControlHelper defines access policy and administration methods required by bot commands.
type AccessControlHelper interface {
	GetPolicy(ctx context.Context) (*models.AccessPolicy, error)
	UpdatePolicy(ctx context.Context, patch models.PolicyPatch, adminID int64) (*models.AccessPolicy, error)
	AddRequiredChat(ctx context.Context, chat models.RequiredChat) (*models.AccessPolicy, error)
	RemoveRequiredChat(ctx context.Context, chatID int64) (*models.AccessPolicy, error)
	CreateInvite(ctx context.Context, adminID int64, maxUses int, ttlDays int, note string) (*models.InviteCode, error)
	ListInvites(ctx context.Context, includeDead bool) ([]*models.InviteCode, error)
	LockUser(ctx context.Context, userID int64, reason string, adminID int64, revokeSessions bool) error
	UnlockUser(ctx context.Context, userID int64) error
	RevokeSessions(ctx context.Context, userID int64) (int, error)
	AddToBypass(ctx context.Context, userID int64, adminID int64, note string) error
	RemoveFromBypass(ctx context.Context, userID int64) bool
	ListBypass(ctx context.Context) ([]*models.AllowlistEntry, error)
}

// AccessFilterHelper defines source allow/ban and filter mode methods required by bot commands.
type AccessFilterHelper interface {
	GetStats() (mode int, modeName string, allowedCount int, bannedCount int, channelID int64)
	GetMode() int
	SetMode(mode int)
	AllowChat(ctx context.Context, chatID int64) error
	DisallowChat(ctx context.Context, chatID int64) error
	BanChat(ctx context.Context, id int64, reason string) error
	UnbanChat(ctx context.Context, id int64) error
}

// TrackSearchHelper defines track search queries for /search command.
type TrackSearchHelper interface {
	Search(ctx context.Context, query string, limit int) ([]*models.BrowseItem, error)
}

// Handler coordinates Telegram bot command routing and interactive inline queries.
type Handler struct {
	cfg          *config.Config
	db           *database.Client
	tgService    *telegram.Service
	listener     *telegram.IngestionListener
	authSvc      AuthHelper
	accessCtrl   AccessControlHelper
	accessFilter AccessFilterHelper
	trackSvc     TrackSearchHelper
	startTime    time.Time
}

// New creates a new command Handler.
func New(
	cfg *config.Config,
	db *database.Client,
	tgService *telegram.Service,
	listener *telegram.IngestionListener,
	authSvc AuthHelper,
	accessCtrl AccessControlHelper,
	accessFilter AccessFilterHelper,
	trackSvc TrackSearchHelper,
) *Handler {
	return &Handler{
		cfg:          cfg,
		db:           db,
		tgService:    tgService,
		listener:     listener,
		authSvc:      authSvc,
		accessCtrl:   accessCtrl,
		accessFilter: accessFilter,
		trackSvc:     trackSvc,
		startTime:    time.Now(),
	}
}

// SetupDispatcher binds message routers and inline callback query handlers to the update dispatcher.
func (h *Handler) SetupDispatcher(d *tg.UpdateDispatcher) {
	d.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		if msg, ok := u.Message.(*tg.Message); ok {
			h.routeMessage(ctx, e, u, msg)
		}
		return nil
	})

	d.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		if msg, ok := u.Message.(*tg.Message); ok {
			h.routeMessage(ctx, e, u, msg)
		}
		return nil
	})

	d.OnBotCallbackQuery(func(ctx context.Context, e tg.Entities, u *tg.UpdateBotCallbackQuery) error {
		h.handleCallbackQuery(ctx, e, u)
		return nil
	})
}

// routeMessage routes an incoming message to the matching command or audio listener.
func (h *Handler) routeMessage(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, msg *tg.Message) {
	if msg == nil {
		return
	}

	text := strings.TrimSpace(msg.Message)

	// Check if this message is an audio upload
	isAudio := false
	if msg.Media != nil {
		if docMedia, ok := msg.Media.(*tg.MessageMediaDocument); ok && docMedia.Document != nil {
			if doc, ok := docMedia.Document.(*tg.Document); ok {
				mime := strings.ToLower(doc.MimeType)
				hasAudioAttr := false
				for _, attr := range doc.Attributes {
					if _, ok := attr.(*tg.DocumentAttributeAudio); ok {
						hasAudioAttr = true
						break
					}
				}
				if strings.HasPrefix(mime, "audio/") || hasAudioAttr {
					isAudio = true
				}
			}
		}
	}

	// Normal text / audio uploads without a leading '/' command are forwarded to ingestion listener
	if !strings.HasPrefix(text, "/") {
		if isAudio && h.listener != nil {
			h.listener.HandleMessage(ctx, msg)
		}
		return
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}

	rawCmd := parts[0][1:] // strip '/'
	cmdName := strings.ToLower(strings.Split(rawCmd, "@")[0])
	args := parts[1:]

	chatID, senderID, _, _ := h.extractPeerInfo(msg, e)

	// Ingest audio concurrently if command is not a media inspector
	if isAudio && h.listener != nil && cmdName != "mediainfo" && cmdName != "mi" {
		h.listener.HandleMessage(ctx, msg)
	}

	switch cmdName {
	case "start", "help":
		h.handleStartOrHelp(ctx, e, upd, msg, args)
	case "ping", "alive":
		h.handlePing(ctx, e, upd, msg)
	case "id":
		h.handleID(ctx, e, upd, msg, chatID, senderID)
	case "search":
		h.handleSearch(ctx, e, upd, msg, args)
	case "mediainfo", "mi":
		h.handleMediaInfo(ctx, e, upd, msg)
	case "sources":
		h.handleSources(ctx, e, upd, msg, senderID)
	case "filter_mode", "filtermode":
		h.handleFilterMode(ctx, e, upd, msg, senderID, args)
	case "allow", "allow_source", "allowsource":
		h.handleAllow(ctx, e, upd, msg, senderID, args)
	case "disallow", "disallow_source", "disallowsource":
		h.handleDisallow(ctx, e, upd, msg, senderID, args)
	case "ban", "ban_source", "bansources", "bansource":
		h.handleBan(ctx, e, upd, msg, senderID, args)
	case "unban", "unban_source", "unbansources", "unbansource":
		h.handleUnban(ctx, e, upd, msg, senderID, args)
	case "access":
		h.handleAccess(ctx, e, upd, msg, senderID)
	case "registration":
		h.handleRegistration(ctx, e, upd, msg, senderID, args)
	case "membership":
		h.handleMembership(ctx, e, upd, msg, senderID, args)
	case "requirechat":
		h.handleRequireChat(ctx, e, upd, msg, senderID, args)
	case "unrequirechat":
		h.handleUnrequireChat(ctx, e, upd, msg, senderID, args)
	case "invite":
		h.handleInvite(ctx, e, upd, msg, senderID, args)
	case "invites":
		h.handleInvites(ctx, e, upd, msg, senderID)
	case "lock":
		h.handleLock(ctx, e, upd, msg, senderID, args)
	case "unlock":
		h.handleUnlock(ctx, e, upd, msg, senderID, args)
	case "revoke":
		h.handleRevoke(ctx, e, upd, msg, senderID, args)
	case "bypass", "exempt":
		h.handleBypass(ctx, e, upd, msg, senderID, args)
	case "unbypass", "unexempt":
		h.handleUnbypass(ctx, e, upd, msg, senderID, args)
	case "bypasses", "bypassed":
		h.handleBypasses(ctx, e, upd, msg, senderID)
	case "fileid":
		h.handleFileID(ctx, e, upd, msg, senderID)
	case "log", "logs":
		h.handleLogs(ctx, e, upd, msg, senderID)
	case "sudo":
		h.handleSudo(ctx, e, upd, msg, senderID)
	case "restart":
		h.handleRestart(ctx, e, upd, msg, senderID)
	}
}

func (h *Handler) isAdmin(userID int64) bool {
	if h.cfg == nil {
		return false
	}
	for _, id := range h.cfg.OwnerIDs {
		if id == userID {
			return true
		}
	}
	for _, id := range h.cfg.SudoUsers {
		if id == userID {
			return true
		}
	}
	return false
}

func (h *Handler) extractPeerInfo(msg *tg.Message, e tg.Entities) (chatID int64, senderID int64, senderName string, senderUsername string) {
	switch p := msg.PeerID.(type) {
	case *tg.PeerChannel:
		chatID = -1000000000000 - p.ChannelID
	case *tg.PeerChat:
		chatID = -p.ChatID
	case *tg.PeerUser:
		chatID = p.UserID
	}

	if msg.FromID != nil {
		switch p := msg.FromID.(type) {
		case *tg.PeerUser:
			senderID = p.UserID
		case *tg.PeerChannel:
			senderID = -1000000000000 - p.ChannelID
		}
	} else if chatID > 0 {
		senderID = chatID
	}

	if u, ok := e.Users[senderID]; ok {
		senderName = u.FirstName
		if u.LastName != "" {
			senderName = strings.TrimSpace(senderName + " " + u.LastName)
		}
		senderUsername = u.Username
	}

	return
}

func (h *Handler) replyHTML(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, htmlContent string, replyMarkup tg.ReplyMarkupClass) error {
	w := h.tgService.PrimaryWorker()
	if w == nil || w.API == nil {
		return errors.New("no ready Telegram worker")
	}

	sender := message.NewSender(w.API)
	b := sender.Reply(e, upd)
	if replyMarkup != nil {
		b = b.Markup(replyMarkup)
	}

	_, err := b.StyledText(ctx, tgHtml.String(nil, htmlContent))
	if err != nil {
		// Fallback to sending as answer to the chat if reply fails
		b2 := sender.Answer(e, upd).CloneBuilder()
		if replyMarkup != nil {
			b2 = b2.Markup(replyMarkup)
		}
		_, err = b2.StyledText(ctx, tgHtml.String(nil, htmlContent))
	}
	return err
}

func (h *Handler) editMessage(ctx context.Context, peer tg.PeerClass, e tg.Entities, msgID int, htmlContent string, replyMarkup tg.ReplyMarkupClass) error {
	w := h.tgService.PrimaryWorker()
	if w == nil || w.API == nil {
		return errors.New("no ready Telegram worker")
	}

	inputPeer := extractInputPeer(peer, e, w)
	if inputPeer == nil {
		return errors.New("could not resolve input peer")
	}

	text, entities := parseHTML(htmlContent)
	req := &tg.MessagesEditMessageRequest{
		Peer:        inputPeer,
		ID:          msgID,
		Message:     text,
		Entities:    entities,
		ReplyMarkup: replyMarkup,
		NoWebpage:   true,
	}

	_, err := w.API.MessagesEditMessage(ctx, req)
	return err
}

func extractInputPeer(p tg.PeerClass, e tg.Entities, w *telegram.ClientWorker) tg.InputPeerClass {
	if p == nil {
		return nil
	}
	switch peer := p.(type) {
	case *tg.PeerUser:
		if u, ok := e.Users[peer.UserID]; ok {
			return u.AsInputPeer()
		}
		return &tg.InputPeerUser{UserID: peer.UserID}
	case *tg.PeerChat:
		if c, ok := e.Chats[peer.ChatID]; ok {
			return c.AsInputPeer()
		}
		return &tg.InputPeerChat{ChatID: peer.ChatID}
	case *tg.PeerChannel:
		if ch, ok := e.Channels[peer.ChannelID]; ok {
			return ch.AsInputPeer()
		}
		accessHash := int64(0)
		if w != nil {
			accessHash = w.GetChannelAccessHash(peer.ChannelID)
		}
		return &tg.InputPeerChannel{ChannelID: peer.ChannelID, AccessHash: accessHash}
	}
	return nil
}

func parseHTML(raw string) (string, []tg.MessageEntityClass) {
	var b entity.Builder
	if err := tgHtml.HTML(strings.NewReader(raw), &b, tgHtml.Options{}); err != nil {
		return raw, nil
	}
	return b.Complete()
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	days := seconds / 86400
	remainder := seconds % 86400
	hours := remainder / 3600
	remainder = remainder % 3600
	minutes := remainder / 60
	secs := remainder % 60

	var result string
	if days != 0 {
		result += fmt.Sprintf("%dd ", days)
	}
	if hours != 0 {
		result += fmt.Sprintf("%dh ", hours)
	}
	if minutes != 0 {
		result += fmt.Sprintf("%dm ", minutes)
	}
	result += fmt.Sprintf("%ds ", secs)
	return strings.TrimSpace(result)
}

func formatSeconds(sec int32) string {
	if sec <= 0 {
		return "0:00"
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
