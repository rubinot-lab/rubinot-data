package scraper

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

type PlayerOnline struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Vocation string `json:"vocation"`
}

type WorldInfo struct {
	Status        string `json:"status,omitempty"`
	PlayersOnline int    `json:"players_online"`
	Location      string `json:"location,omitempty"`
	PVPType       string `json:"pvp_type,omitempty"`
	CreationDate  string `json:"creation_date,omitempty"`
}

type WorldResult struct {
	Name          string         `json:"name"`
	Info          WorldInfo      `json:"info"`
	PlayersOnline []PlayerOnline `json:"players_online_list"`
}

func FetchWorld(baseURL, world string) (WorldResult, string, error) {
	formatted := strings.Title(strings.ToLower(strings.TrimSpace(world)))
	sourceURL := fmt.Sprintf("%s/?subtopic=worlds&world=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(formatted))

	client := resty.New().SetTimeout(10 * 1000 * 1000 * 1000).SetRetryCount(2)
	res, err := client.R().SetHeader("User-Agent", "rubinot-data/0.1").Get(sourceURL)
	if err != nil {
		return WorldResult{}, sourceURL, err
	}
	if res.StatusCode() != 200 {
		return WorldResult{}, sourceURL, fmt.Errorf("unexpected status from upstream: %d", res.StatusCode())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(res.Body())))
	if err != nil {
		return WorldResult{}, sourceURL, err
	}

	result := WorldResult{Name: formatted}
	result.Info = parseWorldInfo(doc)
	result.PlayersOnline = parsePlayers(doc)

	if result.Info.PlayersOnline == 0 {
		result.Info.PlayersOnline = len(result.PlayersOnline)
	}

	return result, sourceURL, nil
}

func parseWorldInfo(doc *goquery.Document) WorldInfo {
	info := WorldInfo{}
	doc.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() != 2 {
			return
		}
		label := strings.TrimSpace(strings.TrimSuffix(tds.Eq(0).Text(), ":"))
		value := strings.TrimSpace(tds.Eq(1).Text())
		switch strings.ToLower(label) {
		case "status":
			info.Status = value
		case "players online":
			info.PlayersOnline = parseInt(value)
		case "location":
			info.Location = value
		case "pvp type":
			info.PVPType = value
		case "creation date":
			info.CreationDate = value
		}
	})
	return info
}

func parsePlayers(doc *goquery.Document) []PlayerOnline {
	out := make([]PlayerOnline, 0)
	doc.Find("table").EachWithBreak(func(_ int, table *goquery.Selection) bool {
		headers := strings.ToLower(strings.Join(strings.Fields(table.Find("tr").First().Text()), " "))
		if !(strings.Contains(headers, "name") && strings.Contains(headers, "level") && strings.Contains(headers, "vocation")) {
			return true
		}

		table.Find("tr").Slice(1, goquery.ToEnd).Each(func(_ int, tr *goquery.Selection) {
			tds := tr.Find("td")
			if tds.Length() < 3 {
				return
			}
			name := strings.TrimSpace(tds.Eq(0).Text())
			lvl := parseInt(strings.TrimSpace(tds.Eq(1).Text()))
			voc := strings.TrimSpace(tds.Eq(2).Text())
			if name == "" || lvl <= 0 || voc == "" {
				return
			}
			out = append(out, PlayerOnline{Name: name, Level: lvl, Vocation: voc})
		})
		return false
	})
	return out
}

func parseInt(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	i, _ := strconv.Atoi(s)
	return i
}
