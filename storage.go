package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"

	"telegram-chat-bot/db"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

type Storage struct {
	DB      *sql.DB
	Queries *db.Queries
}

func NewStorage(ctx context.Context, dbPath string) (*Storage, error) {
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, ddl); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return &Storage{
		DB:      sqlDB,
		Queries: db.New(sqlDB),
	}, nil
}

func (s *Storage) Close() error {
	return s.DB.Close()
}
