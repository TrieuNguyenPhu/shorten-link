package generator

import (
	"context"
	"errors"
	"regexp"
	"testing"
)

func TestNewCryptoRandomRejectsInvalidLength(t *testing.T) {
	if _, err := NewCryptoRandom(0); err == nil {
		t.Fatal("NewCryptoRandom(0) returned no error")
	}
}

func TestCryptoRandomGenerate(t *testing.T) {
	generator, err := NewCryptoRandom(7)
	if err != nil {
		t.Fatalf("NewCryptoRandom() error = %v", err)
	}

	code, err := generator.Generate(context.Background())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !regexp.MustCompile(`^[a-z0-9]{7}$`).MatchString(code) {
		t.Fatalf("Generate() = %q, want seven lowercase alphanumeric characters", code)
	}
}

func TestCryptoRandomHonorsCanceledContext(t *testing.T) {
	generator, err := NewCryptoRandom(7)
	if err != nil {
		t.Fatalf("NewCryptoRandom() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := generator.Generate(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Generate() error = %v, want context.Canceled", err)
	}
}
