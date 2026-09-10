package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
	"gopkg.in/yaml.v3"
)

const (
	pluginID              = "tokenrhythm-balance"
	pluginName            = "Token Rhythm Balance"
	pluginVer             = "0.2.1"
	defaultBase           = "https://tokenrhythm.studio"
	defaultPollConcurrent = 3
	maxPollConcurrent     = 16
	maxAPIKeysPerAccount  = 10
)

type sessionConfig struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	TRSession string `yaml:"tr_session"`
}

type pluginConfig struct {
	Enabled                bool            `yaml:"enabled"`
	Priority               int             `yaml:"priority"`
	TRSession              string          `yaml:"tr_session"`
	Sessions               []sessionConfig `yaml:"sessions"`
	BaseURL                string          `yaml:"base_url"`
	RefreshIntervalSeconds int             `yaml:"refresh_interval_seconds"`
	LowBalanceThreshold    float64         `yaml:"low_balance_threshold"`
	PollConcurrency        int             `yaml:"poll_concurrency"`
}

type resolvedSession struct {
	ID        string
	Name      string
	TRSession string
}

func defaultConfig() pluginConfig {
	return pluginConfig{
		BaseURL:                defaultBase,
		RefreshIntervalSeconds: 60,
		LowBalanceThreshold:    10,
		PollConcurrency:        defaultPollConcurrent,
	}
}

var (
	configMu      sync.RWMutex
	currentConfig = defaultConfig()
	configEpoch   uint64
)

func (c pluginConfig) normalized() pluginConfig {
	out := c
	out.TRSession = strings.TrimSpace(out.TRSession)
	out.BaseURL = strings.TrimRight(strings.TrimSpace(out.BaseURL), "/")
	if out.BaseURL == "" {
		out.BaseURL = defaultBase
	}
	if out.RefreshIntervalSeconds < 5 {
		out.RefreshIntervalSeconds = 5
	}
	if out.RefreshIntervalSeconds > 86400 {
		out.RefreshIntervalSeconds = 86400
	}
	if out.LowBalanceThreshold < 0 {
		out.LowBalanceThreshold = 0
	}
	if out.PollConcurrency < 1 {
		out.PollConcurrency = defaultPollConcurrent
	}
	if out.PollConcurrency > maxPollConcurrent {
		out.PollConcurrency = maxPollConcurrent
	}
	sessions := make([]sessionConfig, 0, len(out.Sessions))
	for _, sess := range out.Sessions {
		sess.ID = strings.TrimSpace(sess.ID)
		sess.Name = strings.TrimSpace(sess.Name)
		sess.TRSession = strings.TrimSpace(sess.TRSession)
		if sess.TRSession == "" {
			continue
		}
		sessions = append(sessions, sess)
	}
	out.Sessions = sessions
	return out
}

func (c pluginConfig) resolvedSessions() []resolvedSession {
	out := make([]resolvedSession, 0, len(c.Sessions)+1)
	used := map[string]int{}
	add := func(id, name, cookie string) {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			return
		}
		id = strings.TrimSpace(id)
		name = strings.TrimSpace(name)
		if id == "" {
			id = sessionSlug(name)
		}
		if id == "" {
			id = fmt.Sprintf("session-%d", len(out)+1)
		}
		base := id
		for i := 1; ; i++ {
			candidate := base
			if i > 1 {
				candidate = fmt.Sprintf("%s-%d", base, i)
			}
			if used[candidate] == 0 {
				id = candidate
				used[id] = 1
				break
			}
		}
		if name == "" {
			name = id
		}
		out = append(out, resolvedSession{ID: id, Name: name, TRSession: cookie})
	}
	for _, sess := range c.Sessions {
		add(sess.ID, sess.Name, sess.TRSession)
	}
	if len(out) == 0 {
		add("default", "default", c.TRSession)
	}
	return out
}

func sessionSlug(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '_' || r == '-' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (c pluginConfig) sessionByID(id string) (resolvedSession, bool) {
	id = strings.TrimSpace(id)
	sessions := c.resolvedSessions()
	if id == "" {
		if len(sessions) == 1 {
			return sessions[0], true
		}
		return resolvedSession{}, false
	}
	for _, sess := range sessions {
		if sess.ID == id {
			return sess, true
		}
	}
	return resolvedSession{}, false
}

func snapshotConfig() (pluginConfig, uint64) {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig, configEpoch
}

func applyConfig(raw []byte) pluginConfig {
	cfg := defaultConfig()
	if len(raw) > 0 {
		if errUnmarshal := yaml.Unmarshal(raw, &cfg); errUnmarshal != nil {
			// Keep defaults; the management handler reports configuration problems.
			cfg = defaultConfig()
		}
	}
	cfg = cfg.normalized()
	configMu.Lock()
	currentConfig = cfg
	configEpoch++
	configMu.Unlock()
	return cfg
}

type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	ManagementAPI bool `json:"management_api"`
}

func registerPlugin(request []byte) registration {
	var lifecycle lifecycleRequest
	if len(request) > 0 {
		_ = json.Unmarshal(request, &lifecycle)
	}
	applyConfig(lifecycle.ConfigYAML)
	return registration{
		SchemaVersion: 6,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVer,
			Author:           "cpa-tokenrhythm",
			GitHubRepository: "https://github.com/Charlie237/cpa-tokenrhythm",
			ConfigFields: []pluginapi.ConfigField{
				{
					Name:        "tr_session",
					Type:        pluginapi.ConfigFieldTypeString,
					Description: "Single Token Rhythm session cookie (tr_session=... or the raw sess_... value). Ignored when sessions is set.",
				},
				{
					Name:        "sessions",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Multiple Token Rhythm sessions. Each item: {id, name, tr_session}. Creating API keys requires a full cookie that includes tr_csrf.",
				},
				{
					Name:        "base_url",
					Type:        pluginapi.ConfigFieldTypeString,
					Description: "Token Rhythm origin. Defaults to https://tokenrhythm.studio.",
				},
				{
					Name:        "refresh_interval_seconds",
					Type:        pluginapi.ConfigFieldTypeInteger,
					Description: "Dashboard auto-refresh and server cache interval in seconds (5-86400). Defaults to 60.",
				},
				{
					Name:        "poll_concurrency",
					Type:        pluginapi.ConfigFieldTypeInteger,
					Description: "How many sessions to poll in parallel (1-16). Defaults to 3.",
				},
				{
					Name:        "low_balance_threshold",
					Type:        pluginapi.ConfigFieldTypeNumber,
					Description: "Available balance below this value is flagged as low. Defaults to 10.",
				},
			},
		},
		Capabilities: registrationCapability{ManagementAPI: true},
	}
}

func configError() string {
	cfg, _ := snapshotConfig()
	if len(cfg.resolvedSessions()) == 0 {
		return "no Token Rhythm session is configured (set tr_session or sessions)"
	}
	return ""
}

func fmtFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
