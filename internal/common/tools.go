package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// generateUUID generates a UUID with an optional prefix
func GenerateUUID(prefix string) string {
	id := uuid.New()
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, strings.ReplaceAll(id.String(), "-", ""))
	}
	return id.String()
}

// GenerateShortUUID generates a shorter UUID without dashes
func GenerateShortUUID(prefix string) string {
	id := uuid.New()
	shortID := strings.ReplaceAll(id.String(), "-", "")
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, shortID)
	}
	return shortID
}

// GenerateOrderID generates an order ID with "ord" prefix
func GenerateOrderID() string {
	return GenerateUUID("ord")
}

// GeneratePositionID generates a position ID with "pos" prefix
func GeneratePositionID() string {
	return GenerateUUID("pos")
}
