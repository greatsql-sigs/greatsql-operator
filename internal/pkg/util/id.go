package util

import (
	"strings"

	"github.com/google/uuid"
)

// GetUUID generates a new UUID
func GetUUID() string {
	return uuid.New().String()
}

// GetUUIDWithoutDashes generates a new UUID not horizontal line
func GetUUIDWithoutDashes() string {
	u := uuid.New().String()

	return strings.ReplaceAll(u, "-", "")
}
