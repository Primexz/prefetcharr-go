package app

import (
	"context"
	"slices"
	"strings"
	"time"

	sonarr "github.com/devopsarr/sonarr-go/sonarr"
	"go.uber.org/zap"
)

const dedupeTTL = 7 * 24 * time.Hour

type App struct {
	cfg      Config
	log      *zap.Logger
	jellyfin *JellyfinClient
	sonarr   *SonarrClient
	dedupe   *dedupe
}

func New(cfg Config, log *zap.Logger) (*App, error) {
	return &App{
		cfg:      cfg,
		log:      log,
		jellyfin: NewJellyfinClient(cfg.Jellyfin),
		sonarr:   NewSonarrClient(cfg.Sonarr),
		dedupe:   newDedupe(dedupeTTL),
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.log.Info("prefetcharr-go started", zap.Duration("interval", a.cfg.Interval.Duration))
	if err := a.tick(ctx); err != nil {
		a.log.Warn("poll failed", zap.Error(err))
	}

	ticker := time.NewTicker(a.cfg.Interval.Duration)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.tick(ctx); err != nil {
				a.log.Warn("poll failed", zap.Error(err))
			}
		}
	}
}

func (a *App) tick(ctx context.Context) error {
	items, err := a.jellyfin.NowPlaying(ctx)
	if err != nil {
		return err
	}
	for _, np := range items {
		if !a.userAllowed(np.UserName) {
			a.log.Debug("skip user", zap.String("user", np.UserName), zap.String("title", np.Title))
			continue
		}
		if err := a.prefetch(ctx, np); err != nil {
			a.log.Warn("prefetch failed",
				zap.String("title", np.Title),
				zap.Int32("tvdb_id", np.TVDBID),
				zap.Error(err),
			)
		}
	}
	return nil
}

func (a *App) prefetch(ctx context.Context, np NowPlaying) error {
	series, err := a.sonarr.SeriesByTVDB(ctx, np.TVDBID)
	if err != nil {
		return err
	}

	a.log.Info("now playing",
		zap.String("title", np.Title),
		zap.String("sonarr_title", series.GetTitle()),
		zap.Int32("tvdb_id", np.TVDBID),
		zap.Int32("season", np.Season),
		zap.Int32("episode", np.Episode),
		zap.String("user", np.UserName),
	)

	excludedTag, excluded, err := a.excludedSonarrTag(ctx, series)
	if err != nil {
		return err
	}
	if excluded {
		a.log.Info("skip excluded Sonarr tag",
			zap.String("title", series.GetTitle()),
			zap.String("tag", excludedTag),
		)
		return nil
	}

	if a.cfg.Prefetch.MinSeasonProgress > 0 {
		progress, ok := seasonProgressPercent(series, np.Season, np.Episode)
		if !ok {
			a.log.Debug("skip unknown season progress",
				zap.String("title", series.GetTitle()),
				zap.Int32("season", np.Season),
				zap.Int32("episode", np.Episode),
				zap.String("user", np.UserName),
				zap.Int("min_season_progress_percent", a.cfg.Prefetch.MinSeasonProgress),
			)
			return nil
		}
		if progress < float64(a.cfg.Prefetch.MinSeasonProgress) {
			a.log.Debug("skip season below minimum progress",
				zap.String("title", series.GetTitle()),
				zap.Int32("season", np.Season),
				zap.Int32("episode", np.Episode),
				zap.Float64("season_progress_percent", progress),
				zap.Int("min_season_progress_percent", a.cfg.Prefetch.MinSeasonProgress),
				zap.String("user", np.UserName),
			)
			return nil
		}
	}

	for _, season := range targetSeasons(np.Season, a.cfg.Prefetch) {
		if !seasonExists(series, season) {
			a.log.Debug("skip missing season in Sonarr",
				zap.String("title", series.GetTitle()),
				zap.Int32("season", season),
			)
			continue
		}
		if !a.cfg.Prefetch.SearchCompleteSeasons && seasonComplete(series, season) {
			a.log.Debug("skip complete season",
				zap.String("title", series.GetTitle()),
				zap.Int32("season", season),
			)
			continue
		}

		key := seasonKey{SeriesID: series.GetId(), Season: season}
		now := time.Now()
		if a.dedupe.Seen(key, now) {
			a.log.Debug("skip recently searched season",
				zap.String("title", series.GetTitle()),
				zap.Int32("season", season),
			)
			continue
		}

		if err := a.sonarr.MonitorSeason(ctx, series, season); err != nil {
			return err
		}
		a.log.Info("searching season", zap.String("title", series.GetTitle()), zap.Int32("season", season))
		if err := a.sonarr.SearchSeason(ctx, series.GetId(), season); err != nil {
			return err
		}
		a.dedupe.Mark(key, now)
	}
	return nil
}

func (a *App) userAllowed(user string) bool {
	return len(a.cfg.AllowedUsers) == 0 || slices.Contains(a.cfg.AllowedUsers, user)
}

func (a *App) excludedSonarrTag(ctx context.Context, series sonarrSeries) (string, bool, error) {
	if len(a.cfg.Prefetch.ExcludedSonarrTags) == 0 {
		return "", false, nil
	}

	tags, err := a.sonarr.Tags(ctx)
	if err != nil {
		return "", false, err
	}
	excluded := excludedSonarrTagIDs(tags, a.cfg.Prefetch.ExcludedSonarrTags)
	tag, ok := matchingExcludedSonarrTag(series.GetTags(), excluded)
	return tag, ok, nil
}

func targetSeasons(current int32, cfg PrefetchConfig) []int32 {
	start := current + 1
	if cfg.IncludeCurrentSeason {
		start = current
	}

	seasons := make([]int32, 0, cfg.SeasonsAhead)
	for i := 0; i < cfg.SeasonsAhead; i++ {
		seasons = append(seasons, start+int32(i))
	}
	return seasons
}

type sonarrSeries interface {
	GetTags() []int32
}

func excludedSonarrTagIDs(tags []sonarr.TagResource, names []string) map[int32]string {
	labels := make(map[string]string, len(names))
	for _, name := range names {
		labels[normalizeTag(name)] = strings.TrimSpace(name)
	}

	ids := make(map[int32]string, len(labels))
	for i := range tags {
		if label, ok := labels[normalizeTag(tags[i].GetLabel())]; ok {
			ids[tags[i].GetId()] = label
		}
	}
	return ids
}

func normalizeTag(label string) string {
	return strings.ToLower(strings.TrimSpace(label))
}

func matchingExcludedSonarrTag(seriesTags []int32, excluded map[int32]string) (string, bool) {
	for _, id := range seriesTags {
		if label, ok := excluded[id]; ok {
			return label, true
		}
	}
	return "", false
}
