package scraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

var brtLocation = time.FixedZone("BRT", -3*3600)

func adaptResponse(upstreamPath string, rubinidataBody string) (string, error) {
	parsed, err := url.Parse(upstreamPath)
	if err != nil {
		return rubinidataBody, nil
	}
	p := parsed.Path

	switch {
	case p == "/api/worlds":
		return adaptWorldsResponse(rubinidataBody)
	case strings.HasPrefix(p, "/api/worlds/"):
		return adaptWorldDetailResponse(rubinidataBody)
	case p == "/api/characters/search":
		return adaptCharacterResponse(rubinidataBody)
	case p == "/api/guilds" && parsed.Query().Get("world") != "":
		return adaptGuildsListResponse(rubinidataBody)
	case strings.HasPrefix(p, "/api/guilds/"):
		return adaptGuildDetailResponse(rubinidataBody)
	case p == "/api/highscores":
		return adaptHighscoresResponse(rubinidataBody)
	case p == "/api/killstats":
		return adaptKillstatisticsResponse(rubinidataBody)
	case p == "/api/deaths":
		return adaptDeathsResponse(rubinidataBody)
	case p == "/api/bans":
		return adaptBanishmentsResponse(rubinidataBody)
	case p == "/api/transfers":
		return adaptTransfersResponse(rubinidataBody)
	case p == "/api/boosted":
		return adaptBoostedResponse(rubinidataBody)
	default:
		return rubinidataBody, nil
	}
}

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal adapted response: %w", err)
	}
	return string(b), nil
}

func parseBRTDate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	t, err := time.ParseInLocation("Jan 02 2006, 15:04:05 BRT", s, brtLocation)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func parseSimpleDate(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	t, err := time.ParseInLocation("Jan 02 2006", s, time.UTC)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func parseISO8601(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func parseOutfitURL(outfitURL string) (looktype, head, body, legs, feet, addons int) {
	parsed, err := url.Parse(outfitURL)
	if err != nil {
		return
	}
	q := parsed.Query()
	looktype, _ = strconv.Atoi(q.Get("type"))
	head, _ = strconv.Atoi(q.Get("head"))
	body, _ = strconv.Atoi(q.Get("body"))
	legs, _ = strconv.Atoi(q.Get("legs"))
	feet, _ = strconv.Atoi(q.Get("feet"))
	addons, _ = strconv.Atoi(q.Get("addons"))
	return
}

func vocationNameToUpstreamID(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "knight", "elite knight":
		return 5
	case "paladin", "royal paladin":
		return 4
	case "sorcerer", "master sorcerer":
		return 2
	case "druid", "elder druid":
		return 3
	case "monk", "exalted monk":
		return 9
	case "none":
		return 1
	default:
		return 0
	}
}

func extractFilename(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return path.Base(parsed.Path)
}

func hasNestedObject(body string, key string) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &probe); err != nil {
		return false
	}
	raw, ok := probe[key]
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(string(raw))
	return len(trimmed) > 0 && trimmed[0] == '{'
}

