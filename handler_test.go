package main

import (
	"context"
	"strings"
	"testing"

	"telegram-chat-bot/db"
)

type fakeSender struct {
	messages []SendMessageRequest
}

func (f *fakeSender) SendMessage(_ context.Context, req SendMessageRequest) error {
	f.messages = append(f.messages, req)
	return nil
}

func (f *fakeSender) last() SendMessageRequest {
	return f.messages[len(f.messages)-1]
}

func (f *fakeSender) reset() { f.messages = nil }

type testEnv struct {
	handler *Handler
	sender  *fakeSender
	storage *Storage
}

const testDate = "2026-01-15"

func setup(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	storage, err := NewStorage(ctx, ":memory:")
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	t.Cleanup(func() { storage.Close() })

	translations := map[string]string{
		"join_success":        "Welcome to the roulette! You're in the game now.",
		"leave_success":       "%s has left the roulette.",
		"leave_not_in_game":   "You're not in the game yet.",
		"no_participants":     "No players registered yet. Use /join to enter the roulette!",
		"already_played":      "The wheel has already been spun today! Today's winner is %s!",
		"fallback_winner":     "And the winner is... %s!",
		"stats_header":        "<b>Hall of Fame:</b>",
		"stats_year_header":   "<b>Hall of Fame (%d):</b>",
		"stats_invalid_year":  "Invalid year: %s",
		"stats_no_results":    "No results for %d.",
		"stats_line":          "%d. %s — %d win(s)",
		"participants_header": "<b>Players in the roulette:</b>",
		"reset_no_result":     "Nothing to reset. The wheel hasn't been spun yet.",
		"reset_success":       "The wheel has been reset. Spin again with /roll!",
		"unknown_user":        "Player #%d",
	}
	for k, v := range translations {
		if _, err := storage.db.ExecContext(ctx,
			"INSERT INTO translations (key, value) VALUES (?, ?)", k, v); err != nil {
			t.Fatalf("insert translation %q: %v", k, err)
		}
	}

	tr, err := NewTranslator(ctx, storage.Queries)
	if err != nil {
		t.Fatalf("NewTranslator: %v", err)
	}

	sender := &fakeSender{}
	handler := NewHandler(sender, storage, tr, "testbot", "roll", nil)
	handler.todayFunc = func() string { return testDate }

	return &testEnv{handler: handler, sender: sender, storage: storage}
}

func commandMsg(chatID, userID int64, firstName, text string) Update {
	cmdLen := len(text)
	if i := strings.IndexByte(text, ' '); i >= 0 {
		cmdLen = i
	}
	return Update{
		Message: &Message{
			From: &User{ID: userID, FirstName: firstName},
			Chat: Chat{ID: chatID},
			Text: text,
			Entities: []MessageEntity{
				{Type: "bot_command", Offset: 0, Length: cmdLen},
			},
		},
	}
}

func commandMsgUsername(chatID, userID int64, firstName, username, text string) Update {
	u := commandMsg(chatID, userID, firstName, text)
	u.Message.From.Username = username
	return u
}

func TestJoin(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; got != "Welcome to the roulette! You're in the game now." {
		t.Errorf("unexpected reply: %s", got)
	}

	ps, err := env.storage.Queries.GetParticipants(ctx, 100)
	if err != nil {
		t.Fatalf("GetParticipants: %v", err)
	}
	if len(ps) != 1 || ps[0].FirstName != "Alice" {
		t.Errorf("unexpected participants: %+v", ps)
	}
}

func TestJoinIdempotent(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice2", "/join"))

	ps, err := env.storage.Queries.GetParticipants(ctx, 100)
	if err != nil {
		t.Fatalf("GetParticipants: %v", err)
	}
	if len(ps) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(ps))
	}
	if ps[0].FirstName != "Alice2" {
		t.Errorf("expected upserted name Alice2, got %s", ps[0].FirstName)
	}
}

