package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	characterTitlePattern     = regexp.MustCompile(`^(.+?)\s*\((\d+)\s+titles\s+unlocked\)$`)
	characterDeathPattern     = regexp.MustCompile(`(?i)killed\s+at\s+level\s+(\d+)\s+by\s+(.+)$`)
	characterParentheticalRe  = regexp.MustCompile(`\(([^)]*)\)`)
	characterHouseIDPattern   = regexp.MustCompile(`(?i)[?&]houseid=(\d+)`)
	characterGuildRankPattern = regexp.MustCompile(`(?i)^(.+?)\s+of\s+the\s+(.+)$`)
	characterRowPrefixPattern = regexp.MustCompile(`^\d+\.\s*`)
)

func FetchCharacter(ctx context.Context, baseURL, characterName string, opts FetchOptions) (domain.CharacterResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchCharacter")
	defer span.End()

	started := time.Now()
	sourceURL := fmt.Sprintf(
		"%s/?subtopic=characters&name=%s",
		strings.TrimRight(baseURL, "/"),
		url.QueryEscape(characterName),
	)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "character"),
		attribute.String("rubinot.character", characterName),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("character").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("character", "error").Inc()
		return domain.CharacterResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("character", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseCharacterHTML(htmlBody)
	parseDuration.WithLabelValues("character").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.CharacterResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseCharacterHTML(htmlBody string) (domain.CharacterResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.CharacterResult{}, err
	}

	errorText := normalizeText(doc.Find(".ErrorMessage").First().Text())
	if isCharacterNotFound(errorText) {
		return domain.CharacterResult{}, validation.NewError(validation.ErrorEntityNotFound, "character not found", nil)
	}

	result := domain.CharacterResult{}
	result.CharacterInfo.IsBanned = isCharacterBanned(errorText)
	if result.CharacterInfo.IsBanned {
		result.CharacterInfo.BanReason = errorText
	}

	characterInfoContainer := findContainerByHeaders(doc, []string{"character information", "informacoes do personagem", "informações do personagem"})
	if characterInfoContainer != nil {
		info, parseErr := parseCharacterInfo(characterInfoContainer)
		if parseErr != nil {
			return domain.CharacterResult{}, parseErr
		}
		result.CharacterInfo = info
		if !result.CharacterInfo.IsBanned && isCharacterBanned(errorText) {
			result.CharacterInfo.IsBanned = true
			result.CharacterInfo.BanReason = errorText
		}
	}

	deathsContainer := findContainerByHeaders(doc, []string{"character deaths", "mortes"})
	if deathsContainer != nil {
		deaths, parseErr := parseCharacterDeaths(deathsContainer)
		if parseErr != nil {
			return domain.CharacterResult{}, parseErr
		}
		result.Deaths = deaths
	}

	accountInfoContainer := findContainerByHeaders(doc, []string{"account information", "informacoes da conta", "informações da conta"})
	if accountInfoContainer != nil {
		accountInfo, parseErr := parseAccountInformation(accountInfoContainer)
		if parseErr != nil {
			return domain.CharacterResult{}, parseErr
		}
		result.AccountInfo = accountInfo
	}

	otherCharactersContainer := findContainerByHeaders(doc, []string{"characters", "personagens"})
	if otherCharactersContainer != nil {
		result.OtherCharacters = parseOtherCharacters(otherCharactersContainer)
	}

	if strings.TrimSpace(result.CharacterInfo.Name) == "" {
		if result.CharacterInfo.IsBanned {
			return result, nil
		}
		return domain.CharacterResult{}, validation.NewError(validation.ErrorEntityNotFound, "character not found", nil)
	}

	return result, nil
}

func findContainerByHeaders(doc *goquery.Document, expected []string) *goquery.Selection {
	var found *goquery.Selection
	doc.Find(".TableContainer").EachWithBreak(func(_ int, container *goquery.Selection) bool {
		header := strings.ToLower(normalizeText(container.Find(".CaptionContainer .Text").First().Text()))
		for _, candidate := range expected {
			if strings.Contains(header, candidate) {
				found = container
				return false
			}
		}
		return true
	})
	return found
}