func adaptWorldsResponse(body string) (string, error) {
	if !hasNestedObject(body, "worlds") {
		return body, nil
	}

	var src struct {
		Worlds struct {
			Overview struct {
				TotalPlayersOnline int    `json:"total_players_online"`
				OverallMaximum     int    `json:"overall_maximum"`
				MaximumDate        string `json:"maximum_date"`
			} `json:"overview"`
			RegularWorlds []struct {
				Name          string `json:"name"`
				PlayersOnline int    `json:"players_online"`
				PVPType       string `json:"pvp_type"`
				PVPTypeLabel  string `json:"pvp_type_label"`
				WorldType     string `json:"world_type"`
				Locked        bool   `json:"locked"`
				ID            int    `json:"id"`
			} `json:"regular_worlds"`
		} `json:"worlds"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return body, nil
	}

	type worldEntry struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		PVPType       string `json:"pvpType"`
		PVPTypeLabel  string `json:"pvpTypeLabel"`
		WorldType     string `json:"worldType"`
		Locked        bool   `json:"locked"`
		PlayersOnline int    `json:"playersOnline"`
	}

	worlds := make([]worldEntry, 0, len(src.Worlds.RegularWorlds))
	for _, w := range src.Worlds.RegularWorlds {
		worlds = append(worlds, worldEntry{
			ID:            w.ID,
			Name:          w.Name,
			PVPType:       w.PVPType,
			PVPTypeLabel:  w.PVPTypeLabel,
			WorldType:     w.WorldType,
			Locked:        w.Locked,
			PlayersOnline: w.PlayersOnline,
		})
	}

	out := struct {
		Worlds            []worldEntry `json:"worlds"`
		TotalOnline       int          `json:"totalOnline"`
		OverallRecord     int          `json:"overallRecord"`
		OverallRecordTime int64        `json:"overallRecordTime"`
	}{
		Worlds:            worlds,
		TotalOnline:       src.Worlds.Overview.TotalPlayersOnline,
		OverallRecord:     src.Worlds.Overview.OverallMaximum,
		OverallRecordTime: parseBRTDate(src.Worlds.Overview.MaximumDate),
	}

	return marshalJSON(out)
}

func adaptWorldDetailResponse(body string) (string, error) {
	var src struct {
		World struct {
			Name         string `json:"name"`
			Status       string `json:"status"`
			PlayersOnline int   `json:"players_online"`
			OnlineRecord struct {
				Players int    `json:"players"`
				Date    string `json:"date"`
			} `json:"online_record"`
			CreationDate string `json:"creation_date"`
			PVPType      string `json:"pvp_type"`
			PVPTypeLabel string `json:"pvp_type_label"`
			WorldType    string `json:"world_type"`
			Locked       bool   `json:"locked"`
			ID           int    `json:"id"`
		} `json:"world"`
		Players []struct {
			Name       string `json:"name"`
			Level      int    `json:"level"`
			Vocation   string `json:"vocation"`
			VocationID int    `json:"vocation_id"`
		} `json:"players"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata world detail: %w", err)
	}

	type playerEntry struct {
		Name       string `json:"name"`
		Level      int    `json:"level"`
		Vocation   string `json:"vocation"`
		VocationID int    `json:"vocationId"`
	}

	players := make([]playerEntry, 0, len(src.Players))
	for _, p := range src.Players {
		players = append(players, playerEntry{
			Name:       p.Name,
			Level:      p.Level,
			Vocation:   p.Vocation,
			VocationID: p.VocationID,
		})
	}

	out := struct {
		World struct {
			ID           int    `json:"id"`
			Name         string `json:"name"`
			PVPType      string `json:"pvpType"`
			PVPTypeLabel string `json:"pvpTypeLabel"`
			WorldType    string `json:"worldType"`
			Locked       bool   `json:"locked"`
			CreationDate int64  `json:"creationDate"`
		} `json:"world"`
		PlayersOnline int           `json:"playersOnline"`
		Record        int           `json:"record"`
		RecordTime    int64         `json:"recordTime"`
		Players       []playerEntry `json:"players"`
	}{
		PlayersOnline: src.World.PlayersOnline,
		Record:        src.World.OnlineRecord.Players,
		RecordTime:    parseBRTDate(src.World.OnlineRecord.Date),
		Players:       players,
	}
	out.World.ID = src.World.ID
	out.World.Name = src.World.Name
	out.World.PVPType = src.World.PVPType
	out.World.PVPTypeLabel = src.World.PVPTypeLabel
	out.World.WorldType = src.World.WorldType
	out.World.Locked = src.World.Locked
	out.World.CreationDate = parseSimpleDate(src.World.CreationDate)

	return marshalJSON(out)
}

