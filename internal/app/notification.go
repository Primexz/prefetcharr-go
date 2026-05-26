package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nicholas-fedor/shoutrrr"
	"github.com/nicholas-fedor/shoutrrr/pkg/types"
	"go.uber.org/zap"
)

const (
	notificationEventSearchSubmitted = "search_submitted"
)

type notificationSender interface {
	Send(message string, params *types.Params) []error
}

type notifier struct {
	enabled bool
	events  map[string]struct{}
	sender  notificationSender
	log     *zap.Logger
	urls    []string
}

type notificationPayload struct {
	SeriesTitle   string
	Season        int32
	User          string
	TriggerReason string
}

func newNotifier(cfg NotificationsConfig, log *zap.Logger) (*notifier, error) {
	n := &notifier{
		enabled: cfg.Enabled,
		events:  notificationEventSet(cfg.Events),
		log:     log,
		urls:    trimmedNotificationURLs(cfg.URLs),
	}
	if !cfg.Enabled {
		return n, nil
	}

	sender, err := shoutrrr.CreateSender(n.urls...)
	if err != nil {
		return nil, fmt.Errorf("initialize notifications: %w", errors.New(redactNotificationText(err.Error(), n.urls)))
	}
	n.sender = sender
	return n, nil
}

func (n *notifier) Notify(event string, payload notificationPayload) {
	if n == nil || !n.enabled || n.sender == nil || !n.eventEnabled(event) {
		return
	}

	message := notificationMessage(event, payload)
	params := types.Params{}
	params.SetTitle(notificationTitle(event))
	params.SetMessage(message)

	for i, err := range n.sender.Send(message, &params) {
		if err == nil {
			continue
		}
		n.log.Warn("notification delivery failed",
			zap.String("event", event),
			zap.Int("delivery", i+1),
			zap.String("error", redactNotificationText(err.Error(), n.urls)),
		)
	}
}

func (n *notifier) eventEnabled(event string) bool {
	if len(n.events) == 0 {
		return false
	}
	_, ok := n.events[event]
	return ok
}

func defaultNotificationEvents() []string {
	return []string{
		notificationEventSearchSubmitted,
	}
}

func validNotificationEvent(event string) bool {
	switch strings.TrimSpace(event) {
	case notificationEventSearchSubmitted:
		return true
	default:
		return false
	}
}

func notificationEventSet(events []string) map[string]struct{} {
	set := make(map[string]struct{}, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		set[event] = struct{}{}
	}
	return set
}

func trimmedNotificationURLs(rawURLs []string) []string {
	urls := make([]string, 0, len(rawURLs))
	for _, rawURL := range rawURLs {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL != "" {
			urls = append(urls, rawURL)
		}
	}
	return urls
}

func notificationTitle(event string) string {
	switch event {
	case notificationEventSearchSubmitted:
		return "prefetcharr-go search submitted"
	default:
		return "prefetcharr-go notification"
	}
}

func notificationMessage(event string, payload notificationPayload) string {
	var b strings.Builder
	b.WriteString(notificationTitle(event))
	writeNotificationLine(&b, "Series", payload.SeriesTitle)
	if payload.Season > 0 {
		writeNotificationLine(&b, "Season", fmt.Sprintf("%d", payload.Season))
	}
	writeNotificationLine(&b, "User", payload.User)
	writeNotificationLine(&b, "Trigger", payload.TriggerReason)
	return b.String()
}

func writeNotificationLine(b *strings.Builder, label string, value string) {
	if value == "" {
		return
	}
	b.WriteString("\n")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(value)
}

func redactNotificationText(text string, urls []string) string {
	for _, rawURL := range urls {
		if rawURL == "" {
			continue
		}
		text = strings.ReplaceAll(text, rawURL, "<redacted notification url>")
	}
	return text
}
