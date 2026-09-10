package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

func TestManagementRegisterIncludesCreateAPIRoute(t *testing.T) {
	reg := managementRegistrationResponse()
	var hasCreate bool
	for _, route := range reg.Routes {
		if route.Method == http.MethodPost && strings.HasSuffix(route.Path, "/api-keys") {
			hasCreate = true
		}
	}
	if !hasCreate {
		t.Fatalf("expected POST /tokenrhythm/api-keys route, got %#v", reg.Routes)
	}
}

func TestCreateAPIKeyWithoutCSRFStillPosts(t *testing.T) {
	applyConfig([]byte("tr_session: sess_x\n"))
	var sawPost bool
	hostCall = func(method string, payload any) (json.RawMessage, error) {
		raw, _ := json.Marshal(payload)
		var req struct {
			Method string `json:"method"`
			URL    string `json:"url"`
		}
		_ = json.Unmarshal(raw, &req)
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL, "/api/api-keys") {
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":[]}`)})
		}
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL, "/api/api-keys") {
			sawPost = true
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":{"id":"k1","name":"test","key":"sk_x","status":"enabled"}}`)})
		}
		return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":{}}`)})
	}
	t.Cleanup(func() { hostCall = nil })
	raw, errHandle := handleManagement(mustJSON(t, managementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/tokenrhythm/api-keys",
		Body:   []byte(`{"name":"test"}`),
	}))
	if errHandle != nil {
		t.Fatalf("handleManagement() error = %v", errHandle)
	}
	resp := decodeManagement(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, resp.Body)
	}
	if !sawPost {
		t.Fatal("expected POST /api/api-keys even without tr_csrf")
	}
}

func TestCreateAPIKeyPostsUpstream(t *testing.T) {
	applyConfig([]byte("sessions:\n  - id: main\n    name: 主账号\n    tr_session: 'tr_session=sess_a; tr_csrf=csrf_a'\n"))
	var sawPost bool
	hostCall = func(method string, payload any) (json.RawMessage, error) {
		if method != pluginabi.MethodHostHTTPDo {
			t.Fatalf("unexpected host method %s", method)
		}
		raw, _ := json.Marshal(payload)
		var req struct {
			Method  string              `json:"method"`
			URL     string              `json:"url"`
			Headers map[string][]string `json:"headers"`
			Body    []byte              `json:"body"`
		}
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			t.Fatalf("decode host payload: %v", errUnmarshal)
		}
		if req.Method == http.MethodGet && strings.HasSuffix(req.URL, "/api/api-keys") {
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"message":"ok","data":[]}`)})
		}
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL, "/api/api-keys") {
			sawPost = true
			csrf := headerValue(req.Headers, "X-CSRF-Token")
			if csrf != "csrf_a" {
				t.Fatalf("X-CSRF-Token = %q", csrf)
			}
			if !strings.Contains(string(req.Body), `"name":"studio-key"`) {
				t.Fatalf("body = %s", req.Body)
			}
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"message":"ok","data":{"id":"k1","name":"studio-key","key":"sk_secret_value","maskedKey":"sk_se***","status":"enabled"}}`)})
		}
		return json.Marshal(hostHTTPResponse{StatusCode: 404, Body: []byte(`{"code":1,"message":"missing stub "+req.URL}`)})
	}
	t.Cleanup(func() { hostCall = nil })

	raw, errHandle := handleManagement(mustJSON(t, managementRequest{
		Method: http.MethodPost,
		Path:   "/v0/management/tokenrhythm/api-keys",
		Body:   []byte(`{"session":"main","name":"studio-key"}`),
	}))
	if errHandle != nil {
		t.Fatalf("handleManagement() error = %v", errHandle)
	}
	resp := decodeManagement(t, raw)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, resp.Body)
	}
	if !sawPost {
		t.Fatal("expected POST /api/api-keys")
	}
	var created createAPIKeyResponse
	if errUnmarshal := json.Unmarshal(resp.Body, &created); errUnmarshal != nil {
		t.Fatalf("decode body: %v", errUnmarshal)
	}
	if !created.OK || created.APIKey.Key != "sk_secret_value" || created.Session != "main" {
		t.Fatalf("created = %#v", created)
	}
}

func TestBalanceJSONPollsAllSessions(t *testing.T) {
	applyConfig([]byte("sessions:\n  - id: a\n    tr_session: sess_a\n  - id: b\n    tr_session: sess_b\nrefresh_interval_seconds: 5\n"))
	hostCall = func(method string, payload any) (json.RawMessage, error) {
		raw, _ := json.Marshal(payload)
		var req struct {
			URL     string              `json:"url"`
			Headers map[string][]string `json:"headers"`
		}
		_ = json.Unmarshal(raw, &req)
		cookie := headerValue(req.Headers, "Cookie")
		balance := "10.00"
		if strings.Contains(cookie, "sess_b") {
			balance = "20.00"
		}
		if strings.HasSuffix(req.URL, "/api/wallet/summary") {
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":{"availableBalanceCny":"` + balance + `","currency":"CNY"}}`)})
		}
		if strings.HasSuffix(req.URL, "/api/api-keys") {
			return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":[]}`)})
		}
		return json.Marshal(hostHTTPResponse{StatusCode: 200, Body: []byte(`{"code":0,"data":{}}`)})
	}
	t.Cleanup(func() { hostCall = nil })

	payload, errFetch := balanceJSON(true, "")
	if errFetch != nil {
		t.Fatalf("balanceJSON() error = %v", errFetch)
	}
	var report balanceReport
	if errUnmarshal := json.Unmarshal(payload, &report); errUnmarshal != nil {
		t.Fatalf("decode report: %v", errUnmarshal)
	}
	if !report.OK || report.SessionCount != 2 || report.Available != 30 {
		t.Fatalf("report = %#v", report)
	}
	if len(report.Sessions) != 2 || !report.Sessions[0].OK || !report.Sessions[1].OK {
		t.Fatalf("sessions = %#v", report.Sessions)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		t.Fatalf("marshal: %v", errMarshal)
	}
	return raw
}

func decodeManagement(t *testing.T, raw []byte) managementResponse {
	t.Helper()
	var env envelope
	if errUnmarshal := json.Unmarshal(raw, &env); errUnmarshal != nil {
		t.Fatalf("decode envelope: %v\n%s", errUnmarshal, raw)
	}
	if !env.OK {
		t.Fatalf("envelope not ok: %s", raw)
	}
	var resp managementResponse
	if errUnmarshal := json.Unmarshal(env.Result, &resp); errUnmarshal != nil {
		t.Fatalf("decode management response: %v\n%s", errUnmarshal, env.Result)
	}
	return resp
}

func headerValue(headers map[string][]string, name string) string {
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
