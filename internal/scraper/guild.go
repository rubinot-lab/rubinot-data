package scraper

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/giovannirco/rubinot-data/internal/domain"
	"github.com/giovannirco/rubinot-data/internal/validation"
	"go.opentelemetry.io/otel/attribute"
)

var (
	guildTitlePattern       = regexp.MustCompile(`^(.*?)\s+-\s+Guilds\s+-\s+RubinOT$`)
	guildFoundedPattern     = regexp.MustCompile(`(?i)founded on\s+(.+?)\s+on\s+([A-Za-z]{3}\s+\d{1,2}\s+\d{4})`)
	guildDisbandDatePattern = regexp.MustCompile(`(?i)disband(?:ed|ing process and will be disbanded)\s+on\s+([^.]+)`)
)

func FetchGuild(ctx context.Context, baseURL, guildName string, opts FetchOptions) (domain.GuildResult, string, error) {
	ctx, span := tracer.Start(ctx, "scraper.FetchGuild")
	defer span.End()

	started := time.Now()
	sourceURL := fmt.Sprintf(
		"%s/?subtopic=guilds&page=view&GuildName=%s",
		strings.TrimRight(baseURL, "/"),
		url.QueryEscape(guildName),
	)
	client := NewClient(opts)

	span.SetAttributes(
		attribute.String("rubinot.endpoint", "guild"),
		attribute.String("rubinot.guild", guildName),
		attribute.String("rubinot.source_url", sourceURL),
	)

	htmlBody, err := client.Fetch(ctx, sourceURL)
	scrapeDuration.WithLabelValues("guild").Observe(time.Since(started).Seconds())
	if err != nil {
		scrapeRequests.WithLabelValues("guild", "error").Inc()
		return domain.GuildResult{}, sourceURL, err
	}
	scrapeRequests.WithLabelValues("guild", "ok").Inc()

	parseStarted := time.Now()
	result, parseErr := parseGuildHTML(guildName, htmlBody)
	parseDuration.WithLabelValues("guild").Observe(time.Since(parseStarted).Seconds())
	if parseErr != nil {
		return domain.GuildResult{}, sourceURL, parseErr
	}

	return result, sourceURL, nil
}

func parseGuildHTML(requestedName, htmlBody string) (domain.GuildResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		return domain.GuildResult{}, err
	}

	fullText := strings.ToLower(normalizeText(doc.Text()))
	if strings.Contains(fullText, "a guild by that name was not found") || strings.Contains(fullText, "guild not found") {
		return domain.GuildResult{}, validation.NewError(validation.ErrorEntityNotFound, "guild not found", nil)
	}

	result := domain.GuildResult{
		Name:   strings.TrimSpace(requestedName),
		Active: true,
	}

	if title := strings.TrimSpace(doc.Find("title").First().Text()); title != "" {
		if match := guildTitlePattern.FindStringSubmatch(title); len(match) == 2 && strings.TrimSpace(match[1]) != "" {
			result.Name = strings.TrimSpace(match[1])
		}
	}

	if hiddenGuildName, exists := doc.Find("input[name='GuildName']").First().Attr("value"); exists && strings.TrimSpace(hiddenGuildName) != "" {
		result.Name = strings.TrimSpace(hiddenGuildName)
	}

	infoContainer := doc.Find("#GuildInformationContainer").First()
	infoText := normalizeText(infoContainer.Text())
	if infoText == "" {
		return domain.GuildResult{}, validation.NewError(validation.ErrorEntityNotFound, "guild not found", nil)
	}

	parseGuildInfo(infoText, infoContainer, &result)

	membersContainer := findContainerByHeaders(doc, []string{"guild members", "membros da guild"})
	if membersContainer != nil {
		members, parseErr := parseGuildMembers(membersContainer)
		if parseErr != nil {
			return domain.GuildResult{}, parseErr
		}
		result.Members = members
	}

	invitesContainer := findContainerByHeaders(doc, []string{"invited characters", "personagens convidados"})
	if invitesContainer != nil {
		invites, parseErr := parseGuildInvites(invitesContainer)
		if parseErr != nil {
			return domain.GuildResult{}, parseErr
		}
		result.Invites = invites
	}

	result.MembersTotal = len(result.Members)
	result.MembersInvited = len(result.Invites)
	for _, member := range result.Members {
		if member.IsOnline {
			result.PlayersOnline++
		} else {
			result.PlayersOffline++
		}
	}

	return result, nil
}

