package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type apiKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Key        string `json:"key,omitempty"`
	MaskedKey  string `json:"maskedKey,omitempty"`
	KeyPrefix  string `json:"keyPrefix,omitempty"`
	Status     string `json:"status,omitempty"`
	CreatedAt  string `json:"createdAt,omitempty"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type createAPIKeyRequest struct {
	Session string `json:"session"`
	Name    string `json:"name"`
}

type createAPIKeyResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Session string `json:"session,omitempty"`
	APIKey  apiKey `json:"api_key,omitempty"`
}

type listAPIKeysResponse struct {
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
	Session string   `json:"session,omitempty"`
	APIKeys []apiKey `json:"api_keys"`
}

func parseAPIKeyList(raw json.RawMessage) []apiKey {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var keys []apiKey
	if errUnmarshal := json.Unmarshal(raw, &keys); errUnmarshal == nil {
		return sanitizeAPIKeys(keys, false)
	}
	var wrap struct {
		List  []apiKey `json:"list"`
		Items []apiKey `json:"items"`
		Keys  []apiKey `json:"keys"`
	}
	if errUnmarshal := json.Unmarshal(raw, &wrap); errUnmarshal != nil {
		return nil
	}
	switch {
	case len(wrap.List) > 0:
		keys = wrap.List
	case len(wrap.Items) > 0:
		keys = wrap.Items
	default:
		keys = wrap.Keys
	}
	return sanitizeAPIKeys(keys, false)
}

func sanitizeAPIKey(key apiKey, keepSecret bool) apiKey {
	if strings.TrimSpace(key.MaskedKey) == "" {
		if key.KeyPrefix != "" {
			key.MaskedKey = key.KeyPrefix
		} else if key.Key != "" {
			key.MaskedKey = maskAPIKey(key.Key)
		}
	}
	if !keepSecret {
		key.Key = ""
	}
	return key
}

func sanitizeAPIKeys(keys []apiKey, keepSecret bool) []apiKey {
	out := make([]apiKey, 0, len(keys))
	for _, key := range keys {
		if strings.EqualFold(key.Status, "deleted") {
			continue
		}
		out = append(out, sanitizeAPIKey(key, keepSecret))
	}
	return out
}

func maskAPIKey(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	if len(secret) <= 10 {
		return secret[:2] + "***"
	}
	return secret[:6] + "..." + secret[len(secret)-4:]
}

func listAPIKeys(cfg pluginConfig, session string) ([]apiKey, error) {
	raw, errGet := doTrGet(cfg, session, "/api/api-keys")
	if errGet != nil {
		return nil, errGet
	}
	return parseAPIKeyList(raw), nil
}

func createAPIKey(cfg pluginConfig, sess resolvedSession, name string) (apiKey, error) {
	existing, errList := listAPIKeys(cfg, sess.TRSession)
	active := 0
	if errList == nil {
		for _, key := range existing {
			if strings.EqualFold(key.Status, "enabled") || strings.EqualFold(key.Status, "active") || key.Status == "" {
				active++
			}
		}
		if active >= maxAPIKeysPerAccount {
			return apiKey{}, fmt.Errorf("session %s already has %d API keys (limit %d)", sess.ID, active, maxAPIKeysPerAccount)
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("CPA Key %d", active+1)
	}
	body, errMarshal := json.Marshal(map[string]string{"name": name})
	if errMarshal != nil {
		return apiKey{}, errMarshal
	}
	raw, errPost := doTrRequest(cfg, sess.TRSession, http.MethodPost, "/api/api-keys", body)
	if errPost != nil {
		return apiKey{}, errPost
	}
	var created apiKey
	if errUnmarshal := json.Unmarshal(raw, &created); errUnmarshal != nil {
		return apiKey{}, fmt.Errorf("decode created API key: %w", errUnmarshal)
	}
	if created.Name == "" {
		created.Name = name
	}
	if created.CreatedAt == "" {
		created.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if created.Status == "" {
		created.Status = "enabled"
	}
	return sanitizeAPIKey(created, true), nil
}
