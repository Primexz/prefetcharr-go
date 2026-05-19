<div align="center">

# prefetcharr-go

<a href="https://github.com/Primexz/lndnotify">
    <img src="./assets/gopher.png" width="350" />
</a>

Go rewrite of the original Rust [`prefetcharr`](https://github.com/p-hueber/prefetcharr), focused on one workflow: when someone watches a Jellyfin series, ask Sonarr to search upcoming seasons.

</div>


## Behavior

On each poll, prefetcharr reads active Jellyfin sessions, ignores anything that is not an episode, resolves the parent series TVDB ID, finds that series in Sonarr, and searches configured future seasons.

With `seasons_ahead: 1` and `include_current_season: false`, watching `S01E01` searches season 2. Set `seasons_ahead: 2` to search seasons 2 and 3.

Already searched seasons are deduplicated for `dedupe_ttl` to avoid submitting the same Sonarr search every poll.

The original implementation supported multiple media servers and episode-window prefetching. This rewrite intentionally supports only Jellyfin and Sonarr, and it prefetches by season instead of by upcoming episode count. Radarr is not implemented because Radarr has no season model.

## Configuration

Copy `config.example.yaml` and fill in the API keys:

```yaml
interval: 300s
log_level: debug

prefetch:
  seasons_ahead: 1
  include_current_season: false
  search_complete_seasons: false
  dedupe_ttl: 168h

jellyfin:
  url: http://jellyfin:8096
  api_key: your-jellyfin-api-key

sonarr:
  url: http://sonarr:8989
  api_key: your-sonarr-api-key

allowed_users: []
```

## Run

```sh
go run ./cmd/prefetcharr -config config.yaml
```

## Build

```sh
go build -o prefetcharr ./cmd/prefetcharr
```
