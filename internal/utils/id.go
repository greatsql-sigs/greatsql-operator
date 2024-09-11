package utils

import (
	"strings"

	"github.com/google/uuid"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-05-02 00:34:31
 * @file: id.go
 * @description: id util
 */

// GetUUID generates a new UUID
func GetUUID() string {
	return uuid.New().String()
}

// GetUUIDWithoutDashes generates a new UUID not horizontal line
func GetUUIDWithoutDashes() string {
	uuid := uuid.New().String()

	return strings.Replace(uuid, "-", "", -1)
}
