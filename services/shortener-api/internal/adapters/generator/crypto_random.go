package generator

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

type CryptoRandom struct {
	length int
}

func NewCryptoRandom(length int) (*CryptoRandom, error) {
	if length <= 0 {
		return nil, errors.New("short code length must be greater than zero")
	}
	return &CryptoRandom{length: length}, nil
}

func (g *CryptoRandom) Generate(ctx context.Context) (string, error) {
	code := make([]byte, g.length)
	limit := big.NewInt(int64(len(alphabet)))
	for i := range code {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		index, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", fmt.Errorf("read cryptographic randomness: %w", err)
		}
		code[i] = alphabet[index.Int64()]
	}
	return string(code), nil
}
