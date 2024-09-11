package mysql

import (
	"fmt"
	"testing"
)

func TestGetMaxGTIDMember(t *testing.T) {

	gtidArryList := []string{
		"3f65a290-a2f8-11ee-acdd-d08e7908bcb1:1-1049331886",
		"3f65a4e4-a2f8-11ee-acdd-d08e7908bcb1:1-54",
		"46dda72d-ceec-11ee-be3f-d08e7908bcb1:1-1906906",
		"46dda990-ceec-11ee-be3f-d08e7908bcb1:1-53",
		"922da9f5-ba80-11ee-bd11-d08e7908bcb1:1-76",
		"922dac85-ba80-11ee-bd11-d08e7908bcb1:1-6",
		"9d4e207c-a2f7-11ee-8953-d08e7908bcb1:1-3294083",
		"f4a28df0-aebc-11ee-98ca-d08e7908bcb1:1-107367635",
		"f4a291f6-aebc-11ee-98ca-d08e7908bcb1:1-9",
	}

	maxGTIDMember, err := GetMaxGTIDMember(gtidArryList)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("Node with max GTID: %s:%d-%d\n", maxGTIDMember.UUID, maxGTIDMember.Start, maxGTIDMember.End)
}