func TestLeave(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/leave"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "Alice") || !strings.Contains(got, "left") {
		t.Errorf("unexpected reply: %s", got)
	}

	ps, err := env.storage.Queries.GetParticipants(ctx, 100)
	if err != nil {
		t.Fatalf("GetParticipants: %v", err)
	}
	if len(ps) != 0 {
		t.Errorf("expected 0 participants after leave, got %d", len(ps))
	}
}

func TestLeaveNotInGame(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/leave"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; got != "You're not in the game yet." {
		t.Errorf("unexpected reply: %s", got)
	}
}

func TestRoulette(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "winner") || !strings.Contains(got, "Alice") {
		t.Errorf("expected fallback winner message with Alice, got: %s", got)
	}

	row, err := env.storage.Queries.GetTodayResult(ctx, db.GetTodayResultParams{
		ChatID: 100, PlayedDate: testDate,
	})
	if err != nil {
		t.Fatalf("GetTodayResult: %v", err)
	}
	if row.UserID != 1 {
		t.Errorf("expected winner user_id=1, got %d", row.UserID)
	}
}

func TestRouletteAlreadyPlayed(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 2, "Bob", "/roll"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "already been spun") {
		t.Errorf("expected already-played message, got: %s", got)
	}
}

func TestRouletteNoParticipants(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "No players") {
		t.Errorf("expected no-participants message, got: %s", got)
	}
}

func TestRouletteWithMessageSet(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	_, err := env.storage.db.ExecContext(ctx, "INSERT INTO message_sets (id) VALUES (1)")
	if err != nil {
		t.Fatalf("insert message_sets: %v", err)
	}
	_, err = env.storage.db.ExecContext(ctx,
		"INSERT INTO set_messages (set_id, position, body) VALUES (1, 1, 'Spinning...')")
	if err != nil {
		t.Fatalf("insert set_messages: %v", err)
	}
	_, err = env.storage.db.ExecContext(ctx,
		"INSERT INTO set_messages (set_id, position, body) VALUES (1, 2, 'Winner is %s!')")
	if err != nil {
		t.Fatalf("insert set_messages: %v", err)
	}

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))

	if len(env.sender.messages) != 2 {
		t.Fatalf("expected 2 messages (announcement sequence), got %d", len(env.sender.messages))
	}
	if got := env.sender.messages[0].Text; got != "Spinning..." {
		t.Errorf("first message: %s", got)
	}
	if got := env.sender.messages[1].Text; !strings.Contains(got, "Alice") {
		t.Errorf("last message should contain winner name, got: %s", got)
	}
}

func TestParticipants(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 2, "Bob", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/participants"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	got := env.sender.last().Text
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("expected both participants listed, got: %s", got)
	}
	if !strings.Contains(got, "1. Alice") || !strings.Contains(got, "2. Bob") {
		t.Errorf("expected numbered list, got: %s", got)
	}
}

func TestParticipantsEmpty(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/participants"))

	if got := env.sender.last().Text; !strings.Contains(got, "No players") {
		t.Errorf("expected no-participants message, got: %s", got)
	}
}

func TestStats(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 2, "Bob", "/join"))

	err := env.storage.Queries.SaveResult(ctx, db.SaveResultParams{
		ChatID: 100, UserID: 1, PlayedDate: "2025-06-01",
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	env.sender.reset()
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/stats"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	got := env.sender.last().Text
	if !strings.Contains(got, "Hall of Fame") {
		t.Errorf("expected stats header, got: %s", got)
	}
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "1 win(s)") {
		t.Errorf("expected Alice with 1 win, got: %s", got)
	}
	if !strings.Contains(got, "Bob") || !strings.Contains(got, "0 win(s)") {
		t.Errorf("expected Bob with 0 wins, got: %s", got)
	}
}