func parseGuildInfo(infoText string, infoContainer *goquery.Selection, result *domain.GuildResult) {
	lowerInfo := strings.ToLower(infoText)

	if foundedMatch := guildFoundedPattern.FindStringSubmatch(infoText); len(foundedMatch) == 3 {
		result.World = strings.TrimSpace(foundedMatch[1])
		if foundedDate, err := parseRubinotDateToUTC(strings.TrimSpace(foundedMatch[2])); err == nil {
			result.Founded = foundedDate
		}
	}

	result.OpenApplications = strings.Contains(lowerInfo, "opened for applications")
	if strings.Contains(lowerInfo, "closed for applications") {
		result.OpenApplications = false
	}

	if strings.Contains(lowerInfo, "disband") {
		result.Active = false
		result.DisbandCondition = infoText
		if disbandMatch := guildDisbandDatePattern.FindStringSubmatch(infoText); len(disbandMatch) == 2 {
			rawDate := strings.TrimSpace(disbandMatch[1])
			if disbandDate, err := parseRubinotDateTimeToUTC(rawDate); err == nil {
				result.DisbandDate = disbandDate
			} else if dateOnly, dateErr := parseRubinotDateToUTC(rawDate); dateErr == nil {
				result.DisbandDate = dateOnly
			}
		}
	}

	result.InWar = strings.Contains(lowerInfo, "war against") ||
		strings.Contains(lowerInfo, "in war with") ||
		strings.Contains(lowerInfo, "currently at war") ||
		strings.Contains(lowerInfo, "is at war")

	if guildhallAnchor := infoContainer.Find("a[href*='subtopic=houses'][href*='houseid=']").First(); guildhallAnchor.Length() > 0 {
		hall := &domain.GuildHall{Name: normalizeText(guildhallAnchor.Text())}
		if href, exists := guildhallAnchor.Attr("href"); exists {
			if idMatch := characterHouseIDPattern.FindStringSubmatch(href); len(idMatch) == 2 {
				hall.HouseID = parseInt(idMatch[1])
			}
		}
		result.Guildhall = hall
	}
}

func parseGuildMembers(container *goquery.Selection) ([]domain.GuildMember, error) {
	members := make([]domain.GuildMember, 0)

	container.Find(".TableContent tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 6 {
			return
		}

		if strings.EqualFold(normalizeText(cells.Eq(0).Text()), "Rank") {
			return
		}

		rank := normalizeText(cells.Eq(0).Text())
		nameCell := cells.Eq(1)
		name := normalizeText(nameCell.Find("a").First().Text())
		if name == "" {
			name = normalizeText(nameCell.Text())
		}
		if name == "" {
			return
		}

		fullNameTitle := normalizeText(nameCell.Text())
		title := strings.TrimSpace(strings.TrimPrefix(fullNameTitle, name))

		member := domain.GuildMember{
			Rank:     rank,
			Name:     name,
			Title:    title,
			Vocation: normalizeText(cells.Eq(2).Text()),
			Level:    parseInt(normalizeText(cells.Eq(3).Text())),
			Status:   strings.ToLower(normalizeText(cells.Eq(5).Text())),
		}
		if strings.Contains(member.Status, "online") {
			member.IsOnline = true
			member.Status = "online"
		} else {
			member.Status = "offline"
		}

		if joinedRaw := normalizeText(cells.Eq(4).Text()); joinedRaw != "" {
			if joinedUTC, err := parseRubinotDateToUTC(joinedRaw); err == nil {
				member.Joined = joinedUTC
			}
		}

		members = append(members, member)
	})

	return members, nil
}

func parseGuildInvites(container *goquery.Selection) ([]domain.GuildInvite, error) {
	invites := make([]domain.GuildInvite, 0)

	container.Find(".TableContent tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}
		if strings.EqualFold(normalizeText(cells.Eq(0).Text()), "Name") {
			return
		}

		name := normalizeText(cells.Eq(0).Find("a").First().Text())
		if name == "" {
			name = normalizeText(cells.Eq(0).Text())
		}
		if name == "" {
			return
		}

		invite := domain.GuildInvite{Name: name}
		dateRaw := normalizeText(cells.Eq(1).Text())
		if dateRaw != "" {
			if inviteDate, err := parseRubinotDateToUTC(dateRaw); err == nil {
				invite.Date = inviteDate
			}
		}

		invites = append(invites, invite)
	})

	return invites, nil
}
