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

func ParseLevelFilter(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	level, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewError(ErrorLevelFilterInvalid, fmt.Sprintf("invalid level value: %s", raw), err)
	}
	if level < 1 {
		return 0, NewError(ErrorLevelFilterInvalid, "level must be greater than or equal to 1", nil)
	}
	return level, nil
}

func ParseMonth(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	month, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewError(ErrorMonthInvalid, fmt.Sprintf("invalid month value: %s", raw), err)
	}
	if month < 1 || month > 12 {
		return 0, NewError(ErrorMonthInvalid, "month must be between 1 and 12", nil)
	}
	return month, nil
}

func ParseYear(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	year, err := strconv.Atoi(value)
	if err != nil {
		return 0, NewError(ErrorYearInvalid, fmt.Sprintf("invalid year value: %s", raw), err)
	}
	if year <= 2000 {
		return 0, NewError(ErrorYearInvalid, "year must be greater than 2000", nil)
	}
	return year, nil
}

func ParsePvPOnlyFilter(raw string) (value bool, provided bool, err error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "":
		return false, false, nil
	case "1", "true", "yes":
		return true, true, nil
	case "0", "false", "no":
		return false, true, nil
	default:
		return false, true, NewError(ErrorPvPFilterInvalid, fmt.Sprintf("invalid pvp value: %s", raw), nil)
	}
}