func adaptCharacterResponse(body string) (string, error) {
	if !hasNestedObject(body, "characters") {
		return body, nil
	}

	var src struct {
		Characters struct {
			Character struct {
				ID                int      `json:"id"`
				Name              string   `json:"name"`
				Traded            bool     `json:"traded"`
				Level             int      `json:"level"`
				Vocation          string   `json:"vocation"`
				WorldName         string   `json:"world_name"`
				WorldID           int      `json:"world_id"`
				Sex               string   `json:"sex"`
				AchievementPoints int      `json:"achievement_points"`
				Residence         string   `json:"residence"`
				LastLogin         string   `json:"last_login"`
				AccountStatus     string   `json:"account_status"`
				House             string   `json:"house"`
				FormerNames       []string `json:"former_names"`
				FoundByOldName    bool     `json:"found_by_old_name"`
				OutfitURL         string   `json:"outfit_url"`
				LoyaltyPoints     int      `json:"loyalty_points"`
				Created           string   `json:"created"`
			} `json:"character"`
			OtherCharacters []struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Level     int    `json:"level"`
				Vocation  string `json:"vocation"`
				WorldID   int    `json:"world_id"`
				WorldName string `json:"world_name"`
			} `json:"other_characters"`
		} `json:"characters"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata character: %w", err)
	}

	ch := src.Characters.Character
	looktype, lookhead, lookbody, looklegs, lookfeet, lookaddons := parseOutfitURL(ch.OutfitURL)

	type otherChar struct {
		Name     string `json:"name"`
		World    string `json:"world"`
		WorldID  int    `json:"world_id"`
		Level    int    `json:"level"`
		Vocation string `json:"vocation"`
		IsOnline bool   `json:"isOnline"`
	}

	others := make([]otherChar, 0, len(src.Characters.OtherCharacters))
	for _, oc := range src.Characters.OtherCharacters {
		others = append(others, otherChar{
			Name:     oc.Name,
			World:    oc.WorldName,
			WorldID:  oc.WorldID,
			Level:    oc.Level,
			Vocation: oc.Vocation,
		})
	}

	formerNames := ch.FormerNames
	if formerNames == nil {
		formerNames = []string{}
	}

	player := map[string]any{
		"id":                nil,
		"account_id":       nil,
		"name":             ch.Name,
		"level":            ch.Level,
		"vocation":         ch.Vocation,
		"vocationId":       vocationNameToUpstreamID(ch.Vocation),
		"world_id":         ch.WorldID,
		"sex":              ch.Sex,
		"residence":        ch.Residence,
		"lastlogin":        fmt.Sprintf("%d", parseISO8601(ch.LastLogin)),
		"created":          parseISO8601(ch.Created),
		"comment":          "",
		"account_created":  0,
		"loyalty_points":   ch.LoyaltyPoints,
		"isHidden":         false,
		"guild":            nil,
		"house":            nil,
		"partner":          nil,
		"formerNames":      formerNames,
		"title":            nil,
		"auction":          nil,
		"looktype":         looktype,
		"lookhead":         lookhead,
		"lookbody":         lookbody,
		"looklegs":         looklegs,
		"lookfeet":         lookfeet,
		"lookaddons":       lookaddons,
		"vip_time":         0,
		"achievementPoints": ch.AchievementPoints,
	}

	out := map[string]any{
		"player":                       player,
		"deaths":                       []any{},
		"otherCharacters":              others,
		"accountBadges":                []any{},
		"displayedAchievements":        []any{},
		"banInfo":                       nil,
		"canSeeCharacterIdentifiers":   false,
		"canSeeDeathDetails":           false,
		"foundByOldName":               ch.FoundByOldName,
	}

	return marshalJSON(out)
}

func adaptDeathsResponse(body string) (string, error) {
	if !hasNestedObject(body, "deaths") {
		return body, nil
	}

	var src struct {
		Deaths struct {
			Entries []struct {
				Name     string `json:"name"`
				Level    int    `json:"level"`
				Killers  []struct {
					Name   string `json:"name"`
					Player bool   `json:"player"`
				} `json:"killers"`
				Time     string `json:"time"`
				Datetime string `json:"datetime"`
				WorldID  int    `json:"world_id"`
				PlayerID int    `json:"player_id"`
			} `json:"entries"`
		} `json:"deaths"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata deaths: %w", err)
	}

	type deathEntry struct {
		PlayerID           int    `json:"player_id"`
		Time               string `json:"time"`
		Level              int    `json:"level"`
		KilledBy           string `json:"killed_by"`
		IsPlayer           int    `json:"is_player"`
		MostDamageBy       string `json:"mostdamage_by"`
		MostDamageIsPlayer int    `json:"mostdamage_is_player"`
		Victim             string `json:"victim"`
		WorldID            int    `json:"world_id"`
	}

	deaths := make([]deathEntry, 0, len(src.Deaths.Entries))
	for _, e := range src.Deaths.Entries {
		killedBy := ""
		isPlayer := 0
		mostDamageBy := ""
		mostDamageIsPlayer := 0

		if len(e.Killers) > 0 {
			first := e.Killers[0]
			killedBy = first.Name
			if first.Player {
				isPlayer = 1
			}
			last := e.Killers[len(e.Killers)-1]
			mostDamageBy = last.Name
			if last.Player {
				mostDamageIsPlayer = 1
			}
		}

		ts := fmt.Sprintf("%d", parseISO8601(e.Datetime))

		deaths = append(deaths, deathEntry{
			PlayerID:           e.PlayerID,
			Time:               ts,
			Level:              e.Level,
			KilledBy:           killedBy,
			IsPlayer:           isPlayer,
			MostDamageBy:       mostDamageBy,
			MostDamageIsPlayer: mostDamageIsPlayer,
			Victim:             e.Name,
			WorldID:            e.WorldID,
		})
	}

	totalCount := len(deaths)
	out := struct {
		Deaths     []deathEntry `json:"deaths"`
		Pagination struct {
			CurrentPage  int `json:"currentPage"`
			TotalPages   int `json:"totalPages"`
			TotalCount   int `json:"totalCount"`
			ItemsPerPage int `json:"itemsPerPage"`
		} `json:"pagination"`
	}{
		Deaths: deaths,
	}
	out.Pagination.CurrentPage = 1
	out.Pagination.TotalPages = 1
	out.Pagination.TotalCount = totalCount
	out.Pagination.ItemsPerPage = totalCount

	return marshalJSON(out)
}

