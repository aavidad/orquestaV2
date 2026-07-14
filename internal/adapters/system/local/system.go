package local

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"
)

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Now().UTC()
}

type IDGenerator struct{}

func (IDGenerator) NewID(ctx context.Context, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if !validPrefix(prefix) {
		return "", errors.New("id.prefix_invalid")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return prefix + ":" + hex.EncodeToString(random[:]), nil
}

func validPrefix(prefix string) bool {
	if prefix == "" || strings.TrimSpace(prefix) != prefix {
		return false
	}
	for _, character := range prefix {
		if !unicode.IsLower(character) && !unicode.IsDigit(character) && character != '-' {
			return false
		}
	}
	return true
}