func parseCharacterInfo(container *goquery.Selection) (domain.CharacterInfo, error) {
	info := domain.CharacterInfo{}
	var parseErr error

	container.Find(".TableContent tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return true
		}

		label := normalizeLabel(cells.Eq(0).Text())
		valueCell := cells.Eq(1)
		valueText := normalizeText(valueCell.Text())

		switch strings.ToLower(label) {
		case "name", "nome":
			name := strings.TrimSpace(valueCell.Find("b").First().Text())
			if name == "" {
				name = strings.TrimSpace(valueText)
			}
			info.Name = stripCharacterNameMarkers(name)
			if strings.Contains(strings.ToLower(valueText), "(traded)") {
				info.Traded = true
			}
			if auctionURL, exists := valueCell.Find("a[href*='currentcharactertrades/']").Attr("href"); exists {
				info.AuctionURL = strings.TrimSpace(auctionURL)
			}

		case "former names", "nomes anteriores":
			info.FormerNames = splitCSV(valueText)

		case "sex", "sexo":
			info.Sex = valueText

		case "title", "titulo", "título":
			info.Title, info.UnlockedTitles = parseCharacterTitle(valueText)

		case "vocation", "vocação", "vocacao":
			info.Vocation = valueText

		case "level", "nível", "nivel":
			info.Level = parseInt(valueText)

		case "achievement points", "pontos de conquista":
			info.AchievementPoints = parseInt(valueText)

		case "world", "mundo":
			info.World = valueText

		case "former worlds", "mundos anteriores":
			info.FormerWorlds = splitCSV(valueText)

		case "residence", "residência", "residencia":
			info.Residence = valueText

		case "married to", "casado com", "casada com":
			info.MarriedTo = valueText

		case "house", "casa":
			info.Houses = parseCharacterHouses(valueCell)

		case "guild":
			info.Guild = parseCharacterGuild(valueCell, valueText)

		case "last login", "último login", "ultimo login":
			lastLogin, dateErr := parseRubinotDateTimeToUTC(valueText)
			if dateErr != nil {
				parseErr = validation.NewError(validation.ErrorUpstreamUnknown, dateErr.Error(), dateErr)
				return false
			}
			info.LastLogin = lastLogin

		case "account status", "status da conta":
			info.AccountStatus = valueText

		case "deletion date", "data de exclusão", "data de exclusao":
			deletionDate, dateErr := parseRubinotDateTimeToUTC(valueText)
			if dateErr != nil {
				parseErr = validation.NewError(validation.ErrorUpstreamUnknown, dateErr.Error(), dateErr)
				return false
			}
			info.DeletionDate = deletionDate

		case "comment", "comentário", "comentario":
			info.Comment = valueText
		}
		return true
	})

	if parseErr != nil {
		return domain.CharacterInfo{}, parseErr
	}

	return info, nil
}

func parseCharacterDeaths(container *goquery.Selection) ([]domain.CharacterDeath, error) {
	deaths := make([]domain.CharacterDeath, 0)

	for _, row := range container.Find(".TableContent tr").Slice(0, goquery.ToEnd).Nodes {
		selection := goquery.NewDocumentFromNode(row).Selection
		cells := selection.Find("td")
		if cells.Length() < 2 {
			continue
		}

		timeText := normalizeText(cells.Eq(0).Text())
		deathText := normalizeText(cells.Eq(1).Text())
		if timeText == "" || deathText == "" {
			continue
		}

		timeUTC, err := parseRubinotDateTimeToUTC(timeText)
		if err != nil {
			return nil, validation.NewError(validation.ErrorUpstreamUnknown, err.Error(), err)
		}

		death, ok := parseCharacterDeathText(timeUTC, deathText)
		if !ok {
			continue
		}
		deaths = append(deaths, death)
	}

	return deaths, nil
}

func parseCharacterDeathText(timeUTC, deathText string) (domain.CharacterDeath, bool) {
	match := characterDeathPattern.FindStringSubmatch(deathText)
	if len(match) != 3 {
		return domain.CharacterDeath{}, false
	}

	level, err := strconv.Atoi(strings.TrimSpace(match[1]))
	if err != nil {
		return domain.CharacterDeath{}, false
	}

	reason := parseDeathReason(deathText)
	targets := strings.TrimSpace(match[2])
	targets = characterParentheticalRe.ReplaceAllString(targets, "")
	targets = strings.TrimSpace(strings.TrimSuffix(targets, "."))
	if targets == "" {
		return domain.CharacterDeath{}, false
	}

	killersText := targets
	assistsText := ""
	if assistIdx := strings.Index(strings.ToLower(targets), " assisted by "); assistIdx >= 0 {
		killersText = strings.TrimSpace(targets[:assistIdx])
		assistsText = strings.TrimSpace(targets[assistIdx+len(" assisted by "):])
	}

	death := domain.CharacterDeath{
		Time:    timeUTC,
		Level:   level,
		Killers: splitKillParticipants(killersText),
		Assists: splitKillParticipants(assistsText),
		Reason:  reason,
	}
	return death, true
}

func parseAccountInformation(container *goquery.Selection) (domain.AccountInformation, error) {
	account := domain.AccountInformation{}
	var parseErr error

	container.Find(".TableContent tr").EachWithBreak(func(_ int, row *goquery.Selection) bool {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return true
		}
		label := strings.ToLower(normalizeLabel(cells.Eq(0).Text()))
		value := normalizeText(cells.Eq(1).Text())
		switch label {
		case "created", "criada", "criado":
			created, dateErr := parseRubinotDateTimeToUTC(value)
			if dateErr != nil {
				parseErr = validation.NewError(validation.ErrorUpstreamUnknown, dateErr.Error(), dateErr)
				return false
			}
			account.Created = created
		case "loyalty title", "titulo de lealdade", "título de lealdade":
			account.LoyaltyTitle = value
		}
		return true
	})

	if parseErr != nil {
		return domain.AccountInformation{}, parseErr
	}

	return account, nil
}

