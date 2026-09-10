package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type managementRequest struct {
	Method  string              `json:"Method"`
	Path    string              `json:"Path"`
	Headers map[string][]string `json:"Headers"`
	Query   map[string][]string `json:"Query"`
	Body    []byte              `json:"Body"`
}

type managementResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers"`
	Body       []byte              `json:"Body"`
}

type managementRegistration struct {
	Routes    []managementRoute `json:"routes,omitempty"`
	Resources []resourceRoute   `json:"resources,omitempty"`
}

type managementRoute struct {
	Method      string `json:"Method"`
	Path        string `json:"Path"`
	Menu        string `json:"Menu,omitempty"`
	Description string `json:"Description,omitempty"`
}

type resourceRoute struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

func managementRegistrationResponse() managementRegistration {
	return managementRegistration{
		Routes: []managementRoute{
			{
				Method:      http.MethodGet,
				Path:        "/tokenrhythm/balance",
				Description: "Returns Token Rhythm wallet, usage, and API keys for all configured sessions.",
			},
			{
				Method:      http.MethodGet,
				Path:        "/tokenrhythm/sessions",
				Description: "Lists configured Token Rhythm sessions and their last polled status.",
			},
			{
				Method:      http.MethodGet,
				Path:        "/tokenrhythm/api-keys",
				Description: "Lists API keys for a Token Rhythm session.",
			},
			{
				Method:      http.MethodPost,
				Path:        "/tokenrhythm/api-keys",
				Description: "Creates an API key on a Token Rhythm session. Requires tr_csrf in the session cookie.",
			},
		},
		Resources: []resourceRoute{{
			Path:        "/balance",
			Menu:        "Token Rhythm 账户",
			Description: "Token Rhythm multi-session balance, polling, and API key dashboard.",
		}},
	}
}

func handleManagement(request []byte) (raw []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			raw, err = okEnvelope(jsonError(http.StatusOK, fmt.Sprintf("plugin panic: %v", recovered)))
		}
	}()
	var req managementRequest
	if len(request) > 0 {
		if errUnmarshal := json.Unmarshal(request, &req); errUnmarshal != nil {
			return errorEnvelope("invalid_request", "invalid management request: "+errUnmarshal.Error()), nil
		}
	}
	path := strings.TrimRight(req.Path, "/")
	switch {
	case strings.Contains(req.Path, "/v0/resource/"):
		return okEnvelope(resourceResponse(dashboardHTML()))
	case strings.HasSuffix(path, "/api-keys"):
		return okEnvelope(apiKeysResponse(req))
	case strings.HasSuffix(path, "/sessions"):
		return okEnvelope(sessionsResponse(req))
	case strings.HasSuffix(path, "/balance"):
		return okEnvelope(balanceResponse(req))
	default:
		return okEnvelope(managementResponse{
			StatusCode: http.StatusNotFound,
			Headers:    jsonHeaders(),
			Body:       []byte(`{"ok":false,"error":"not found"}`),
		})
	}
}