func TestStatsWithTodayWinnerCrown(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	err := env.storage.Queries.SaveResult(ctx, db.SaveResultParams{
		ChatID: 100, UserID: 1, PlayedDate: testDate,
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	env.sender.reset()
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/stats"))

	got := env.sender.last().Text
	if !strings.Contains(got, "👑") {
		t.Errorf("expected crown emoji for today's winner, got: %s", got)
	}
}

func TestStatsByYear(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	err := env.storage.Queries.SaveResult(ctx, db.SaveResultParams{
		ChatID: 100, UserID: 1, PlayedDate: "2025-06-01",
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	env.sender.reset()
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/stats 2025"))

	got := env.sender.last().Text
	if !strings.Contains(got, "2025") {
		t.Errorf("expected year in header, got: %s", got)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("expected Alice in stats, got: %s", got)
	}
}

func TestStatsByYearNoResults(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/stats 2020"))

	got := env.sender.last().Text
	if !strings.Contains(got, "No results for 2020") {
		t.Errorf("expected no-results message, got: %s", got)
	}
}

func TestStatsInvalidYear(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/stats abc"))

	got := env.sender.last().Text
	if !strings.Contains(got, "Invalid year") {
		t.Errorf("expected invalid-year message, got: %s", got)
	}
}

func TestStatsViaRollCommand(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll stats"))

	got := env.sender.last().Text
	if !strings.Contains(got, "Hall of Fame") {
		t.Errorf("expected stats via /roll stats, got: %s", got)
	}
}

func TestStatsViaRollCommandWithYear(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	err := env.storage.Queries.SaveResult(ctx, db.SaveResultParams{
		ChatID: 100, UserID: 1, PlayedDate: "2025-03-01",
	})
	if err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	env.sender.reset()
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll stats 2025"))

	got := env.sender.last().Text
	if !strings.Contains(got, "2025") || !strings.Contains(got, "Alice") {
		t.Errorf("expected yearly stats via /roll stats 2025, got: %s", got)
	}
}

func TestResetAsAdmin(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler = NewHandler(env.sender, env.storage, env.handler.tr, "testbot", "roll", []int64{1})
	env.handler.todayFunc = func() string { return testDate }

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/reset"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "has been reset") {
		t.Errorf("expected reset-success message, got: %s", got)
	}

	_, err := env.storage.Queries.GetTodayResult(ctx, db.GetTodayResultParams{
		ChatID: 100, PlayedDate: testDate,
	})
	if err == nil {
		t.Error("expected result to be deleted after reset")
	}
}

func TestResetAsNonAdmin(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler = NewHandler(env.sender, env.storage, env.handler.tr, "testbot", "roll", []int64{99})
	env.handler.todayFunc = func() string { return testDate }

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/reset"))

	if len(env.sender.messages) != 0 {
		t.Errorf("expected no reply for non-admin reset, got %d messages", len(env.sender.messages))
	}
}

func TestResetNoAdminRestriction(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/reset"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	if got := env.sender.last().Text; !strings.Contains(got, "has been reset") {
		t.Errorf("expected reset-success message, got: %s", got)
	}
}

func TestResetNothingToReset(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/reset"))

	if got := env.sender.last().Text; !strings.Contains(got, "Nothing to reset") {
		t.Errorf("expected nothing-to-reset message, got: %s", got)
	}
}

func TestCommandForDifferentBot(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	update := Update{
		Message: &Message{
			From: &User{ID: 1, FirstName: "Alice"},
			Chat: Chat{ID: 100},
			Text: "/join@otherbot",
			Entities: []MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 14},
			},
		},
	}
	env.handler.HandleUpdate(ctx, update)

	if len(env.sender.messages) != 0 {
		t.Errorf("expected no reply for command to different bot, got %d", len(env.sender.messages))
	}
}

func TestCommandForThisBot(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	update := Update{
		Message: &Message{
			From: &User{ID: 1, FirstName: "Alice"},
			Chat: Chat{ID: 100},
			Text: "/join@testbot",
			Entities: []MessageEntity{
				{Type: "bot_command", Offset: 0, Length: 13},
			},
		},
	}
	env.handler.HandleUpdate(ctx, update)

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
}

