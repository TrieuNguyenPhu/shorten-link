package main

import (
	"context"
	"log"

	"github.com/TrieuNguyenPhu/shorten-link/backend/internal/httpapi"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	handler, err := httpapi.NewRuntime(context.Background(), "delete-link")
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(httpapi.WithDeadline(handler.Delete))
}
