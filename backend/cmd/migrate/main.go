package main

import (
	"context"
	"log"

	"droplite/internal/config"
	"droplite/internal/database"
	"droplite/internal/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(context.Background(), cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()

	if err := migrations.Apply(context.Background(), db); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	log.Println("migrations applied")
}
