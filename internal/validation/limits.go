package validation

import (
	"fmt"
	"strconv"
	"strings"
)

func ValidatePage(page int) error {
	if page < 1 {
		return NewError(ErrorPageOutOfBounds, "page must be greater than or equal to 1", nil)
	}
	return nil
}

func ParsePage(raw string) (int, error) {
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, NewError(ErrorPageOutOfBounds, fmt.Sprintf("invalid page value: %s", raw), err)
	}
	if page < 1 {
		return 0, NewError(ErrorPageOutOfBounds, "page must be greater than or equal to 1", nil)
	}
	return page, nil
}

func ParseHouseID(raw string) (int, error) {
	houseID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, NewError(ErrorHouseIDInvalid, fmt.Sprintf("invalid house_id value: %s", raw), err)
	}
	if houseID < 1 {
		return 0, NewError(ErrorHouseIDInvalid, "house_id must be greater than or equal to 1", nil)
	}
	return houseID, nil
}
