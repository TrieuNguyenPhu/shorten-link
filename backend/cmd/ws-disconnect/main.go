package main

import (
	"context"
	"log"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/wsapi"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	handler, err := wsapi.NewRuntime(context.Background(), "ws-disconnect")
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler.Disconnect)
}
