package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"telegram-chat-bot/db"
)

type Handler struct {
	bot      *BotClient
	storage  *Storage
	tr       *Translator
	botName  string
	rollCmd  string
	adminIDs map[int64]struct{}
}

func NewHandler(bot *BotClient, storage *Storage, tr *Translator, botName, rollCmd string, adminIDs []int64) *Handler {
	admins := make(map[int64]struct{}, len(adminIDs))
	for _, id := range adminIDs {
		admins[id] = struct{}{}
	}
	return &Handler{
		bot:      bot,
		storage:  storage,
		tr:       tr,
		botName:  botName,
		rollCmd:  "/" + rollCmd,
		adminIDs: admins,
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
	switch {
	case cmd == "/join":
		err = h.handleJoin(ctx, msg)
	case cmd == "/leave":
		err = h.handleLeave(ctx, msg)
	case strings.HasPrefix(cmd, h.rollCmd):
		args := extractArgs(msg)
		if sub, ok := strings.CutPrefix(args, "stats"); ok && (sub == "" || sub[0] == ' ') {
			err = h.handleStats(ctx, msg, strings.TrimSpace(sub))
		} else {
			err = h.handleRoulette(ctx, msg)
		}
	case cmd == "/stats":
		err = h.handleStats(ctx, msg, extractArgs(msg))
	case cmd == "/participants":
		err = h.handleParticipants(ctx, msg)
	case cmd == "/reset":
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

func (h *Handler) send(ctx context.Context, chatID int64, text string) error {
	return h.bot.SendMessage(ctx, SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	})
}

func today() string {
	return time.Now().UTC().Format("2006-01-02")
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

	return h.send(ctx, msg.Chat.ID, h.tr.Get(TrJoinSuccess))
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
		text = h.tr.Getf(TrLeaveSuccess, user.FirstName)
	} else {
		text = h.tr.Get(TrLeaveNotInGame)
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

	if len(participants) == 0 {
		return h.send(ctx, chatID, h.tr.Get(TrNoParticipants))
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

	winnerTag := fmt.Sprintf(`<a href="tg://user?id=%d"><b>%s</b></a>`, winner.UserID, winner.FirstName)

	setID, err := h.storage.Queries.GetRandomMessageSetID(ctx)
	if err != nil {
		text := h.tr.Getf(TrFallbackWinner, winnerTag)
		return h.send(ctx, chatID, text)
	}

	messages, err := h.storage.Queries.GetSetMessages(ctx, setID)
	if err != nil {
		log.Printf("Error fetching message set %d: %v", setID, err)
		text := h.tr.Getf(TrFallbackWinner, winnerTag)
		return h.send(ctx, chatID, text)
	}

	for i, body := range messages {
		var text string
		if i == len(messages)-1 {
			text = fmt.Sprintf(body, winnerTag)
		} else {
			text = body
		}

		if err := h.send(ctx, chatID, text); err != nil {
			log.Printf("Error sending sequence message: %v", err)
		}
		if i < len(messages)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func (h *Handler) showExistingResult(ctx context.Context, msg *Message, result db.GetTodayResultRow) error {
	p, err := h.storage.Queries.GetParticipantByID(ctx, db.GetParticipantByIDParams{
		ChatID: result.ChatID,
		UserID: result.UserID,
	})

	var name string
	if errors.Is(err, sql.ErrNoRows) {
		name = h.tr.Getf(TrUnknownUser, result.UserID)
	} else if err != nil {
		return err
	} else {
		name = p.FirstName
	}

	text := h.tr.Getf(TrAlreadyPlayed, "<b>"+name+"</b>")
	return h.send(ctx, msg.Chat.ID, text)
}

func extractArgs(msg *Message) string {
	if len(msg.Entities) == 0 {
		return ""
	}
	runes := []rune(msg.Text)
	rest := runes[msg.Entities[0].Length:]
	return strings.TrimSpace(string(rest))
}

func (h *Handler) todayWinnerID(ctx context.Context, chatID int64) int64 {
	result, err := h.storage.Queries.GetTodayResult(ctx, db.GetTodayResultParams{
		ChatID:     chatID,
		PlayedDate: today(),
	})
	if err != nil {
		return 0
	}
	return result.UserID
}

func (h *Handler) handleStats(ctx context.Context, msg *Message, arg string) error {
	if arg != "" {
		return h.handleStatsByYear(ctx, msg, arg)
	}

	stats, err := h.storage.Queries.GetStats(ctx, msg.Chat.ID)
	if err != nil {
		return err
	}

	if len(stats) == 0 {
		return h.send(ctx, msg.Chat.ID, h.tr.Get(TrNoParticipants))
	}

	winnerID := h.todayWinnerID(ctx, msg.Chat.ID)

	var sb strings.Builder
	sb.WriteString(h.tr.Get(TrStatsHeader))
	sb.WriteString("\n\n")
	for i, s := range stats {
		name := s.FirstName
		if s.UserID == winnerID {
			name = "👑 " + s.FirstName
		}
		sb.WriteString(h.tr.Getf(TrStatsLine, i+1, name, s.Wins))
		sb.WriteString("\n")
	}

	return h.send(ctx, msg.Chat.ID, sb.String())
}

func (h *Handler) handleStatsByYear(ctx context.Context, msg *Message, arg string) error {
	year, err := strconv.Atoi(arg)
	if err != nil || year < 2000 || year > 2100 {
		return h.send(ctx, msg.Chat.ID, h.tr.Getf(TrStatsInvalidYear, arg))
	}

	from := fmt.Sprintf("%d-01-01", year)
	to := fmt.Sprintf("%d-01-01", year+1)

	stats, err := h.storage.Queries.GetStatsByYear(ctx, db.GetStatsByYearParams{
		ChatID:       msg.Chat.ID,
		PlayedDate:   from,
		PlayedDate_2: to,
	})
	if err != nil {
		return err
	}

	if len(stats) == 0 {
		return h.send(ctx, msg.Chat.ID, h.tr.Getf(TrStatsNoResults, year))
	}

	winnerID := h.todayWinnerID(ctx, msg.Chat.ID)

	var sb strings.Builder
	sb.WriteString(h.tr.Getf(TrStatsYearHeader, year))
	sb.WriteString("\n\n")
	for i, s := range stats {
		name := s.FirstName
		if s.UserID == winnerID {
			name = "👑 " + s.FirstName
		}
		sb.WriteString(h.tr.Getf(TrStatsLine, i+1, name, s.Wins))
		sb.WriteString("\n")
	}

	return h.send(ctx, msg.Chat.ID, sb.String())
}

func (h *Handler) isAdmin(userID int64) bool {
	_, ok := h.adminIDs[userID]
	return ok
}

func (h *Handler) handleReset(ctx context.Context, msg *Message) error {
	if len(h.adminIDs) > 0 && !h.isAdmin(msg.From.ID) {
		return nil
	}

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
		return h.send(ctx, chatID, h.tr.Get(TrResetNoResult))
	}

	return h.send(ctx, chatID, h.tr.Get(TrResetSuccess))
}

func (h *Handler) handleParticipants(ctx context.Context, msg *Message) error {
	participants, err := h.storage.Queries.GetParticipants(ctx, msg.Chat.ID)
	if err != nil {
		return err
	}

	if len(participants) == 0 {
		return h.send(ctx, msg.Chat.ID, h.tr.Get(TrNoParticipants))
	}

	var sb strings.Builder
	sb.WriteString(h.tr.Get(TrParticipantsHeader))
	sb.WriteString("\n\n")
	for i, p := range participants {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, p.FirstName)
	}

	return h.send(ctx, msg.Chat.ID, sb.String())
}
