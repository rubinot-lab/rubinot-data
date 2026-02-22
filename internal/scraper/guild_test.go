package scraper

import (
	"context"
	"testing"

	"github.com/giovannirco/rubinot-data/internal/validation"
)

func TestParseGuildHTMLActiveFixture(t *testing.T) {
	html := readFixture(t, "guild", "active.html")
	guild, err := parseGuildHTML("Old Squad", html)
	if err != nil {
		t.Fatalf("expected active guild fixture to parse, got error: %v", err)
	}

	if guild.Name == "" || guild.World == "" {
		t.Fatalf("expected guild name and world, got %+v", guild)
	}
	if !guild.Active {
		t.Fatal("expected active guild to have active=true")
	}
	if !guild.OpenApplications {
		t.Fatal("expected active guild fixture to have open_applications=true")
	}
	if guild.Founded == "" {
		t.Fatal("expected founded date to be parsed and normalized")
	}
	if len(guild.Members) == 0 {
		t.Fatal("expected member rows to be parsed")
	}
	if len(guild.Invites) == 0 {
		t.Fatal("expected invited character rows to be parsed")
	}
	if guild.MembersTotal != len(guild.Members) {
		t.Fatalf("expected members_total=%d, got %d", len(guild.Members), guild.MembersTotal)
	}
}

func TestParseGuildHTMLDisbandedSynthetic(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "guild", "disbanded.html")
	guild, err := parseGuildHTML("Test Guild", html)
	if err != nil {
		t.Fatalf("expected disbanded fixture to parse, got error: %v", err)
	}
	if guild.Active {
		t.Fatal("expected disbanded fixture to set active=false")
	}
	if guild.DisbandCondition == "" {
		t.Fatal("expected disband_condition to be parsed")
	}
}

func TestParseGuildHTMLWithGuildhallSynthetic(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "guild", "with_guildhall.html")
	guild, err := parseGuildHTML("Guild House Team", html)
	if err != nil {
		t.Fatalf("expected with_guildhall fixture to parse, got error: %v", err)
	}
	if guild.Guildhall == nil {
		t.Fatal("expected guildhall to be parsed")
	}
	if guild.Guildhall.Name == "" || guild.Guildhall.HouseID <= 0 {
		t.Fatalf("expected guildhall name and house_id, got %+v", guild.Guildhall)
	}
}

func TestParseGuildHTMLInWarSynthetic(t *testing.T) {
	// FIXTURE: synthetic, must be replaced with real capture
	html := readFixture(t, "guild", "in_war.html")
	guild, err := parseGuildHTML("War Guild", html)
	if err != nil {
		t.Fatalf("expected in_war fixture to parse, got error: %v", err)
	}
	if !guild.InWar {
		t.Fatal("expected in_war fixture to set in_war=true")
	}
}

func TestParseGuildHTMLNotFoundFixture(t *testing.T) {
	html := readFixture(t, "guild", "not_found.html")
	_, err := parseGuildHTML("DefinitelyNotARealGuildXYZ", html)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}

func TestFetchGuildNotFound(t *testing.T) {
	notFoundFixture := readFixture(t, "guild", "not_found.html")
	server := newFakeFlareSolverrServer(t, func(_ string) string {
		return notFoundFixture
	})
	defer server.Close()

	_, _, err := FetchGuild(
		context.Background(),
		"https://www.rubinot.com.br",
		"DefinitelyNotARealGuildXYZ",
		FetchOptions{FlareSolverrURL: server.URL, MaxTimeoutMs: 120000},
	)
	assertValidationCode(t, err, validation.ErrorEntityNotFound)
}
