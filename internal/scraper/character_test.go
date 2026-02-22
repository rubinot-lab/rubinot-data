package scraper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseCharacterHTMLNormalFixture(t *testing.T) {
	html := readFixture(t, "character", "normal.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected normal character fixture to parse, got error: %v", err)
	}

	if result.CharacterInfo.Name != "Alice Sparrow" {
		t.Fatalf("expected name Alice Sparrow, got %q", result.CharacterInfo.Name)
	}
	if result.CharacterInfo.Traded {
		t.Fatal("expected normal fixture to not be traded")
	}
	if result.CharacterInfo.LastLogin != "2026-02-22T21:34:38Z" {
		t.Fatalf("expected normalized last_login in UTC, got %q", result.CharacterInfo.LastLogin)
	}
	if result.CharacterInfo.AccountStatus == "" {
		t.Fatal("expected account status to be parsed")
	}
	if len(result.Deaths) == 0 {
		t.Fatal("expected at least one death entry")
	}
	if result.Deaths[0].Time == "" || !strings.HasSuffix(result.Deaths[0].Time, "Z") {
		t.Fatalf("expected RFC3339 UTC death time, got %q", result.Deaths[0].Time)
	}
	if len(result.OtherCharacters) == 0 {
		t.Fatal("expected account characters to be parsed")
	}
}

func TestParseCharacterHTMLWithDeathsFixture(t *testing.T) {
	html := readFixture(t, "character", "with_deaths.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected with_deaths fixture to parse, got error: %v", err)
	}

	if len(result.Deaths) == 0 {
		t.Fatal("expected deaths in with_deaths fixture")
	}
	if result.Deaths[0].Level <= 0 {
		t.Fatalf("expected parsed death level, got %+v", result.Deaths[0])
	}
}

func TestParseCharacterHTMLTradedFixture(t *testing.T) {
	html := readFixture(t, "character", "traded.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected traded fixture to parse, got error: %v", err)
	}
	if !result.CharacterInfo.Traded {
		t.Fatal("expected traded fixture to set traded=true")
	}
	if !strings.Contains(result.CharacterInfo.AuctionURL, "currentcharactertrades") {
		t.Fatalf("expected auction URL in traded fixture, got %q", result.CharacterInfo.AuctionURL)
	}
}

func TestParseCharacterHTMLWithHouseFixture(t *testing.T) {
	html := readFixture(t, "character", "with_house.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected with_house fixture to parse, got error: %v", err)
	}
	if len(result.CharacterInfo.Houses) == 0 {
		t.Fatal("expected parsed house list")
	}
	house := result.CharacterInfo.Houses[0]
	if house.Name == "" || house.HouseID <= 0 {
		t.Fatalf("expected house details with name and ID, got %+v", house)
	}
}

func TestParseCharacterHTMLGuildMemberFixture(t *testing.T) {
	html := readFixture(t, "character", "guild_member.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected guild_member fixture to parse, got error: %v", err)
	}
	if result.CharacterInfo.Guild == nil {
		t.Fatal("expected guild info to be parsed")
	}
	if result.CharacterInfo.Guild.Name == "" || result.CharacterInfo.Guild.Rank == "" {
		t.Fatalf("expected guild name and rank, got %+v", result.CharacterInfo.Guild)
	}
}

func TestParseCharacterHTMLNotFoundFixture(t *testing.T) {
	html := readFixture(t, "character", "not_found.html")
	_, err := parseCharacterHTML(html)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func TestParseCharacterHTMLDeletedFixture(t *testing.T) {
	html := readFixture(t, "character", "deleted.html")
	_, err := parseCharacterHTML(html)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func TestParseCharacterHTMLBannedSynthetic(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "character", "banned.html")
	result, err := parseCharacterHTML(html)
	if err != nil {
		t.Fatalf("expected synthetic banned fixture to parse, got error: %v", err)
	}
	if !result.CharacterInfo.IsBanned {
		t.Fatal("expected banned fixture to set is_banned=true")
	}
	if result.CharacterInfo.BanReason == "" {
		t.Fatal("expected banned fixture to include ban_reason")
	}
}

func TestFetchCharacterNotFound(t *testing.T) {
	notFoundFixture := readFixture(t, "character", "not_found.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return notFoundFixture
	})
	defer server.Close()

	_, _, err := FetchCharacter(
		context.Background(),
		"https://www.rubinot.com.br",
		"FakePlayerDefinitelyNotExistsXYZ",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func assertValidationCode(t *testing.T, err error, expectedCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected validation error code %d, got nil", expectedCode)
	}

	var validationErr validation.Error
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation.Error, got %T: %v", err, err)
	}
	if validationErr.Code() != expectedCode {
		t.Fatalf("expected error code %d, got %d", expectedCode, validationErr.Code())
	}
}
