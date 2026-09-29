package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/sentinel-monitoring/sentinel/internal/logpath"
	"github.com/sentinel-monitoring/sentinel/internal/models"
)

type logFollower struct {
	mu      sync.Mutex
	cfg     *agentConfig
	client  *http.Client
	sources map[string]*fileTailState
	stopCh  chan struct{}
	wg      sync.WaitGroup

	maxEventBytes int
	maxPerSec     int
	maxBatch      int
	leases        map[string]time.Time
}

type fileTailState struct {
	source   models.AgentLogSource
	path     string
	file     *os.File
	inode    uint64
	offset   int64
	reader   *bufio.Reader
	dropped  int64
	health   string
	detail   string
	lastSend time.Time
	tokens   float64
	lastTok  time.Time
	pending  []models.LogIngestEvent
	volumes  map[string]int64 // level -> count this minute
	volStart time.Time
	seenFP   map[string]time.Time
}

func newLogFollower(cfg *agentConfig, client *http.Client) *logFollower {
	return &logFollower{
		cfg:           cfg,
		client:        client,
		sources:       map[string]*fileTailState{},
		stopCh:        make(chan struct{}),
		maxEventBytes: 32 * 1024,
		maxPerSec:     100,
		maxBatch:      100,
		leases:        map[string]time.Time{},
	}
}

func (f *logFollower) Stop() {
	close(f.stopCh)
	f.wg.Wait()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, st := range f.sources {
		if st.file != nil {
			st.file.Close()
		}
	}
}

func (f *logFollower) UpdateConfig(cfg models.HostAgentConfig) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if cfg.MaxEventSizeBytes > 0 {
		f.maxEventBytes = cfg.MaxEventSizeBytes
	}
	if cfg.MaxEventsPerSecHost > 0 {
		f.maxPerSec = cfg.MaxEventsPerSecHost
	}
	if cfg.MaxBatchEvents > 0 {
		f.maxBatch = cfg.MaxBatchEvents
	}
	now := time.Now().UTC()
	f.leases = map[string]time.Time{}
	for _, l := range cfg.TailLeases {
		if t, err := time.Parse(time.RFC3339, l.Until); err == nil && t.After(now) {
			f.leases[l.SourceID] = t
		}
	}
	wanted := map[string]models.AgentLogSource{}
	for _, src := range cfg.LogSources {
		if !src.Enabled {
			continue
		}
		if src.Type == "" {
			src.Type = string(models.LogSourceFile)
		}
		wanted[src.ID] = src
	}
	for id, st := range f.sources {
		if _, ok := wanted[id]; !ok {
			if st.file != nil {
				st.file.Close()
			}
			delete(f.sources, id)
		}
	}
	for id, src := range wanted {
		if src.Type == string(models.LogSourceJournal) {
			// journald follower handled in tickJournal
			if _, ok := f.sources[id]; !ok {
				f.sources[id] = &fileTailState{
					source:  src,
					health:  string(models.LogHealthOK),
					seenFP:  map[string]time.Time{},
					volumes: map[string]int64{},
					volStart: time.Now().UTC().Truncate(time.Minute),
					lastTok: time.Now(),
					tokens:  float64(f.maxPerSec),
				}
			} else {
				f.sources[id].source = src
			}
			continue
		}
		st, ok := f.sources[id]
		if !ok {
			st = &fileTailState{
				source:   src,
				health:   string(models.LogHealthOK),
				seenFP:   map[string]time.Time{},
				volumes:  map[string]int64{},
				volStart: time.Now().UTC().Truncate(time.Minute),
				lastTok:  time.Now(),
				tokens:   float64(f.maxPerSec),
			}
			f.sources[id] = st
			f.openSource(st)
		} else {
			if st.source.Path != src.Path {
				if st.file != nil {
					st.file.Close()
					st.file = nil
				}
				st.source = src
				f.openSource(st)
			} else {
				st.source = src
			}
		}
	}
}

func (f *logFollower) Start() {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		flush := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		defer flush.Stop()
		for {
			select {
			case <-f.stopCh:
				f.flushAll()
				return
			case <-ticker.C:
				f.tick()
			case <-flush.C:
				f.flushAll()
			}
		}
	}()
}

func (f *logFollower) tick() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, st := range f.sources {
		if st.source.Type == string(models.LogSourceJournal) {
			f.tickJournal(st)
			continue
		}
		f.tickFile(st)
	}
}

func (f *logFollower) openSource(st *fileTailState) {
	paths := []string{st.source.Path}
	if strings.ContainsAny(st.source.Path, "*?[") {
		expanded, err := logpath.ExpandGlob(st.source.Path)
		if err != nil {
			st.health = string(models.LogHealthRejectedPolicy)
			st.detail = err.Error()
			return
		}
		if len(expanded) == 0 {
			st.health = string(models.LogHealthFileMissing)
			st.detail = "no glob matches"
			return
		}
		// Prefer newest file
		paths = expanded
		newest := paths[0]
		var newestMod time.Time
		for _, p := range paths {
			if info, err := os.Stat(p); err == nil {
				if info.ModTime().After(newestMod) {
					newestMod = info.ModTime()
					newest = p
				}
			}
		}
		paths = []string{newest}
	}
	f.openPath(st, paths[0])
}

