package main

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

const extraSessionsFile = "tokenrhythm-sessions.json"

var (
	extraMu           sync.Mutex
	extraSessionsPath = extraSessionsFile
)

func loadExtraSessions() []sessionConfig {
	extraMu.Lock()
	defer extraMu.Unlock()
	data, errRead := os.ReadFile(extraSessionsPath)
	if errRead != nil || len(data) == 0 {
		return nil
	}
	var wrap struct {
		Sessions []sessionConfig `json:"sessions"`
	}
	if errUnmarshal := json.Unmarshal(data, &wrap); errUnmarshal != nil {
		return nil
	}
	out := make([]sessionConfig, 0, len(wrap.Sessions))
	for _, sess := range wrap.Sessions {
		sess.ID = strings.TrimSpace(sess.ID)
		sess.Name = strings.TrimSpace(sess.Name)
		sess.TRSession = strings.TrimSpace(sess.TRSession)
		if sess.TRSession == "" {
			continue
		}
		out = append(out, sess)
	}
	return out
}

func saveExtraSessions(sessions []sessionConfig) error {
	extraMu.Lock()
	defer extraMu.Unlock()
	raw, errMarshal := json.MarshalIndent(struct {
		Sessions []sessionConfig `json:"sessions"`
	}{Sessions: sessions}, "", "  ")
	if errMarshal != nil {
		return errMarshal
	}
	return os.WriteFile(extraSessionsPath, raw, 0o600)
}

func addExtraSession(id, name, cookie string) (sessionConfig, error) {
	cookie = strings.TrimSpace(cookie)
	name = strings.TrimSpace(name)
	id = strings.TrimSpace(id)
	if cookie == "" {
		return sessionConfig{}, errSessionRequired()
	}
	sessions := loadExtraSessions()
	if id == "" {
		id = sessionSlug(name)
	}
	if id == "" {
		id = "session"
	}
	used := map[string]struct{}{}
	for i, sess := range sessions {
		used[sess.ID] = struct{}{}
		if sess.TRSession == cookie {
			if name != "" {
				sessions[i].Name = name
			}
			return sessions[i], saveAndBump(sessions)
		}
	}
	base := id
	for i := 1; ; i++ {
		candidate := base
		if i > 1 {
			candidate = base + "-" + itoa(i)
		}
		if _, exists := used[candidate]; !exists {
			id = candidate
			break
		}
	}
	if name == "" {
		name = id
	}
	item := sessionConfig{ID: id, Name: name, TRSession: cookie}
	sessions = append(sessions, item)
	if errSave := saveAndBump(sessions); errSave != nil {
		return sessionConfig{}, errSave
	}
	return item, nil
}

func saveAndBump(sessions []sessionConfig) error {
	if errSave := saveExtraSessions(sessions); errSave != nil {
		return errSave
	}
	configMu.Lock()
	configEpoch++
	configMu.Unlock()
	return nil
}

func errSessionRequired() error {
	return errString("session is required")
}

type errString string

func (e errString) Error() string { return string(e) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