func TestNonCommandMessage(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	update := Update{
		Message: &Message{
			From: &User{ID: 1, FirstName: "Alice"},
			Chat: Chat{ID: 100},
			Text: "hello",
		},
	}
	env.handler.HandleUpdate(ctx, update)

	if len(env.sender.messages) != 0 {
		t.Errorf("expected no reply for plain text, got %d", len(env.sender.messages))
	}
}

func TestNilMessage(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, Update{})

	if len(env.sender.messages) != 0 {
		t.Errorf("expected no reply for nil message, got %d", len(env.sender.messages))
	}
}

func TestCustomRollCommand(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler = NewHandler(env.sender, env.storage, env.handler.tr, "testbot", "spin", nil)
	env.handler.todayFunc = func() string { return testDate }

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/spin"))

	if len(env.sender.messages) < 1 {
		t.Fatal("expected at least 1 message for custom roll command")
	}
}

func TestSendUsesHTMLParseMode(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))

	if got := env.sender.last().ParseMode; got != "HTML" {
		t.Errorf("expected HTML parse mode, got %q", got)
	}
}

func TestChatIsolation(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(200, 1, "Alice", "/participants"))

	if got := env.sender.last().Text; !strings.Contains(got, "No players") {
		t.Errorf("expected no participants in different chat, got: %s", got)
	}
}

func TestRouletteShowsExistingWinnerEvenAfterLeave(t *testing.T) {
	env := setup(t)
	ctx := context.Background()

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/join"))
	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/roll"))

	env.handler.HandleUpdate(ctx, commandMsg(100, 1, "Alice", "/leave"))
	env.sender.reset()

	env.handler.HandleUpdate(ctx, commandMsg(100, 2, "Bob", "/roll"))

	if len(env.sender.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(env.sender.messages))
	}
	got := env.sender.last().Text
	if !strings.Contains(got, "already been spun") {
		t.Errorf("expected already-played message, got: %s", got)
	}
	if !strings.Contains(got, "Player #1") {
		t.Errorf("expected unknown user fallback, got: %s", got)
	}
}

func TestExtractCommand(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		botName string
		want    string
	}{
		{
			name:    "no entities",
			msg:     &Message{Text: "hello"},
			botName: "bot",
			want:    "",
		},
		{
			name: "not a bot_command",
			msg: &Message{
				Text:     "hello",
				Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 5}},
			},
			botName: "bot",
			want:    "",
		},
		{
			name: "non-zero offset",
			msg: &Message{
				Text:     "hey /join",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 4, Length: 5}},
			},
			botName: "bot",
			want:    "",
		},
		{
			name: "simple command",
			msg: &Message{
				Text:     "/join",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			},
			botName: "bot",
			want:    "/join",
		},
		{
			name: "command with bot mention",
			msg: &Message{
				Text:     "/join@testbot",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 13}},
			},
			botName: "testbot",
			want:    "/join",
		},
		{
			name: "command with wrong bot",
			msg: &Message{
				Text:     "/join@otherbot",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 14}},
			},
			botName: "testbot",
			want:    "",
		},
		{
			name: "case insensitive",
			msg: &Message{
				Text:     "/JOIN",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			},
			botName: "bot",
			want:    "/join",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommand(tt.msg, tt.botName)
			if got != tt.want {
				t.Errorf("extractCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractArgs(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want string
	}{
		{
			name: "no args",
			msg: &Message{
				Text:     "/roll",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			},
			want: "",
		},
		{
			name: "with args",
			msg: &Message{
				Text:     "/roll stats 2025",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			},
			want: "stats 2025",
		},
		{
			name: "extra whitespace",
			msg: &Message{
				Text:     "/roll   stats  ",
				Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 5}},
			},
			want: "stats",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractArgs(tt.msg)
			if got != tt.want {
				t.Errorf("extractArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}
