package main

import (
	"context"
	"fmt"
	"log"

	"telegram-chat-bot/db"
)

const (
	TrJoinSuccess        = "join_success"
	TrLeaveSuccess       = "leave_success"
	TrLeaveNotInGame     = "leave_not_in_game"
	TrNoParticipants     = "no_participants"
	TrAlreadyPlayed      = "already_played"
	TrFallbackWinner     = "fallback_winner"
	TrStatsHeader        = "stats_header"
	TrStatsYearHeader    = "stats_year_header"
	TrStatsInvalidYear   = "stats_invalid_year"
	TrStatsNoResults     = "stats_no_results"
	TrStatsLine          = "stats_line"
	TrParticipantsHeader = "participants_header"
	TrResetNoResult      = "reset_no_result"
	TrResetSuccess       = "reset_success"
	TrUnknownUser        = "unknown_user"
)

type Translator struct {
	translations map[string]string
}

func NewTranslator(ctx context.Context, queries *db.Queries) (*Translator, error) {
	t := &Translator{}
	if err := t.load(ctx, queries); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Translator) load(ctx context.Context, queries *db.Queries) error {
	rows, err := queries.GetAllTranslations(ctx)
	if err != nil {
		return fmt.Errorf("load translations: %w", err)
	}

	t.translations = make(map[string]string, len(rows))
	for _, row := range rows {
		t.translations[row.Key] = row.Value
	}

	log.Printf("Loaded %d translations", len(t.translations))
	return nil
}

func (t *Translator) Get(key string) string {
	if val, ok := t.translations[key]; ok {
		return val
	}
	return key
}

func (t *Translator) Getf(key string, args ...any) string {
	return fmt.Sprintf(t.Get(key), args...)
}
