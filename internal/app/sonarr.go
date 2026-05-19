package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	sonarr "github.com/devopsarr/sonarr-go/sonarr"
)

type SonarrClient struct {
	api    *sonarr.APIClient
	ctx    context.Context
	base   string
	apiKey string
	http   *http.Client
}

func NewSonarrClient(cfg ServerConfig) *SonarrClient {
	base := strings.TrimRight(cfg.URL, "/")
	apiCfg := sonarr.NewConfiguration()
	apiCfg.Servers = sonarr.ServerConfigurations{{URL: base}}
	apiCfg.AddDefaultHeader("X-Api-Key", cfg.APIKey)

	ctx := context.WithValue(context.Background(), sonarr.ContextAPIKeys, map[string]sonarr.APIKey{
		"apikey": {Key: cfg.APIKey},
	})

	return &SonarrClient{
		api:    sonarr.NewAPIClient(apiCfg),
		ctx:    ctx,
		base:   base,
		apiKey: cfg.APIKey,
		http:   http.DefaultClient,
	}
}

func (c *SonarrClient) SeriesByTVDB(ctx context.Context, tvdbID int32) (*sonarr.SeriesResource, error) {
	reqCtx := mergeContext(ctx, c.ctx)
	series, _, err := c.api.SeriesAPI.ListSeries(reqCtx).TvdbId(tvdbID).Execute()
	if err != nil {
		return nil, err
	}
	for i := range series {
		if series[i].GetTvdbId() == tvdbID {
			return &series[i], nil
		}
	}
	return nil, fmt.Errorf("series with tvdb id %d not found in Sonarr", tvdbID)
}

func (c *SonarrClient) MonitorSeason(ctx context.Context, series *sonarr.SeriesResource, season int32) error {
	changed := false
	if series.Monitored == nil || !*series.Monitored {
		series.SetMonitored(true)
		changed = true
	}

	found := false
	for i := range series.Seasons {
		if series.Seasons[i].GetSeasonNumber() != season {
			continue
		}
		found = true
		if series.Seasons[i].Monitored == nil || !*series.Seasons[i].Monitored {
			series.Seasons[i].SetMonitored(true)
			changed = true
		}
		break
	}
	if !found {
		return fmt.Errorf("series %q has no season %d", series.GetTitle(), season)
	}
	if !changed {
		return nil
	}

	id := strconv.Itoa(int(series.GetId()))
	_, _, err := c.api.SeriesAPI.UpdateSeries(mergeContext(ctx, c.ctx), id).MoveFiles(false).SeriesResource(*series).Execute()
	return err
}

func (c *SonarrClient) SearchSeason(ctx context.Context, seriesID int32, season int32) error {
	body := map[string]any{
		"name":         "SeasonSearch",
		"seriesId":     seriesID,
		"seasonNumber": season,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	u, err := url.Parse(c.base + "/api/v3/command")
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("apikey", c.apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sonarr command failed: %s", resp.Status)
	}
	return nil
}

func seasonExists(series *sonarr.SeriesResource, season int32) bool {
	for i := range series.Seasons {
		if series.Seasons[i].GetSeasonNumber() == season {
			return true
		}
	}
	return false
}

func seasonComplete(series *sonarr.SeriesResource, season int32) bool {
	for i := range series.Seasons {
		s := &series.Seasons[i]
		if s.GetSeasonNumber() != season || s.Statistics == nil {
			continue
		}
		files := s.Statistics.GetEpisodeFileCount()
		total := s.Statistics.GetTotalEpisodeCount()
		if total == 0 {
			total = s.Statistics.GetEpisodeCount()
		}
		return total > 0 && files >= total
	}
	return false
}

func mergeContext(parent context.Context, auth context.Context) context.Context {
	if keys, ok := auth.Value(sonarr.ContextAPIKeys).(map[string]sonarr.APIKey); ok {
		return context.WithValue(parent, sonarr.ContextAPIKeys, keys)
	}
	return parent
}
