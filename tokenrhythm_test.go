package main

import (
	"sync"
	"testing"
)

func TestCookieHeader(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"sess_abc", "tr_session=sess_abc"},
		{"tr_session=sess_abc", "tr_session=sess_abc"},
		{"tr_session=sess_abc; tr_csrf=xyz", "tr_session=sess_abc; tr_csrf=xyz"},
	}
	for _, tc := range cases {
		if got := cookieHeader(tc.in); got != tc.want {
			t.Fatalf("cookieHeader(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAPIURL(t *testing.T) {
	if got := apiURL("https://tokenrhythm.studio/", "/api/wallet/summary"); got != "https://tokenrhythm.studio/api/wallet/summary" {
		t.Fatalf("apiURL() = %q", got)
	}
}

func TestParseFloat(t *testing.T) {
	if got := parseFloat("61.17711198"); got != 61.17711198 {
		t.Fatalf("parseFloat() = %v", got)
	}
	if got := parseFloat("not-a-number"); got != 0 {
		t.Fatalf("parseFloat(invalid) = %v, want 0", got)
	}
}

func TestApplyConfigNormalizes(t *testing.T) {
	cfg := applyConfig([]byte("enabled: true\npriority: 3\ntr_session: sess_x\nbase_url: 'https://example.com/'\nrefresh_interval_seconds: 1\nlow_balance_threshold: -5\n"))
	if cfg.TRSession != "sess_x" {
		t.Fatalf("TRSession = %q", cfg.TRSession)
	}
	if cfg.BaseURL != "https://example.com" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.RefreshIntervalSeconds != 5 {
		t.Fatalf("RefreshIntervalSeconds = %d, want 5", cfg.RefreshIntervalSeconds)
	}
	if cfg.LowBalanceThreshold != 0 {
		t.Fatalf("LowBalanceThreshold = %v, want 0", cfg.LowBalanceThreshold)
	}
	if cfg.PollConcurrency != defaultPollConcurrent {
		t.Fatalf("PollConcurrency = %d, want %d", cfg.PollConcurrency, defaultPollConcurrent)
	}
}

func TestApplyConfigDefaultsOnInvalidYAML(t *testing.T) {
	cfg := applyConfig([]byte(":\n\tbad"))
	if cfg.BaseURL != defaultBase {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, defaultBase)
	}
	if cfg.RefreshIntervalSeconds != 60 {
		t.Fatalf("RefreshIntervalSeconds = %d, want 60", cfg.RefreshIntervalSeconds)
	}
	if cfg.PollConcurrency != defaultPollConcurrent {
		t.Fatalf("PollConcurrency = %d, want %d", cfg.PollConcurrency, defaultPollConcurrent)
	}
}

func TestCSRFFromCookie(t *testing.T) {
	if got := csrfFromCookie("tr_session=sess_abc; tr_csrf=xyz"); got != "xyz" {
		t.Fatalf("csrfFromCookie() = %q", got)
	}
	if got := csrfFromCookie("sess_abc"); got != "" {
		t.Fatalf("csrfFromCookie(raw session) = %q, want empty", got)
	}
}

func TestResolveSessionsFromTRSession(t *testing.T) {
	cfg := applyConfig([]byte("tr_session: sess_x\n"))
	sessions := cfg.resolvedSessions()
	if len(sessions) != 1 || sessions[0].ID != "default" || sessions[0].TRSession != "sess_x" {
		t.Fatalf("resolvedSessions() = %#v", sessions)
	}
}

func TestResolveSessionsFromArray(t *testing.T) {
	cfg := applyConfig([]byte("sessions:\n  - id: main\n    name: 主账号\n    tr_session: sess_a\n  - name: 备用\n    tr_session: 'tr_session=sess_b; tr_csrf=csrf_b'\n  - tr_session: ''\n"))
	sessions := cfg.resolvedSessions()
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2: %#v", len(sessions), sessions)
	}
	if sessions[0].ID != "main" || sessions[0].Name != "主账号" || sessions[0].TRSession != "sess_a" {
		t.Fatalf("sessions[0] = %#v", sessions[0])
	}
	if sessions[1].Name != "备用" || csrfFromCookie(sessions[1].TRSession) != "csrf_b" {
		t.Fatalf("sessions[1] = %#v", sessions[1])
	}
}

func TestResolveSessionsDedupesIDs(t *testing.T) {
	cfg := applyConfig([]byte("sessions:\n  - id: main\n    tr_session: a\n  - id: main\n    tr_session: b\n"))
	sessions := cfg.resolvedSessions()
	if len(sessions) != 2 || sessions[0].ID != "main" || sessions[1].ID != "main-2" {
		t.Fatalf("resolvedSessions() = %#v", sessions)
	}
}

func TestSessionSlug(t *testing.T) {
	if got := sessionSlug("备用 Account 1"); got != "account-1" {
		t.Fatalf("sessionSlug() = %q", got)
	}
}

func TestParseAPIKeyList(t *testing.T) {
	keys := parseAPIKeyList([]byte(`[{"id":"1","name":"A","key":"sk_secret","status":"enabled"},{"id":"2","name":"B","status":"deleted"}]`))
	if len(keys) != 1 || keys[0].ID != "1" || keys[0].Key != "" || keys[0].MaskedKey == "" {
		t.Fatalf("parseAPIKeyList() = %#v", keys)
	}
}

func TestMaskAPIKey(t *testing.T) {
	if got := maskAPIKey("sk_abcdefghijklmnop"); got != "sk_abc...mnop" {
		t.Fatalf("maskAPIKey() = %q", got)
	}
}

func TestRememberCSRFFromSetCookie(t *testing.T) {
	csrfMemo = sync.Map{}
	session := "sess_x"
	rememberCSRF(session, map[string][]string{
		"Set-Cookie": {"tr_csrf=token-from-set-cookie; Path=/; HttpOnly"},
	})
	if got := csrfToken(session); got != "token-from-set-cookie" {
		t.Fatalf("csrfToken() = %q", got)
	}
}
