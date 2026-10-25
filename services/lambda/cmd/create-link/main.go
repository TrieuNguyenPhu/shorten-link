package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"log"
	"shortlink/internal/awsconfig"
)

func main() {
	handler, err := awsconfig.Handler()
	if err != nil {
		log.Fatal(err)
	}
	lambda.Start(handler.Create)
}
