package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	jellyfin "github.com/sj14/jellyfin-go/api"
)

type NowPlaying struct {
	Title     string
	TVDBID    int32
	Season    int32
	Episode   int32
	UserName  string
	UserID    string
	SessionID string
}

type JellyfinClient struct {
	api *jellyfin.APIClient
}

func NewJellyfinClient(cfg ServerConfig) *JellyfinClient {
	apiCfg := &jellyfin.Configuration{
		Servers: jellyfin.ServerConfigurations{{URL: strings.TrimRight(cfg.URL, "/")}},
		DefaultHeader: map[string]string{
			"Authorization": fmt.Sprintf(`MediaBrowser Token="%s"`, cfg.APIKey),
		},
	}
	return &JellyfinClient{api: jellyfin.NewAPIClient(apiCfg)}
}

func (c *JellyfinClient) NowPlaying(ctx context.Context) ([]NowPlaying, error) {
	sessions, _, err := c.api.SessionAPI.GetSessions(ctx).Execute()
	if err != nil {
		return nil, err
	}

	var result []NowPlaying
	for _, session := range sessions {
		item, ok := session.GetNowPlayingItemOk()
		if !ok || item == nil || item.Type == nil || *item.Type != jellyfin.BASEITEMKIND_EPISODE {
			continue
		}

		seriesID, ok := item.GetSeriesIdOk()
		if !ok || seriesID == nil || *seriesID == "" {
			continue
		}
		series, _, err := c.api.UserLibraryAPI.GetItem(ctx, *seriesID).UserId(session.GetUserId()).Execute()
		if err != nil {
			continue
		}
		seriesTVDBID, err := extractTVDBID(series)
		if err != nil {
			continue
		}

		season, ok := item.GetParentIndexNumberOk()
		if !ok || season == nil {
			continue
		}
		episode, ok := item.GetIndexNumberOk()
		if !ok || episode == nil {
			continue
		}

		result = append(result, NowPlaying{
			Title:     item.GetSeriesName(),
			TVDBID:    seriesTVDBID,
			Season:    *season,
			Episode:   *episode,
			UserName:  session.GetUserName(),
			UserID:    session.GetUserId(),
			SessionID: session.GetId(),
		})
	}
	return result, nil
}

func extractTVDBID(item *jellyfin.BaseItemDto) (int32, error) {
	providerIDs := item.GetProviderIds()
	for _, key := range []string{"Tvdb", "TVDB", "tvdb"} {
		if value, ok := providerIDs[key]; ok && value != "" {
			id, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return 0, err
			}
			return int32(id), nil
		}
	}
	return 0, fmt.Errorf("missing TVDB provider id")
}