func queryValue(query map[string][]string, key string) string {
	for _, value := range query[key] {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func queryFlag(query map[string][]string, key string) bool {
	for _, value := range query[key] {
		trimmed := strings.TrimSpace(value)
		if trimmed == "1" || strings.EqualFold(trimmed, "true") {
			return true
		}
	}
	return false
}

func balanceResponse(req managementRequest) managementResponse {
	payload, _ := balanceJSON(queryFlag(req.Query, "refresh"), queryValue(req.Query, "session"))
	if len(payload) == 0 {
		return jsonError(http.StatusOK, "empty balance response")
	}
	return managementResponse{
		StatusCode: http.StatusOK,
		Headers:    jsonHeaders(),
		Body:       payload,
	}
}

func sessionsResponse(req managementRequest) managementResponse {
	payload, errFetch := balanceJSON(queryFlag(req.Query, "refresh"), "")
	if errFetch != nil && payload == nil {
		return jsonError(http.StatusBadRequest, errFetch.Error())
	}
	var report balanceReport
	if errUnmarshal := json.Unmarshal(payload, &report); errUnmarshal != nil {
		return jsonError(http.StatusInternalServerError, errUnmarshal.Error())
	}
	summaries := make([]sessionSummary, 0, len(report.Sessions))
	for _, sess := range report.Sessions {
		summaries = append(summaries, sess.sessionSummary)
	}
	body, _ := json.Marshal(map[string]any{
		"ok":                       report.OK,
		"error":                    report.Error,
		"fetched_at":               report.FetchedAt,
		"refresh_interval_seconds": report.RefreshIntervalSeconds,
		"poll_concurrency":         report.PollConcurrency,
		"session_count":            report.SessionCount,
		"sessions":                 summaries,
	})
	return managementResponse{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: body}
}

func apiKeysResponse(req managementRequest) managementResponse {
	if strings.EqualFold(strings.TrimSpace(req.Method), http.MethodPost) {
		return handleCreateAPIKey(req)
	}
	return handleListAPIKeys(req)
}

func handleListAPIKeys(req managementRequest) managementResponse {
	cfg, _ := snapshotConfig()
	sessionID := queryValue(req.Query, "session")
	sess, ok := cfg.sessionByID(sessionID)
	if !ok {
		if sessionID == "" && len(cfg.resolvedSessions()) > 1 {
			return jsonError(http.StatusBadRequest, "session is required when multiple sessions are configured")
		}
		return jsonError(http.StatusBadRequest, "unknown or unconfigured session")
	}
	keys, errList := listAPIKeys(cfg, sess.TRSession)
	if errList != nil {
		body, _ := json.Marshal(listAPIKeysResponse{OK: false, Error: errList.Error(), Session: sess.ID})
		return managementResponse{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: body}
	}
	body, _ := json.Marshal(listAPIKeysResponse{OK: true, Session: sess.ID, APIKeys: keys})
	return managementResponse{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: body}
}

func handleCreateAPIKey(req managementRequest) managementResponse {
	cfg, _ := snapshotConfig()
	var bodyReq createAPIKeyRequest
	if len(req.Body) > 0 {
		if errUnmarshal := json.Unmarshal(req.Body, &bodyReq); errUnmarshal != nil {
			return jsonError(http.StatusBadRequest, "invalid JSON body: "+errUnmarshal.Error())
		}
	}
	sessionID := strings.TrimSpace(bodyReq.Session)
	if sessionID == "" {
		sessionID = queryValue(req.Query, "session")
	}
	sess, ok := cfg.sessionByID(sessionID)
	if !ok {
		if sessionID == "" && len(cfg.resolvedSessions()) > 1 {
			return jsonError(http.StatusBadRequest, "session is required when multiple sessions are configured")
		}
		return jsonError(http.StatusBadRequest, "unknown or unconfigured session")
	}
	created, errCreate := createAPIKey(cfg, sess, bodyReq.Name)
	if errCreate != nil {
		body, _ := json.Marshal(createAPIKeyResponse{OK: false, Error: errCreate.Error(), Session: sess.ID})
		return managementResponse{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: body}
	}
	invalidateSessionCache(sess.ID)
	body, _ := json.Marshal(createAPIKeyResponse{OK: true, Session: sess.ID, APIKey: created})
	return managementResponse{StatusCode: http.StatusOK, Headers: jsonHeaders(), Body: body}
}

func jsonError(status int, message string) managementResponse {
	body, _ := json.Marshal(map[string]any{"ok": false, "error": message})
	return managementResponse{StatusCode: status, Headers: jsonHeaders(), Body: body}
}

func resourceResponse(html string) managementResponse {
	return managementResponse{
		StatusCode: http.StatusOK,
		Headers: map[string][]string{
			"Content-Type":  {"text/html; charset=utf-8"},
			"Cache-Control": {"no-store"},
		},
		Body: []byte(html),
	}
}

func jsonHeaders() map[string][]string {
	return map[string][]string{
		"Content-Type":  {"application/json; charset=utf-8"},
		"Cache-Control": {"no-store"},
	}
}
