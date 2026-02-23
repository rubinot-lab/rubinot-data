package validation

import (
	"errors"
	"strings"
	"testing"
)

func TestParseLatestDeathsWorldOptions(t *testing.T) {
	html := `
		<html><body>
			<select name="world">
				<option value="">All Worlds</option>
				<option value="15">Belaria</option>
				<option value="22">Serenian</option>
			</select>
		</body></html>
	`

	worlds, err := ParseLatestDeathsWorldOptions(html)
	if err != nil {
		t.Fatalf("expected worlds to parse successfully, got error: %v", err)
	}
	if len(worlds) != 2 {
		t.Fatalf("expected 2 worlds, got %d", len(worlds))
	}
	if worlds[0].Name != "Belaria" || worlds[0].ID != 15 {
		t.Fatalf("unexpected first world: %+v", worlds[0])
	}
}

func TestWorldExists(t *testing.T) {
	validator := testValidator()

	t.Run("valid", func(t *testing.T) {
		name, id, ok := validator.WorldExists("Belaria")
		if !ok || name != "Belaria" || id != 15 {
			t.Fatalf("expected Belaria world match, got name=%q id=%d ok=%v", name, id, ok)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		name, id, ok := validator.WorldExists("bElArIa")
		if !ok || name != "Belaria" || id != 15 {
			t.Fatalf("expected case-insensitive world match, got name=%q id=%d ok=%v", name, id, ok)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, _, ok := validator.WorldExists("Unknown")
		if ok {
			t.Fatal("expected unknown world to be invalid")
		}
	})
}

func TestTownExists(t *testing.T) {
	validator := testValidator()

	t.Run("valid", func(t *testing.T) {
		name, id, ok := validator.TownExists("Venore")
		if !ok || name != "Venore" || id != 1 {
			t.Fatalf("expected Venore town match, got name=%q id=%d ok=%v", name, id, ok)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		name, id, ok := validator.TownExists("ab dendriel")
		if !ok || name != "Ab Dendriel" || id != 5 {
			t.Fatalf("expected Ab Dendriel alias match, got name=%q id=%d ok=%v", name, id, ok)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, _, ok := validator.TownExists("Unknown Town")
		if ok {
			t.Fatal("expected unknown town to be invalid")
		}
	})
}

func TestCharacterNameValidation(t *testing.T) {
	valid, err := IsCharacterNameValid(" Test   Name ")
	if err != nil {
		t.Fatalf("expected valid character name, got error: %v", err)
	}
	if valid != "Test Name" {
		t.Fatalf("expected normalized name 'Test Name', got %q", valid)
	}

	_, err = IsCharacterNameValid("A")
	assertValidationCode(t, err, ErrorCharacterNameTooShort)

	_, err = IsCharacterNameValid(strings.Repeat("A", 30))
	assertValidationCode(t, err, ErrorCharacterNameTooLong)

	_, err = IsCharacterNameValid("Name@")
	assertValidationCode(t, err, ErrorCharacterNameInvalidBoundary)
}

func TestGuildNameValidation(t *testing.T) {
	valid, err := IsGuildNameValid("  Great   Guild  ")
	if err != nil {
		t.Fatalf("expected valid guild name, got error: %v", err)
	}
	if valid != "Great Guild" {
		t.Fatalf("expected normalized guild name 'Great Guild', got %q", valid)
	}

	_, err = IsGuildNameValid("")
	assertValidationCode(t, err, ErrorGuildNameEmpty)

	_, err = IsGuildNameValid("##")
	assertValidationCode(t, err, ErrorGuildNameTooShort)
}

func TestResolveHighscoreCategory(t *testing.T) {
	validator := testValidator()

	category, ok := validator.ResolveHighscoreCategory("exp")
	if !ok {
		t.Fatal("expected exp alias to resolve")
	}
	if category.Slug != "experience" || category.ID != 6 {
		t.Fatalf("unexpected category resolution: %+v", category)
	}

	_, ok = validator.ResolveHighscoreCategory("invalid-category")
	if ok {
		t.Fatal("expected invalid category alias to fail")
	}
}

func TestResolveVocation(t *testing.T) {
	validator := testValidator()

	vocation, ok := validator.ResolveVocation("ek")
	if !ok || vocation != "Knights" {
		t.Fatalf("expected ek alias to resolve to Knights, got vocation=%q ok=%v", vocation, ok)
	}

	_, ok = validator.ResolveVocation("not-a-vocation")
	if ok {
		t.Fatal("expected invalid vocation alias to fail")
	}
}

func TestResolveHighscoreVocation(t *testing.T) {
	validator := testValidator()

	vocation, ok := validator.ResolveHighscoreVocation("all")
	if !ok {
		t.Fatal("expected all alias to resolve")
	}
	if vocation.Name != "(all)" || vocation.ProfessionID != 0 {
		t.Fatalf("unexpected all vocation resolution: %+v", vocation)
	}

	knight, ok := validator.ResolveHighscoreVocation("ek")
	if !ok {
		t.Fatal("expected ek alias to resolve")
	}
	if knight.Name != "Knights" || knight.ProfessionID != 2 {
		t.Fatalf("unexpected ek vocation resolution: %+v", knight)
	}
}

func TestValidatePage(t *testing.T) {
	if err := ValidatePage(1); err != nil {
		t.Fatalf("expected page 1 to be valid, got error: %v", err)
	}

	if _, err := ParsePage("0"); err == nil {
		t.Fatal("expected page 0 to fail")
	} else {
		assertValidationCode(t, err, ErrorPageOutOfBounds)
	}

	if _, err := ParsePage("abc"); err == nil {
		t.Fatal("expected non-int page to fail")
	} else {
		assertValidationCode(t, err, ErrorPageOutOfBounds)
	}
}

func TestParseHouseID(t *testing.T) {
	id, err := ParseHouseID("123")
	if err != nil {
		t.Fatalf("expected house id to parse successfully, got: %v", err)
	}
	if id != 123 {
		t.Fatalf("expected house id 123, got %d", id)
	}

	_, err = ParseHouseID("0")
	assertValidationCode(t, err, ErrorHouseIDInvalid)

	_, err = ParseHouseID("abc")
	assertValidationCode(t, err, ErrorHouseIDInvalid)
}

func TestParseNewsID(t *testing.T) {
	id, err := ParseNewsID("140")
	if err != nil {
		t.Fatalf("expected news id to parse successfully, got: %v", err)
	}
	if id != 140 {
		t.Fatalf("expected news id 140, got %d", id)
	}

	_, err = ParseNewsID("0")
	assertValidationCode(t, err, ErrorNewsIDInvalid)

	_, err = ParseNewsID("abc")
	assertValidationCode(t, err, ErrorNewsIDInvalid)
}

func TestParseAuctionID(t *testing.T) {
	auctionID, err := ParseAuctionID(" 165320 ")
	if err != nil {
		t.Fatalf("expected auction id to parse successfully, got: %v", err)
	}
	if auctionID != "165320" {
		t.Fatalf("expected auction id 165320, got %q", auctionID)
	}

	_, err = ParseAuctionID("")
	assertValidationCode(t, err, ErrorAuctionIDInvalid)
}

func TestParseArchiveDays(t *testing.T) {
	days, err := ParseArchiveDays("", 90)
	if err != nil {
		t.Fatalf("expected empty archive days to fallback, got: %v", err)
	}
	if days != 90 {
		t.Fatalf("expected fallback archive days 90, got %d", days)
	}

	days, err = ParseArchiveDays("30", 90)
	if err != nil {
		t.Fatalf("expected archive days to parse, got: %v", err)
	}
	if days != 30 {
		t.Fatalf("expected archive days 30, got %d", days)
	}

	_, err = ParseArchiveDays("0", 90)
	assertValidationCode(t, err, ErrorArchiveDaysInvalid)

	_, err = ParseArchiveDays("abc", 90)
	assertValidationCode(t, err, ErrorArchiveDaysInvalid)
}

func TestParseLevelFilter(t *testing.T) {
	level, err := ParseLevelFilter("")
	if err != nil {
		t.Fatalf("expected empty level filter to be valid, got: %v", err)
	}
	if level != 0 {
		t.Fatalf("expected level 0 for empty filter, got %d", level)
	}

	level, err = ParseLevelFilter("200")
	if err != nil {
		t.Fatalf("expected level to parse successfully, got: %v", err)
	}
	if level != 200 {
		t.Fatalf("expected level 200, got %d", level)
	}

	_, err = ParseLevelFilter("0")
	assertValidationCode(t, err, ErrorLevelFilterInvalid)

	_, err = ParseLevelFilter("abc")
	assertValidationCode(t, err, ErrorLevelFilterInvalid)
}

func TestParseMonth(t *testing.T) {
	month, err := ParseMonth("")
	if err != nil {
		t.Fatalf("expected empty month to be optional, got %v", err)
	}
	if month != 0 {
		t.Fatalf("expected empty month=0, got %d", month)
	}

	month, err = ParseMonth("2")
	if err != nil {
		t.Fatalf("expected month 2 to parse successfully, got %v", err)
	}
	if month != 2 {
		t.Fatalf("expected month=2, got %d", month)
	}

	_, err = ParseMonth("13")
	assertValidationCode(t, err, ErrorMonthInvalid)

	_, err = ParseMonth("abc")
	assertValidationCode(t, err, ErrorMonthInvalid)
}

func TestParseYear(t *testing.T) {
	year, err := ParseYear("")
	if err != nil {
		t.Fatalf("expected empty year to be optional, got %v", err)
	}
	if year != 0 {
		t.Fatalf("expected empty year=0, got %d", year)
	}

	year, err = ParseYear("2026")
	if err != nil {
		t.Fatalf("expected year 2026 to parse successfully, got %v", err)
	}
	if year != 2026 {
		t.Fatalf("expected year=2026, got %d", year)
	}

	_, err = ParseYear("2000")
	assertValidationCode(t, err, ErrorYearInvalid)

	_, err = ParseYear("abc")
	assertValidationCode(t, err, ErrorYearInvalid)
}

func TestParsePvPOnlyFilter(t *testing.T) {
	value, provided, err := ParsePvPOnlyFilter("")
	if err != nil {
		t.Fatalf("expected empty pvp filter to be valid, got: %v", err)
	}
	if provided {
		t.Fatalf("expected provided=false for empty pvp filter, got %v", provided)
	}
	if value {
		t.Fatalf("expected value=false for empty pvp filter, got %v", value)
	}

	value, provided, err = ParsePvPOnlyFilter("1")
	if err != nil {
		t.Fatalf("expected pvp filter 1 to be valid, got: %v", err)
	}
	if !provided || !value {
		t.Fatalf("expected provided=true value=true for pvp=1, got provided=%v value=%v", provided, value)
	}

	value, provided, err = ParsePvPOnlyFilter("false")
	if err != nil {
		t.Fatalf("expected pvp filter false to be valid, got: %v", err)
	}
	if !provided || value {
		t.Fatalf("expected provided=true value=false for pvp=false, got provided=%v value=%v", provided, value)
	}

	_, _, err = ParsePvPOnlyFilter("maybe")
	assertValidationCode(t, err, ErrorPvPFilterInvalid)
}

func testValidator() *Validator {
	return NewValidator([]World{
		{ID: 15, Name: "Belaria"},
		{ID: 22, Name: "Serenian"},
	})
}

func assertValidationCode(t *testing.T, err error, expected int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error code %d, got nil", expected)
	}

	var validationErr Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation.Error, got %T: %v", err, err)
	}
	if validationErr.Code() != expected {
		t.Fatalf("expected code %d, got %d", expected, validationErr.Code())
	}
}