func adaptHighscoresResponse(body string) (string, error) {
	if !hasNestedObject(body, "highscores") {
		return body, nil
	}

	var src struct {
		Highscores struct {
			Category      string `json:"category"`
			World         string `json:"world"`
			HighscoreList []struct {
				Rank      int    `json:"rank"`
				ID        int    `json:"id"`
				Name      string `json:"name"`
				Vocation  string `json:"vocation"`
				WorldID   int    `json:"world_id"`
				WorldName string `json:"world_name"`
				Level     int    `json:"level"`
				Value     int64  `json:"value"`
			} `json:"highscore_list"`
		} `json:"highscores"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata highscores: %w", err)
	}

	type playerEntry struct {
		Rank      int    `json:"rank"`
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Level     int    `json:"level"`
		Vocation  int    `json:"vocation"`
		WorldID   int    `json:"world_id"`
		WorldName string `json:"worldName"`
		Value     int64  `json:"value"`
	}

	players := make([]playerEntry, 0, len(src.Highscores.HighscoreList))
	for _, h := range src.Highscores.HighscoreList {
		players = append(players, playerEntry{
			Rank:      h.Rank,
			ID:        h.ID,
			Name:      h.Name,
			Level:     h.Level,
			Vocation:  vocationNameToUpstreamID(h.Vocation),
			WorldID:   h.WorldID,
			WorldName: h.WorldName,
			Value:     h.Value,
		})
	}

	out := struct {
		Players          []playerEntry `json:"players"`
		TotalCount       int           `json:"totalCount"`
		CachedAt         int64         `json:"cachedAt"`
		AvailableSeasons []int         `json:"availableSeasons"`
	}{
		Players:          players,
		TotalCount:       len(players),
		CachedAt:         time.Now().UnixMilli(),
		AvailableSeasons: []int{},
	}

	return marshalJSON(out)
}

func adaptGuildsListResponse(body string) (string, error) {
	if !hasNestedObject(body, "guilds") {
		return body, nil
	}

	var src struct {
		Guilds struct {
			Guilds []struct {
				Name        string `json:"name"`
				LogoURL     string `json:"logo_url"`
				Description string `json:"description"`
				ID          int    `json:"id"`
				WorldID     int    `json:"world_id"`
			} `json:"guilds"`
		} `json:"guilds"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata guilds list: %w", err)
	}

	type guildEntry struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		WorldID     int    `json:"world_id"`
		LogoName    string `json:"logo_name"`
	}

	guilds := make([]guildEntry, 0, len(src.Guilds.Guilds))
	for _, g := range src.Guilds.Guilds {
		guilds = append(guilds, guildEntry{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			WorldID:     g.WorldID,
			LogoName:    extractFilename(g.LogoURL),
		})
	}

	out := struct {
		Guilds      []guildEntry `json:"guilds"`
		TotalCount  int          `json:"totalCount"`
		TotalPages  int          `json:"totalPages"`
		CurrentPage int          `json:"currentPage"`
	}{
		Guilds:      guilds,
		TotalCount:  len(guilds),
		TotalPages:  1,
		CurrentPage: 1,
	}

	return marshalJSON(out)
}

