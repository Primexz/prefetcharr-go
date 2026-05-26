<div align="center">

# prefetcharr-go

<a href="https://github.com/Primexz/prefetcharr-go">
    <img src="./assets/gopher.png" width="350" />
</a>


prefetcharr-go watches active Jellyfin TV sessions and asks Sonarr to search upcoming seasons at the right moment, so the next season is ready before the viewer gets there.

[![Release](https://github.com/Primexz/prefetcharr-go/actions/workflows/release.yml/badge.svg)](https://github.com/Primexz/prefetcharr-go/actions/workflows/release.yml)
[![golangci-lint](https://github.com/Primexz/prefetcharr-go/actions/workflows/golangci.yml/badge.svg)](https://github.com/Primexz/prefetcharr-go/actions/workflows/golangci.yml)

</div>


## Behavior

On each poll, prefetcharr-go reads active Jellyfin sessions, ignores anything that is not an episode, resolves the parent series TVDB ID, finds that series in Sonarr, and searches configured future seasons.

With `seasons_ahead: 1`, `min_season_progress_percent: 0`, and `include_current_season: false`, watching `S01E01` searches season 2. Set `seasons_ahead: 2` to search seasons 2 and 3.

Set `min_season_progress_percent` to wait until later in the current season before prefetching. For example, with a 10 episode season, `min_season_progress_percent: 30` means episodes 1 and 2 do not trigger prefetching, while episode 3 and later do.

Already searched seasons are deduplicated for 7 days to avoid submitting the same Sonarr search every poll.

## ⚙️ Configuration

Copy `config.example.yaml` and fill in the API keys:

```yaml
interval: 300s
log_level: debug

prefetch:
  seasons_ahead: 1
  min_season_progress_percent: 0
  include_current_season: false
  search_complete_seasons: false

notifications:
  enabled: false
  urls: []
  events:
    - search_submitted

jellyfin:
  url: http://jellyfin:8096
  api_key: your-jellyfin-api-key

sonarr:
  url: http://sonarr:8989
  api_key: your-sonarr-api-key

allowed_users: []
```

Notifications use [shoutrrr](https://github.com/nicholas-fedor/shoutrrr) URLs, so the same config can target Discord, Gotify, ntfy, Slack, SMTP, webhooks, and other supported services. Notification delivery is best-effort: failures are logged with notification URLs redacted and do not stop polling or searching.

```yaml
notifications:
  enabled: true
  urls:
    - ntfy://ntfy.sh/prefetcharr-alerts
  events:
    - search_submitted
```

Supported events are:

- `search_submitted`: a Sonarr season search command was submitted.

## 🐳 Docker Compose

Create `./prefetcharr-go/config.yaml` from `config.example.yaml`, then run prefetcharr-go with Docker Compose:

```yaml
services:
  prefetcharr:
    image: ghcr.io/primexz/prefetcharr-go:latest
    container_name: prefetcharr-go
    networks:
      - arr
    volumes:
      - ./prefetcharr:/config:ro
    restart: always
```

The container expects the configuration file at `/config/config.yaml`, so with the volume above your local config must be placed at:

```text
./prefetcharr/config.yaml
```

Start the container with:

```sh
docker compose up -d
```

## 💻 Run

```sh
go run ./cmd/prefetcharr -config config.yaml
```

## 🛠 Build

```sh
go build -o prefetcharr-go ./cmd/prefetcharr
```
