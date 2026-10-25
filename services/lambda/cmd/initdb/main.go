package main

import (
	"context"
	"log"
	"time"

	"shortlink/internal/adapters/dynamostore"
	"shortlink/internal/localconfig"
)

func main() {
	cfg, err := localconfig.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := dynamostore.EnsureTable(ctx, cfg.Client(), cfg.Table); err != nil {
		log.Fatal(err)
	}
	log.Printf("DynamoDB Local table %s is ready at %s", cfg.Table, cfg.Endpoint)
}