func (f *logFollower) openPath(st *fileTailState, path string) {
	file, resolved, err := logpath.OpenLogFile(path)
	if err != nil {
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "permission denied"):
			st.health = string(models.LogHealthPermissionDenied)
		case strings.Contains(errStr, "file missing") || os.IsNotExist(err):
			st.health = string(models.LogHealthFileMissing)
		case errorsIsRejected(err):
			st.health = string(models.LogHealthRejectedPolicy)
		default:
			st.health = string(models.LogHealthFileMissing)
		}
		st.detail = errStr
		return
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		st.health = string(models.LogHealthPermissionDenied)
		st.detail = err.Error()
		return
	}
	// Start from end (follow new lines only)
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		st.health = string(models.LogHealthPermissionDenied)
		st.detail = err.Error()
		return
	}
	st.file = file
	st.path = resolved
	st.offset = off
	st.inode = fileInode(info)
	st.reader = bufio.NewReaderSize(file, 64*1024)
	st.health = string(models.LogHealthOK)
	st.detail = ""
}

func errorsIsRejected(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "path not allowed") || strings.Contains(err.Error(), "not a regular file"))
}

func (f *logFollower) tickFile(st *fileTailState) {
	if st.file == nil {
		f.openSource(st)
		if st.file == nil {
			return
		}
	}
	// Detect rotation
	info, err := os.Stat(st.path)
	if err != nil {
		st.file.Close()
		st.file = nil
		st.health = string(models.LogHealthRotated)
		st.detail = "file disappeared (rotated?)"
		f.openSource(st)
		return
	}
	ino := fileInode(info)
	if ino != 0 && st.inode != 0 && ino != st.inode {
		st.file.Close()
		st.file = nil
		st.health = string(models.LogHealthRotated)
		st.detail = "inode changed"
		f.openSource(st)
		return
	}
	// Re-validate policy periodically on reopen path already does; also check size shrink
	if info.Size() < st.offset {
		st.file.Close()
		st.file = nil
		f.openSource(st)
		return
	}

	liveTail := f.leaseActive(st.source.ID)
	for {
		line, err := st.reader.ReadString('\n')
		if len(line) > 0 {
			st.offset += int64(len(line))
			f.handleLine(st, strings.TrimRight(line, "\r\n"), liveTail)
		}
		if err != nil {
			break
		}
	}
}

func (f *logFollower) leaseActive(sourceID string) bool {
	until, ok := f.leases[sourceID]
	return ok && until.After(time.Now().UTC())
}

func (f *logFollower) handleLine(st *fileTailState, line string, liveTail bool) {
	if line == "" || !utf8.ValidString(line) {
		return
	}
	maxBytes := f.maxEventBytes
	if st.source.MaxEventBytes > 0 {
		maxBytes = st.source.MaxEventBytes
	}
	if len(line) > maxBytes {
		line = line[:maxBytes]
	}
	line = redactSecrets(line)
	level := parseLogLevel(line)
	bumpVolume(st, level)

	minLevel := models.LogLevel(st.source.MinLevel)
	if minLevel == "" {
		minLevel = models.LogLevelWarn
	}
	ship := liveTail || levelAtLeast(level, minLevel)
	if ship && !liveTail && len(st.source.IncludePatterns) > 0 {
		ship = matchesAny(line, st.source.IncludePatterns)
	}
	if !ship {
		return
	}
	if !f.takeToken(st) {
		st.dropped++
		st.health = string(models.LogHealthRateLimited)
		st.detail = "rate limited"
		return
	}
	fp := fingerprintLine(line)
	now := time.Now().UTC()
	if t, ok := st.seenFP[fp]; ok && now.Sub(t) < 30*time.Second {
		return // dedupe
	}
	st.seenFP[fp] = now
	if len(st.seenFP) > 2000 {
		st.seenFP = map[string]time.Time{fp: now}
	}
	st.pending = append(st.pending, models.LogIngestEvent{
		SourceID:    st.source.ID,
		Timestamp:   now.Format(time.RFC3339),
		Level:       string(level),
		Message:     line,
		Fingerprint: fp,
	})
	if len(st.pending) >= f.maxBatch {
		f.flushState(st)
	}
	if st.health == string(models.LogHealthRateLimited) {
		st.health = string(models.LogHealthOK)
		st.detail = ""
	}
}

func (f *logFollower) takeToken(st *fileTailState) bool {
	max := f.maxPerSec
	if st.source.MaxEventsPerSec > 0 {
		max = st.source.MaxEventsPerSec
	}
	now := time.Now()
	elapsed := now.Sub(st.lastTok).Seconds()
	st.lastTok = now
	st.tokens += elapsed * float64(max)
	if st.tokens > float64(max) {
		st.tokens = float64(max)
	}
	if st.tokens < 1 {
		return false
	}
	st.tokens--
	return true
}

