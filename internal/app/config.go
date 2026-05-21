package app

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interval     Duration       `yaml:"interval"`
	LogLevel     string         `yaml:"log_level"`
	Prefetch     PrefetchConfig `yaml:"prefetch"`
	Jellyfin     ServerConfig   `yaml:"jellyfin"`
	Sonarr       ServerConfig   `yaml:"sonarr"`
	AllowedUsers []string       `yaml:"allowed_users"`
}

type PrefetchConfig struct {
	SeasonsAhead          int      `yaml:"seasons_ahead"`
	MinSeasonProgress     int      `yaml:"min_season_progress_percent"`
	IncludeCurrentSeason  bool     `yaml:"include_current_season"`
	SearchCompleteSeasons bool     `yaml:"search_complete_seasons"`
	DedupeTTL             Duration `yaml:"dedupe_ttl"`
}

type ServerConfig struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"api_key"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err == nil {
		parsed, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		d.Duration = parsed
		return nil
	}

	var seconds int
	if err := value.Decode(&seconds); err != nil {
		return err
	}
	d.Duration = time.Duration(seconds) * time.Second
	return nil
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Interval: Duration{Duration: 5 * time.Minute},
		LogLevel: "info",
		Prefetch: PrefetchConfig{
			SeasonsAhead: 1,
			DedupeTTL:    Duration{Duration: 7 * 24 * time.Hour},
		},
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	var missing []string
	if strings.TrimSpace(c.Jellyfin.URL) == "" {
		missing = append(missing, "jellyfin.url")
	}
	if strings.TrimSpace(c.Jellyfin.APIKey) == "" {
		missing = append(missing, "jellyfin.api_key")
	}
	if strings.TrimSpace(c.Sonarr.URL) == "" {
		missing = append(missing, "sonarr.url")
	}
	if strings.TrimSpace(c.Sonarr.APIKey) == "" {
		missing = append(missing, "sonarr.api_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	if c.Interval.Duration <= 0 {
		return errors.New("interval must be positive")
	}
	if c.Prefetch.SeasonsAhead < 1 {
		return errors.New("prefetch.seasons_ahead must be at least 1")
	}
	if c.Prefetch.MinSeasonProgress < 0 || c.Prefetch.MinSeasonProgress > 100 {
		return errors.New("prefetch.min_season_progress_percent must be between 0 and 100")
	}
	if c.Prefetch.DedupeTTL.Duration <= 0 {
		return errors.New("prefetch.dedupe_ttl must be positive")
	}
	return nil
}
