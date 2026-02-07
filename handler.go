package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"telegram-chat-bot/db"
)

type Handler struct {
	bot     *BotClient
	storage *Storage
	botName string
}

func NewHandler(bot *BotClient, storage *Storage, botName string) *Handler {
	return &Handler{
		bot:     bot,
		storage: storage,
		botName: botName,
	}
}

func (h *Handler) HandleUpdate(ctx context.Context, update Update) {
	if update.Message == nil || update.Message.From == nil {
		return
	}

	msg := update.Message
	cmd := extractCommand(msg, h.botName)
	if cmd == "" {
		return
	}

	var err error
	switch cmd {
	case "/join":
		err = h.handleJoin(ctx, msg)
	case "/leave":
		err = h.handleLeave(ctx, msg)
	case "/roulette":
		err = h.handleRoulette(ctx, msg)
	case "/stats":
		err = h.handleStats(ctx, msg)
	case "/participants":
		err = h.handleParticipants(ctx, msg)
	case "/reset":
		err = h.handleReset(ctx, msg)
	}

	if err != nil {
		log.Printf("Error handling %s: %v", cmd, err)
	}
}

func extractCommand(msg *Message, botName string) string {
	if len(msg.Entities) == 0 {
		return ""
	}

	entity := msg.Entities[0]
	if entity.Type != "bot_command" || entity.Offset != 0 {
		return ""
	}

	text := msg.Text
	runes := []rune(text)
	if entity.Length > len(runes) {
		return ""
	}
	cmd := string(runes[:entity.Length])

	// Strip @botname suffix
	if i := strings.Index(cmd, "@"); i != -1 {
		mention := cmd[i+1:]
		if !strings.EqualFold(mention, botName) {
			return "" // Command for a different bot
		}
		cmd = cmd[:i]
	}

	return strings.ToLower(cmd)
}

func formatUserName(firstName, username string) string {
	name := html.EscapeString(firstName)
	if username != "" {
		return fmt.Sprintf("%s (@%s)", name, html.EscapeString(username))
	}
	return name
}

func (h *Handler) reply(ctx context.Context, msg *Message, text string) error {
	return h.bot.SendMessage(ctx, SendMessageRequest{
		ChatID:    msg.Chat.ID,
		Text:      text,
		ParseMode: "HTML",
		ReplyParameters: &ReplyParameters{
			MessageID: msg.MessageID,
		},
	})
}

func (h *Handler) send(ctx context.Context, chatID int64, text string) error {
	return h.bot.SendMessage(ctx, SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
}

func today() string {
	return time.Now().UTC().Format("2011-11-11")
}

func (h *Handler) handleJoin(ctx context.Context, msg *Message) error {
	user := msg.From
	err := h.storage.Queries.AddParticipant(ctx, db.AddParticipantParams{
		ChatID:    msg.Chat.ID,
		UserID:    user.ID,
		FirstName: user.FirstName,
		Username:  user.Username,
	})
	if err != nil {
		return err
	}

	text := fmt.Sprintf("%s joined the roulette!", formatUserName(user.FirstName, user.Username))
	return h.send(ctx, msg.Chat.ID, text)
}

func (h *Handler) handleLeave(ctx context.Context, msg *Message) error {
	user := msg.From
	result, err := h.storage.Queries.RemoveParticipant(ctx, db.RemoveParticipantParams{
		ChatID: msg.Chat.ID,
		UserID: user.ID,
	})
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	var text string
	if rows > 0 {
		text = fmt.Sprintf("%s left the roulette.", formatUserName(user.FirstName, user.Username))
	} else {
		text = "You're not in the roulette."
	}
	return h.send(ctx, msg.Chat.ID, text)
}

func (h *Handler) handleRoulette(ctx context.Context, msg *Message) error {
	chatID := msg.Chat.ID
	date := today()

	existing, err := h.storage.Queries.GetTodayResult(ctx, db.GetTodayResultParams{
		ChatID:     chatID,
		PlayedDate: date,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		return h.showExistingResult(ctx, msg, existing)
	}

	participants, err := h.storage.Queries.GetParticipants(ctx, chatID)
	if err != nil {
		return err
	}

	winner := participants[rand.IntN(len(participants))]

	if err := h.storage.Queries.SaveResult(ctx, db.SaveResultParams{
		ChatID:     chatID,
		UserID:     winner.UserID,
		PlayedDate: date,
	}); err != nil {
		existing, err2 := h.storage.Queries.GetTodayResult(ctx, db.GetTodayResultParams{
			ChatID:     chatID,
			PlayedDate: date,
		})
		if err2 != nil {
			return fmt.Errorf("save result: %w; fetch existing: %w", err, err2)
		}
		return h.showExistingResult(ctx, msg, existing)
	}

	text := fmt.Sprintf("Today's chosen one is <b>%s</b>!", formatUserName(winner.FirstName, winner.Username))
	return h.send(ctx, msg.Chat.ID, text)
}

func (h *Handler) showExistingResult(ctx context.Context, msg *Message, result db.GetTodayResultRow) error {
	p, err := h.storage.Queries.GetParticipantByID(ctx, db.GetParticipantByIDParams{
		ChatID: result.ChatID,
		UserID: result.UserID,
	})

	var name string
	if errors.Is(err, sql.ErrNoRows) {
		name = fmt.Sprintf("user %d", result.UserID)
	} else if err != nil {
		return err
	} else {
		name = formatUserName(p.FirstName, p.Username)
	}

	text := fmt.Sprintf("Today's roulette already played! The chosen one is <b>%s</b>.", name)
	return h.send(ctx, msg.Chat.ID, text)
}

func (h *Handler) handleStats(ctx context.Context, msg *Message) error {
	stats, err := h.storage.Queries.GetStats(ctx, msg.Chat.ID)
	if err != nil {
		return err
	}

	if len(stats) == 0 {
		return h.reply(ctx, msg, "No participants yet. Use /join to register!")
	}

	var sb strings.Builder
	sb.WriteString("<b>Roulette Stats</b>\n\n")
	for i, s := range stats {
		sb.WriteString(fmt.Sprintf("%d. %s — %d win(s)\n", i+1, formatUserName(s.FirstName, s.Username), s.Wins))
	}

	return h.send(ctx, msg.Chat.ID, sb.String())
}

func (h *Handler) handleReset(ctx context.Context, msg *Message) error {
	chatID := msg.Chat.ID
	date := today()

	result, err := h.storage.Queries.DeleteTodayResult(ctx, db.DeleteTodayResultParams{
		ChatID:     chatID,
		PlayedDate: date,
	})
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return h.send(ctx, chatID, "No result to reset for today.")
	}

	return h.send(ctx, chatID, "Today's result has been reset. You can run /roulette again!")
}

func (h *Handler) handleParticipants(ctx context.Context, msg *Message) error {
	participants, err := h.storage.Queries.GetParticipants(ctx, msg.Chat.ID)
	if err != nil {
		return err
	}

	if len(participants) == 0 {
		return h.send(ctx, msg.Chat.ID, "No participants yet. Use /join to register!")
	}

	var sb strings.Builder
	sb.WriteString("<b>Participants</b>\n\n")
	for i, p := range participants {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, formatUserName(p.FirstName, p.Username))
	}

	return h.send(ctx, msg.Chat.ID, sb.String())
}