func bumpVolume(st *fileTailState, level models.LogLevel) {
	now := time.Now().UTC().Truncate(time.Minute)
	if st.volStart.IsZero() {
		st.volStart = now
	}
	if levelAtLeast(level, models.LogLevelWarn) {
		st.volumes[string(level)]++
	}
}

func (f *logFollower) flushAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, st := range f.sources {
		f.flushState(st)
	}
}

func (f *logFollower) flushState(st *fileTailState) {
	if len(st.pending) == 0 && len(st.volumes) == 0 && st.health == string(models.LogHealthOK) {
		return
	}
	batch := models.LogIngestBatch{
		Events: append([]models.LogIngestEvent(nil), st.pending...),
		Health: []models.LogIngestHealth{{
			SourceID: st.source.ID, Health: st.health, Detail: st.detail, DroppedLines: st.dropped,
		}},
	}
	for level, count := range st.volumes {
		if count <= 0 {
			continue
		}
		batch.Volumes = append(batch.Volumes, models.LogIngestVolume{
			SourceID: st.source.ID, BucketStart: st.volStart.Format(time.RFC3339),
			Level: level, Count: count,
		})
	}
	st.pending = nil
	st.volumes = map[string]int64{}
	st.volStart = time.Now().UTC().Truncate(time.Minute)
	go f.postBatch(batch)
}

func (f *logFollower) postBatch(batch models.LogIngestBatch) {
	if len(batch.Events) == 0 && len(batch.Volumes) == 0 && len(batch.Health) == 0 {
		return
	}
	body, err := json.Marshal(batch)
	if err != nil {
		return
	}
	url := strings.TrimRight(f.cfg.ServerURL, "/") + "/api/agent/logs"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.cfg.Token)
	req.Header.Set("User-Agent", agentUserAgent)
	resp, err := f.client.Do(req)
	if err != nil {
		log.Printf("log ingest: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		log.Printf("log ingest: status %d", resp.StatusCode)
	}
}

// tickJournal polls journalctl for allowed units (phase 2).
func (f *logFollower) tickJournal(st *fileTailState) {
	unit := st.source.Path
	if !models.JournalUnitAllowed(unit) {
		st.health = string(models.LogHealthRejectedPolicy)
		st.detail = "journal unit not allowed"
		return
	}
	// Lightweight: run journalctl -u unit -n 20 --no-pager -o short-iso since last
	// Implemented in journal_linux.go / stub elsewhere via runJournalFollow.
	lines, err := readJournalLines(unit, 30)
	if err != nil {
		st.health = string(models.LogHealthPermissionDenied)
		st.detail = err.Error()
		return
	}
	st.health = string(models.LogHealthOK)
	st.detail = ""
	liveTail := f.leaseActive(st.source.ID)
	for _, line := range lines {
		f.handleLine(st, line, liveTail)
	}
}

func parseLogLevel(line string) models.LogLevel {
	upper := strings.ToUpper(line)
	switch {
	case strings.Contains(upper, "CRITICAL") || strings.Contains(upper, " EMERG") || strings.Contains(upper, "FATAL"):
		return models.LogLevelCrit
	case strings.Contains(upper, "ERROR") || strings.Contains(upper, " ERR ") || strings.Contains(upper, "[ERR]"):
		return models.LogLevelError
	case strings.Contains(upper, "WARN") || strings.Contains(upper, "WARNING"):
		return models.LogLevelWarn
	case strings.Contains(upper, "DEBUG"):
		return models.LogLevelDebug
	default:
		return models.LogLevelInfo
	}
}

func levelRank(l models.LogLevel) int {
	switch l {
	case models.LogLevelDebug:
		return 10
	case models.LogLevelInfo:
		return 20
	case models.LogLevelWarn:
		return 30
	case models.LogLevelError:
		return 40
	case models.LogLevelCrit:
		return 50
	default:
		return 20
	}
}

func levelAtLeast(have, min models.LogLevel) bool {
	return levelRank(have) >= levelRank(min)
}

func matchesAny(line string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(line, p) {
			return true
		}
		if re, err := regexp.Compile("(?i)" + p); err == nil && re.MatchString(line) {
			return true
		}
	}
	return false
}

func fingerprintLine(msg string) string {
	norm := regexp.MustCompile(`\d+`).ReplaceAllString(msg, "#")
	if len(norm) > 512 {
		norm = norm[:512]
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}

var (
	reBearer = regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)[A-Za-z0-9._\-+=/]+`)
	reAWS    = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
	reJWT    = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
)

func redactSecrets(line string) string {
	line = reBearer.ReplaceAllString(line, "${1}[REDACTED]")
	line = reAWS.ReplaceAllString(line, "[REDACTED_AWS_KEY]")
	line = reJWT.ReplaceAllString(line, "[REDACTED_JWT]")
	return line
}

func fileInode(info os.FileInfo) uint64 {
	return inodeFromFileInfo(info)
}