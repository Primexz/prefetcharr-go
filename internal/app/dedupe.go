package app

import "time"

type seasonKey struct {
	SeriesID int32
	Season   int32
}

type dedupe struct {
	ttl  time.Duration
	seen map[seasonKey]time.Time
}

func newDedupe(ttl time.Duration) *dedupe {
	return &dedupe{
		ttl:  ttl,
		seen: make(map[seasonKey]time.Time),
	}
}

func (d *dedupe) Seen(key seasonKey, now time.Time) bool {
	for k, t := range d.seen {
		if now.Sub(t) > d.ttl {
			delete(d.seen, k)
		}
	}
	if t, ok := d.seen[key]; ok && now.Sub(t) <= d.ttl {
		return true
	}
	return false
}

func (d *dedupe) Mark(key seasonKey, now time.Time) {
	d.seen[key] = now
}
