package utils

import (
	"fmt"
	"testing"
)

func TestGenUUID(t *testing.T) {
	uuid := GetUUID()
	if len(uuid) != 36 {
		t.Errorf("uuid length is not 36")
	}
	fmt.Printf("uuid: %s", uuid)
}

func TestGetUUIDWithoutDashes(t *testing.T) {
	uuid := GetUUIDWithoutDashes()
	if len(uuid) != 32 {
		t.Error("uuid length is not 32")
	}
	fmt.Printf("uuid: %s", uuid)
}
