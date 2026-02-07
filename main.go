package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is required")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "roulette.db"
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

	handler := NewHandler(bot, storage, me.Username)

	var offset int64
	for {
		updates, err := bot.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Shutting down...")
				return
			}
			log.Printf("GetUpdates error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			handler.HandleUpdate(ctx, update)
			offset = update.UpdateID + 1
		}
	}
}