func adaptGuildDetailResponse(body string) (string, error) {
	var src struct {
		Guild struct {
			Name             string `json:"name"`
			WorldID          int    `json:"world_id"`
			LogoURL          string `json:"logo_url"`
			Description      string `json:"description"`
			Founded          string `json:"founded"`
			Active           bool   `json:"active"`
			GuildBankBalance string `json:"guild_bank_balance"`
			MembersTotal     int    `json:"members_total"`
			MembersOnline    int    `json:"members_online"`
			Members          []struct {
				Name        string `json:"name"`
				Title       string `json:"title"`
				Rank        string `json:"rank"`
				Vocation    string `json:"vocation"`
				Level       int    `json:"level"`
				JoiningDate string `json:"joining_date"`
				IsOnline    bool   `json:"is_online"`
				ID          int    `json:"id"`
			} `json:"members"`
		} `json:"guild"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata guild detail: %w", err)
	}

	type memberEntry struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Level     int    `json:"level"`
		Vocation  int    `json:"vocation"`
		Rank      string `json:"rank"`
		RankLevel int    `json:"rankLevel"`
		Nick      string `json:"nick"`
		JoinDate  int64  `json:"joinDate"`
		IsOnline  bool   `json:"isOnline"`
	}

	type ownerEntry struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Level    int    `json:"level"`
		Vocation int    `json:"vocation"`
	}

	type rankEntry struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Level int    `json:"level"`
	}

	members := make([]memberEntry, 0, len(src.Guild.Members))
	rankSet := make(map[string]bool)
	var owner *ownerEntry

	for _, m := range src.Guild.Members {
		vocID := vocationNameToUpstreamID(m.Vocation)
		me := memberEntry{
			ID:       m.ID,
			Name:     m.Name,
			Level:    m.Level,
			Vocation: vocID,
			Rank:     m.Rank,
			Nick:     m.Title,
			JoinDate: parseSimpleDate(m.JoiningDate),
			IsOnline: m.IsOnline,
		}
		members = append(members, me)
		rankSet[m.Rank] = true

		if strings.EqualFold(m.Rank, "Leader") && owner == nil {
			owner = &ownerEntry{
				ID:       m.ID,
				Name:     m.Name,
				Level:    m.Level,
				Vocation: vocID,
			}
		}
	}

	ranks := make([]rankEntry, 0, len(rankSet))
	rankID := 1
	for name := range rankSet {
		ranks = append(ranks, rankEntry{
			ID:   rankID,
			Name: name,
		})
		rankID++
	}

	logoName := extractFilename(src.Guild.LogoURL)

	type guildOut struct {
		ID           int           `json:"id"`
		Name         string        `json:"name"`
		MOTD         string        `json:"motd"`
		Description  string        `json:"description"`
		Homepage     string        `json:"homepage"`
		WorldID      int           `json:"world_id"`
		LogoName     string        `json:"logo_name"`
		Balance      interface{}   `json:"balance"`
		CreationData int64         `json:"creationdata"`
		Owner        *ownerEntry   `json:"owner"`
		Members      []memberEntry `json:"members"`
		Ranks        []rankEntry   `json:"ranks"`
		Residence    interface{}   `json:"residence"`
	}

	out := struct {
		Guild guildOut `json:"guild"`
	}{
		Guild: guildOut{
			Name:         src.Guild.Name,
			WorldID:      src.Guild.WorldID,
			LogoName:     logoName,
			Description:  src.Guild.Description,
			CreationData: parseSimpleDate(src.Guild.Founded),
			Balance:      src.Guild.GuildBankBalance,
			Owner:        owner,
			Members:      members,
			Ranks:        ranks,
		},
	}

	return marshalJSON(out)
}

func adaptKillstatisticsResponse(body string) (string, error) {
	return body, nil
}

func adaptBanishmentsResponse(body string) (string, error) {
	if !hasNestedObject(body, "banishments") {
		return body, nil
	}

	var src struct {
		Banishments struct {
			Entries []struct {
				AccountID   int    `json:"account_id"`
				AccountName string `json:"account_name"`
				Character   string `json:"character"`
				Reason      string `json:"reason"`
				BannedAt    string `json:"banned_at"`
				ExpiresAt   string `json:"expires_at"`
				BannedBy    string `json:"banned_by"`
				IsPermanent bool   `json:"is_permanent"`
			} `json:"entries"`
		} `json:"banishments"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata banishments: %w", err)
	}

	type banEntry struct {
		AccountID     int    `json:"account_id"`
		AccountName   string `json:"account_name"`
		MainCharacter string `json:"main_character"`
		Reason        string `json:"reason"`
		BannedAt      string `json:"banned_at"`
		ExpiresAt     string `json:"expires_at"`
		BannedBy      string `json:"banned_by"`
		IsPermanent   bool   `json:"is_permanent"`
	}

	bans := make([]banEntry, 0, len(src.Banishments.Entries))
	for _, e := range src.Banishments.Entries {
		bans = append(bans, banEntry{
			AccountID:     e.AccountID,
			AccountName:   e.AccountName,
			MainCharacter: e.Character,
			Reason:        e.Reason,
			BannedAt:      e.BannedAt,
			ExpiresAt:     e.ExpiresAt,
			BannedBy:      e.BannedBy,
			IsPermanent:   e.IsPermanent,
		})
	}

	out := struct {
		Bans        []banEntry `json:"bans"`
		TotalCount  int        `json:"totalCount"`
		TotalPages  int        `json:"totalPages"`
		CurrentPage int        `json:"currentPage"`
		CachedAt    int64      `json:"cachedAt"`
	}{
		Bans:        bans,
		TotalCount:  len(bans),
		TotalPages:  1,
		CurrentPage: 1,
		CachedAt:    time.Now().UnixMilli(),
	}

	return marshalJSON(out)
}

