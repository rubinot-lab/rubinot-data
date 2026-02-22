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

func ParseNewsID(raw string) (int, error) {
	newsID, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, NewError(ErrorNewsIDInvalid, fmt.Sprintf("invalid news_id value: %s", raw), err)
	}
	if newsID <= 0 {
		return 0, NewError(ErrorNewsIDInvalid, "news_id must be greater than 0", nil)
	}
	return newsID, nil
}

func ParseArchiveDays(raw string, fallback int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, nil
	}

	days, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewError(ErrorArchiveDaysInvalid, fmt.Sprintf("invalid archive days value: %s", raw), err)
	}
	if days <= 0 {
		return 0, NewError(ErrorArchiveDaysInvalid, "archive days must be greater than 0", nil)
	}
	return days, nil
}
