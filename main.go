package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "bot.db"
	}

	rollCmd := os.Getenv("ROLL_COMMAND")
	if rollCmd == "" {
		rollCmd = "roll"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	bot := NewBotClient(token)

	me, err := bot.GetMe(ctx)
	if err != nil {
		log.Fatalf("Failed to verify bot token: %v", err)
	}
	log.Printf("Bot started: @%s", me.Username)

	storage, err := NewStorage(ctx, dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer storage.Close()

	tr, err := NewTranslator(ctx, storage.Queries)
	if err != nil {
		log.Fatalf("Failed to load translations: %v", err)
	}

	handler := NewHandler(bot, storage, tr, me.Username, rollCmd)

	var offset int64
	for {
		updates, err := bot.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Shutting down...")
				return
			}
			continue
		}

		for _, update := range updates {
			handler.HandleUpdate(ctx, update)
			offset = update.UpdateID + 1
		}
	}
}
