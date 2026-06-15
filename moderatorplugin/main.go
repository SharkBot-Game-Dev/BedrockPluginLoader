package main

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server"
	"github.com/df-mc/dragonfly/server/player"
)

const (
	maxURLsPerMessage = 1
	urlWindow         = 2 * time.Minute
	maxURLMessages    = 2
	repeatWindow      = time.Minute
)

var (
	urlPattern = regexp.MustCompile(`(?i)\b((?:https?://|www\.)[^\s]+|(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+(?:com|net|org|io|gg|jp|me|xyz|top|site|online|link|ru|cn|biz|info|dev|app|club|shop|co)(?:/[^\s]*)?)`)
	urlSpam    = newURLTracker()
)

func Init(srv *server.Server) {
	log.Printf("[moderatorplugin] URL spam protection enabled")
}

func OnPlayerJoin(p *player.Player) {
	p.Message("moderatorplugin is enabled.")
}

func PlayerHandler(p *player.Player) player.Handler {
	return &moderatorHandler{p: p}
}

type moderatorHandler struct {
	player.NopHandler
	p *player.Player
}

func (h *moderatorHandler) HandleChat(ctx *player.Context, message *string) {
	urls := urlPattern.FindAllString(*message, -1)
	if len(urls) == 0 {
		return
	}

	name := h.p.Name()
	normalizedURLs := normalizeURLs(urls)
	blockReason := urlSpam.record(name, normalizedURLs, time.Now())
	if len(urls) > maxURLsPerMessage {
		blockReason = "Please send only one URL at a time."
	}

	if blockReason == "" {
		return
	}

	ctx.Cancel()
	h.p.Message(blockReason)
	log.Printf("[moderatorplugin] blocked URL spam from %s: %q", name, *message)
}

func normalizeURLs(urls []string) []string {
	normalized := make([]string, 0, len(urls))
	for _, raw := range urls {
		url := strings.ToLower(strings.TrimSpace(raw))
		url = strings.TrimRight(url, ".,;:!?)\"]}'")
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "www.")
		if url != "" {
			normalized = append(normalized, url)
		}
	}
	return normalized
}

type urlTracker struct {
	mu      sync.Mutex
	players map[string]*playerURLState
}

type playerURLState struct {
	messages []time.Time
	lastSeen map[string]time.Time
}

func newURLTracker() *urlTracker {
	return &urlTracker{players: make(map[string]*playerURLState)}
}

func (t *urlTracker) record(playerName string, urls []string, now time.Time) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	state := t.players[playerName]
	if state == nil {
		state = &playerURLState{lastSeen: make(map[string]time.Time)}
		t.players[playerName] = state
	}

	state.messages = pruneTimes(state.messages, now.Add(-urlWindow))
	if len(state.messages) >= maxURLMessages {
		state.messages = append(state.messages, now)
		return "Please slow down. Repeated URL messages are blocked."
	}

	for _, url := range urls {
		if last, ok := state.lastSeen[url]; ok && now.Sub(last) < repeatWindow {
			state.messages = append(state.messages, now)
			state.lastSeen[url] = now
			return "Please do not repeat the same URL."
		}
		state.lastSeen[url] = now
	}

	state.messages = append(state.messages, now)
	t.pruneLastSeen(state, now.Add(-repeatWindow))
	return ""
}

func (t *urlTracker) pruneLastSeen(state *playerURLState, cutoff time.Time) {
	for url, seen := range state.lastSeen {
		if seen.Before(cutoff) {
			delete(state.lastSeen, url)
		}
	}
}

func pruneTimes(times []time.Time, cutoff time.Time) []time.Time {
	kept := times[:0]
	for _, ts := range times {
		if ts.After(cutoff) || ts.Equal(cutoff) {
			kept = append(kept, ts)
		}
	}
	return kept
}

func main() {}