func parseOtherCharacters(container *goquery.Selection) []domain.OtherCharacter {
	characters := make([]domain.OtherCharacter, 0)

	container.Find(".TableContent tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 3 {
			return
		}

		nameRaw := normalizeText(cells.Eq(0).Text())
		world := normalizeText(cells.Eq(1).Text())
		statusRaw := normalizeText(cells.Eq(2).Text())
		if strings.EqualFold(nameRaw, "name") || strings.EqualFold(world, "world") {
			return
		}

		name := characterRowPrefixPattern.ReplaceAllString(nameRaw, "")
		main := strings.Contains(strings.ToLower(name), "(main character)")
		traded := strings.Contains(strings.ToLower(name), "(traded)")
		name = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(name, "(Main Character)", ""), "(Traded)", ""))

		deleted := strings.Contains(strings.ToLower(statusRaw), "deleted")
		status := "offline"
		if strings.Contains(strings.ToLower(statusRaw), "online") {
			status = "online"
		} else if deleted {
			status = "deleted"
		}

		if name == "" || world == "" {
			return
		}

		characters = append(characters, domain.OtherCharacter{
			Name:    name,
			World:   world,
			Status:  status,
			Main:    main,
			Traded:  traded,
			Deleted: deleted,
		})
	})

	return characters
}

func parseCharacterTitle(value string) (string, int) {
	match := characterTitlePattern.FindStringSubmatch(value)
	if len(match) != 3 {
		return value, 0
	}
	titles, err := strconv.Atoi(strings.TrimSpace(match[2]))
	if err != nil {
		return strings.TrimSpace(match[1]), 0
	}
	return strings.TrimSpace(match[1]), titles
}

func parseCharacterGuild(valueCell *goquery.Selection, valueText string) *domain.CharacterGuild {
	if valueText == "" {
		return nil
	}

	guild := &domain.CharacterGuild{}
	if anchorName := normalizeText(valueCell.Find("a").First().Text()); anchorName != "" {
		guild.Name = anchorName
	}

	if match := characterGuildRankPattern.FindStringSubmatch(valueText); len(match) == 3 {
		guild.Rank = strings.TrimSpace(match[1])
		if guild.Name == "" {
			guild.Name = strings.TrimSpace(match[2])
		}
	}

	if guild.Name == "" && guild.Rank == "" {
		return nil
	}
	return guild
}

func parseCharacterHouses(valueCell *goquery.Selection) []domain.CharacterHouse {
	houses := make([]domain.CharacterHouse, 0)
	valueCell.Find("a").Each(func(_ int, anchor *goquery.Selection) {
		house := domain.CharacterHouse{Name: normalizeText(anchor.Text())}
		href, _ := anchor.Attr("href")
		if idMatch := characterHouseIDPattern.FindStringSubmatch(href); len(idMatch) == 2 {
			house.HouseID, _ = strconv.Atoi(idMatch[1])
		}
		if parsedHref, err := url.Parse(href); err == nil {
			house.World = parsedHref.Query().Get("world")
		}
		if house.Name != "" {
			houses = append(houses, house)
		}
	})

	if len(houses) == 0 {
		plain := normalizeText(valueCell.Text())
		if plain != "" {
			houses = append(houses, domain.CharacterHouse{Name: plain})
		}
	}
	return houses
}

func splitKillParticipants(raw string) []string {
	clean := strings.TrimSpace(strings.TrimSuffix(raw, "."))
	if clean == "" {
		return nil
	}
	parts := strings.Split(clean, " and by ")
	participants := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		participants = append(participants, item)
	}
	if len(participants) == 0 {
		participants = append(participants, clean)
	}
	return participants
}

func parseDeathReason(raw string) string {
	matches := characterParentheticalRe.FindAllStringSubmatch(raw, -1)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		reason := strings.ToLower(strings.TrimSpace(match[1]))
		if reason == "soloed" || reason == "assisted" {
			continue
		}
		if reason != "" {
			return reason
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeLabel(raw string) string {
	return strings.TrimSpace(strings.TrimSuffix(normalizeText(raw), ":"))
}

func normalizeText(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func stripCharacterNameMarkers(name string) string {
	clean := strings.TrimSpace(name)
	clean = strings.ReplaceAll(clean, "(Traded)", "")
	clean = strings.ReplaceAll(clean, "(Main Character)", "")
	return strings.TrimSpace(clean)
}

func isCharacterNotFound(errorText string) bool {
	lower := strings.ToLower(errorText)
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "could not find character") ||
		strings.Contains(lower, "does not exist or has been deleted") ||
		strings.Contains(lower, "character not found") ||
		strings.Contains(lower, "não existe") ||
		strings.Contains(lower, "nao existe") ||
		strings.Contains(lower, "não foi encontrado") ||
		strings.Contains(lower, "nao foi encontrado")
}

func isCharacterBanned(errorText string) bool {
	lower := strings.ToLower(errorText)
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "banished") || strings.Contains(lower, "banned")
}
