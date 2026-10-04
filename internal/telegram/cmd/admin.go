package cmd

import (
	"context"
	"fmt"
	"html"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"go.mongodb.org/mongo-driver/v2/bson"

	"streamgo/internal/telegram"
)

// handleFileID extracts and encodes Telegram file_id from replied media.
func (h *Handler) handleFileID(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, msg *tg.Message, senderID int64) {
	if !h.isAdmin(senderID) {
		_ = h.replyHTML(ctx, e, upd, "Access denied.", nil)
		return
	}

	var targetDoc *tg.Document
	if msg.Media != nil {
		if docMedia, ok := msg.Media.(*tg.MessageMediaDocument); ok && docMedia.Document != nil {
			if doc, ok := docMedia.Document.(*tg.Document); ok {
				targetDoc = doc
			}
		}
	}

	if targetDoc == nil {
		_ = h.replyHTML(ctx, e, upd, "Please reply to an image, video, document, or GIF to get its file ID.", nil)
		return
	}

	primaryID := telegram.EncodeFileID(targetDoc.ID, targetDoc.AccessHash, int32(targetDoc.DCID), targetDoc.FileReference)
	reply := fmt.Sprintf("<b>File ID:</b>\n<code>%s</code>", primaryID)
	_ = h.replyHTML(ctx, e, upd, reply, nil)
}

// handleLogs outputs recent server log lines.
func (h *Handler) handleLogs(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, msg *tg.Message, senderID int64) {
	if !h.isAdmin(senderID) {
		_ = h.replyHTML(ctx, e, upd, "Access denied.", nil)
		return
	}

	logPaths := []string{"log.txt", "serverlogs.txt"}
	var lines []string

	for _, p := range logPaths {
		if data, err := os.ReadFile(p); err == nil {
			all := strings.Split(string(data), "\n")
			if len(all) > 40 {
				lines = all[len(all)-40:]
			} else {
				lines = all
			}
			break
		}
	}

	if len(lines) == 0 {
		_ = h.replyHTML(ctx, e, upd, "Log file not found!", nil)
		return
	}

	snippet := strings.Join(lines, "\n")
	if len(snippet) > 990 {
		snippet = snippet[len(snippet)-990:]
	}

	reply := fmt.Sprintf("<b>Recent Logs (1,024 char limit):</b>\n<pre>%s</pre>", html.EscapeString(snippet))
	_ = h.replyHTML(ctx, e, upd, reply, nil)
}

// handleSudo displays an administrative overview of system resources, workers, and database stats.
func (h *Handler) handleSudo(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, msg *tg.Message, senderID int64) {
	if !h.isAdmin(senderID) {
		_ = h.replyHTML(ctx, e, upd, "Access denied.", nil)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	uptime := time.Since(h.startTime)

	var trackCount, userCount int64
	if h.db != nil {
		trackCount, _ = h.db.Collection("audioTracks").CountDocuments(ctx, bson.M{})
		userCount, _ = h.db.Collection("users").CountDocuments(ctx, bson.M{})
	}

	readyWorkers := 0
	primaryBot := "none"
	if h.tgService != nil {
		readyWorkers = h.tgService.ReadyWorkerCount()
		if self := h.tgService.Self(); self != nil {
			primaryBot = "@" + self.Username
		}
	}

	text := fmt.Sprintf(
		"<b>Admin Panel:</b>\n\n"+
			"• <b>Go Version:</b> <code>%s</code>\n"+
			"• <b>OS/Arch:</b> <code>%s/%s</code>\n"+
			"• <b>Goroutines:</b> %d\n"+
			"• <b>RAM:</b> %.2f MB\n"+
			"• <b>Uptime:</b> %s\n\n"+
			"• <b>Bot:</b> %s (Workers: %d ready)\n"+
			"• <b>Tracks:</b> %d\n"+
			"• <b>Users:</b> %d",
		runtime.Version(), runtime.GOOS, runtime.GOARCH,
		runtime.NumGoroutine(), float64(m.Alloc)/(1024*1024),
		formatDuration(uptime),
		primaryBot, readyWorkers,
		trackCount, userCount,
	)

	kb := markup.InlineKeyboard(
		markup.InlineButtonRow(
			markup.Callback("Config", []byte("help_config_1")),
			markup.Callback("Cookies", []byte("help_noop")),
		),
		markup.InlineButtonRow(
			markup.Callback("Stats", []byte("help_noop")),
			markup.Callback("Database", []byte("help_noop")),
		),
	)

	_ = h.replyHTML(ctx, e, upd, text, kb)
}

// handleRestart initiates restart flow with confirmation buttons matching StreamXBot.
func (h *Handler) handleRestart(ctx context.Context, e tg.Entities, upd message.AnswerableMessageUpdate, msg *tg.Message, senderID int64) {
	if !h.isAdmin(senderID) {
		_ = h.replyHTML(ctx, e, upd, "Access denied.", nil)
		return
	}

	kb := markup.InlineKeyboard(
		markup.InlineButtonRow(
			markup.Callback("Restart Only", []byte("confirm_restart restart")),
			markup.Callback("Update & Restart", []byte("confirm_restart update")),
		),
		markup.InlineButtonRow(
			markup.Callback("Cancel", []byte("confirm_restart cancel")),
		),
	)

	_ = h.replyHTML(ctx, e, upd, "How would you like to proceed?", kb)
}

// handleRestartConfirmation handles button callbacks for restart confirmation.
func (h *Handler) handleRestartConfirmation(ctx context.Context, e tg.Entities, update *tg.UpdateBotCallbackQuery) {
	if update == nil {
		return
	}

	data := string(update.Data)
	parts := strings.Split(data, " ")
	choice := "restart"
	if len(parts) > 1 {
		choice = parts[1]
	}

	w := h.tgService.PrimaryWorker()
	if w == nil || w.API == nil {
		return
	}

	if choice == "cancel" {
		_, _ = w.API.MessagesSetBotCallbackAnswer(ctx, &tg.MessagesSetBotCallbackAnswerRequest{
			QueryID: update.QueryID,
			Message: "Cancelled",
		})
		_, _ = w.API.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
			ID:     []int{update.MsgID},
			Revoke: true,
		})
		return
	}

	_, _ = w.API.MessagesSetBotCallbackAnswer(ctx, &tg.MessagesSetBotCallbackAnswerRequest{
		QueryID: update.QueryID,
		Message: "Restarting...",
	})

	_ = h.editMessage(ctx, update.Peer, e, update.MsgID, "Restarting...", nil)

	go func() {
		time.Sleep(1 * time.Second)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
}