func adaptTransfersResponse(body string) (string, error) {
	if !hasNestedObject(body, "transfers") {
		return body, nil
	}

	var src struct {
		Transfers struct {
			Entries []struct {
				Name         string `json:"name"`
				Level        int    `json:"level"`
				FromWorld    string `json:"from_world"`
				ToWorld      string `json:"to_world"`
				TransferDate string `json:"transfer_date"`
			} `json:"entries"`
		} `json:"transfers"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata transfers: %w", err)
	}

	type transferEntry struct {
		ID            int         `json:"id"`
		PlayerID      int         `json:"player_id"`
		PlayerName    string      `json:"player_name"`
		PlayerLevel   int         `json:"player_level"`
		FromWorldID   int         `json:"from_world_id"`
		ToWorldID     int         `json:"to_world_id"`
		FromWorld     string      `json:"from_world"`
		ToWorld       string      `json:"to_world"`
		TransferredAt interface{} `json:"transferred_at"`
	}

	transfers := make([]transferEntry, 0, len(src.Transfers.Entries))
	for _, e := range src.Transfers.Entries {
		ts := parseISO8601(e.TransferDate)
		var transferredAt interface{}
		if ts > 0 {
			transferredAt = ts
		}
		transfers = append(transfers, transferEntry{
			PlayerName:    e.Name,
			PlayerLevel:   e.Level,
			FromWorld:     e.FromWorld,
			ToWorld:       e.ToWorld,
			TransferredAt: transferredAt,
		})
	}

	out := struct {
		Transfers    []transferEntry `json:"transfers"`
		TotalResults int             `json:"totalResults"`
		TotalPages   int             `json:"totalPages"`
		CurrentPage  int             `json:"currentPage"`
	}{
		Transfers:    transfers,
		TotalResults: len(transfers),
		TotalPages:   1,
		CurrentPage:  1,
	}

	return marshalJSON(out)
}

func adaptBoostedResponse(body string) (string, error) {
	if !hasNestedObject(body, "boosted") {
		return body, nil
	}

	var src struct {
		Boosted struct {
			Creature struct {
				Name     string `json:"name"`
				ID       int    `json:"id"`
				LookType int    `json:"looktype"`
				ImageURL string `json:"image_url"`
			} `json:"creature"`
			Boss struct {
				Name     string `json:"name"`
				ID       int    `json:"id"`
				LookType int    `json:"looktype"`
				ImageURL string `json:"image_url"`
			} `json:"boss"`
		} `json:"boosted"`
	}
	if err := json.Unmarshal([]byte(body), &src); err != nil {
		return "", fmt.Errorf("decode rubinidata boosted: %w", err)
	}

	out := struct {
		Boss struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			LookType int    `json:"looktype"`
		} `json:"boss"`
		Monster struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			LookType int    `json:"looktype"`
		} `json:"monster"`
	}{}

	out.Boss.ID = src.Boosted.Boss.ID
	out.Boss.Name = src.Boosted.Boss.Name
	out.Boss.LookType = src.Boosted.Boss.LookType
	out.Monster.ID = src.Boosted.Creature.ID
	out.Monster.Name = src.Boosted.Creature.Name
	out.Monster.LookType = src.Boosted.Creature.LookType

	return marshalJSON(out)
}
