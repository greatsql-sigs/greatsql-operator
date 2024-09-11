package mysql

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-08-04 16:36:55
 * @file: gtid.go
 * @description: gtid compare
 */

// GTID represents a Global Transaction Identifier
type GTID struct {
	Host     string
	UUID     string
	ServerID int
	Start    int64
	End      int64
}

// parseGTID parses a GTID string into a GTID object
func parseGTID(gtid string) (GTID, error) {

	if gtid == "0" || gtid == "1" {
		return GTID{End: 0}, errors.New("gtid is 0 or 1, skip parse comparison")
	}

	gtidPattern := `([0-9a-f-]+):(\d+)-(\d+)`
	rex := regexp.MustCompile(gtidPattern)
	matches := rex.FindStringSubmatch(gtid)
	if len(matches) != 4 {
		return GTID{}, fmt.Errorf("invalid GTID format: %s", gtid)
	}

	start, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return GTID{}, fmt.Errorf("invalid GTID format: %s", gtid)
	}

	end, err := strconv.ParseInt(matches[3], 10, 64)
	if err != nil {
		return GTID{}, fmt.Errorf("invalid GTID format: %s", gtid)
	}

	return GTID{
		UUID:  matches[1],
		Start: start,
		End:   end,
	}, nil
}

// compareGTIDs compares a list of GTIDs and returns the one with the highest end value
func compareGTIDs(gtid []GTID) GTID {

	if len(gtid) == 0 {
		return GTID{}
	}

	maxGTID := gtid[0]
	for _, gtid := range gtid[1:] {
		if gtid.End > maxGTID.End {
			maxGTID = gtid
		}
	}
	return maxGTID
}

// GetMaxGTIDMember returns the GTID with the highest end value from a list of GTID strings
func GetMaxGTIDMember(gtidArryList []string) (GTID, error) {
	var gtids []GTID
	for _, gtidStr := range gtidArryList {
		gtid, err := parseGTID(gtidStr)
		if err != nil {
			return GTID{}, err
		}

		if gtid.End == 0 {
			continue
		}
		gtids = append(gtids, gtid)
	}

	return compareGTIDs(gtids), nil
}
