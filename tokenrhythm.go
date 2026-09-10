package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36 Edg/152.0.0.0"

type hostHTTPResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers"`
	Body       []byte              `json:"Body"`
}

type trEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	TraceID string          `json:"traceId"`
}

type walletData struct {
	Currency            string `json:"currency"`
	AvailableBalanceCny string `json:"availableBalanceCny"`
	GiftAvailableCny    string `json:"giftAvailableCny"`
	GiftLockedCny       string `json:"giftLockedCny"`
	RechargeBalanceCny  string `json:"rechargeBalanceCny"`
	DebtBalanceCny      string `json:"debtBalanceCny"`
	FrozenBalanceCny    string `json:"frozenBalanceCny"`
	GiftTotalCny        string `json:"giftTotalCny"`
	GiftStatus          string `json:"giftStatus"`
	VoidedGiftCny       string `json:"voidedGiftCny"`
	AsOf                string `json:"asOf"`
}

type signupReward struct {
	Policy                    string `json:"policy"`
	TotalEligibleCny          string `json:"totalEligibleCny"`
	GrantedCny                string `json:"grantedCny"`
	PendingActivationCny      string `json:"pendingActivationCny"`
	Status                    string `json:"status"`
	RewardValidityDays        int    `json:"rewardValidityDays"`
	QualifiedAt               string `json:"qualifiedAt"`
	ActivationRewardGrantedAt string `json:"activationRewardGrantedAt"`
	ActivationRewardExpiresAt string `json:"activationRewardExpiresAt"`
}

type usageData struct {
	Calls                int64         `json:"calls"`
	SuccessCalls         int64         `json:"successCalls"`
	ErrorCalls           int64         `json:"errorCalls"`
	AbortedCalls         int64         `json:"abortedCalls"`
	TokenRequestCount    int64         `json:"tokenRequestCount"`
	ImageRequestCount    int64         `json:"imageRequestCount"`
	SuccessfulImageCount int64         `json:"successfulImageCount"`
	InputTokens          int64         `json:"inputTokens"`
	OutputTokens         int64         `json:"outputTokens"`
	CostCny              string        `json:"costCny"`
	TokenCostCny         string        `json:"tokenCostCny"`
	ImageCostCny         string        `json:"imageCostCny"`
	BalanceCny           string        `json:"balanceCny"`
	FrozenBalanceCny     string        `json:"frozenBalanceCny"`
	AvailableBalanceCny  string        `json:"availableBalanceCny"`
	ExpiringBalanceCny   string        `json:"expiringBalanceCny"`
	NextExpiryAt         string        `json:"nextExpiryAt"`
	Currency             string        `json:"currency"`
	SignupReward         *signupReward `json:"signupReward"`
}

type accountData struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	EmailMasked *string `json:"emailMasked"`
	PhoneMasked string  `json:"phoneMasked"`
	Status      string  `json:"status"`
	Role        string  `json:"role"`
	JoinedAt    string  `json:"joinedAt"`
}

type panelSummary struct {
	Calls           int64  `json:"calls"`
	SuccessCalls    int64  `json:"successCalls"`
	ErrorCalls      int64  `json:"errorCalls"`
	InputTokens     int64  `json:"inputTokens"`
	OutputTokens    int64  `json:"outputTokens"`
	TotalTokens     int64  `json:"totalTokens"`
	CacheReadTokens int64  `json:"cacheReadTokens"`
	ReasoningTokens int64  `json:"reasoningTokens"`
	CostCny         string `json:"costCny"`
	ActualCostUsd   string `json:"actualCostUsd"`
	TokenSavingCny  string `json:"tokenSavingCny"`
}

type modelUsage struct {
	ModelID         string `json:"modelId"`
	Model           string `json:"model"`
	Calls           int64  `json:"calls"`
	InputTokens     int64  `json:"inputTokens"`
	OutputTokens    int64  `json:"outputTokens"`
	TotalTokens     int64  `json:"totalTokens"`
	CacheReadTokens int64  `json:"cacheReadTokens"`
	CostCny         string `json:"costCny"`
}

type usagePanel struct {
	Summary  panelSummary `json:"summary"`
	ByModel  []modelUsage `json:"byModel"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type expiringSummary struct {
	ExpiringBalanceCny string `json:"expiringBalanceCny"`
	NextExpiryAt       string `json:"nextExpiryAt"`
}

type expiringCredit struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	SourceLabel  string `json:"sourceLabel"`
	GrantedCny   string `json:"grantedCny"`
	RemainingCny string `json:"remainingCny"`
	GrantedAt    string `json:"grantedAt"`
	ExpiresAt    string `json:"expiresAt"`
}

type expiringData struct {
	AsOf    string           `json:"asOf"`
	Summary expiringSummary  `json:"summary"`
	List    []expiringCredit `json:"list"`
}

type sessionSummary struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	OK          bool    `json:"ok"`
	Error       string  `json:"error,omitempty"`
	FetchedAt   string  `json:"fetched_at,omitempty"`
	Available   float64 `json:"available"`
	LowBalance  bool    `json:"low_balance"`
	AccountName string  `json:"account_name,omitempty"`
	APIKeyCount int     `json:"api_key_count"`
	HasCSRF     bool    `json:"has_csrf"`
}

// sessionReport is one Token Rhythm account snapshot.
type sessionReport struct {
	sessionSummary
	Wallet   *walletData   `json:"wallet,omitempty"`
	Usage    *usageData    `json:"usage,omitempty"`
	Panel    *usagePanel   `json:"panel,omitempty"`
	Account  *accountData  `json:"account,omitempty"`
	Expiring *expiringData `json:"expiring,omitempty"`
	APIKeys  []apiKey      `json:"api_keys,omitempty"`
}

// balanceReport is the JSON payload returned by the management API and consumed
// by the resource dashboard.
type balanceReport struct {
	OK                     bool            `json:"ok"`
	Error                  string          `json:"error,omitempty"`
	FetchedAt              string          `json:"fetched_at"`
	BaseURL                string          `json:"base_url"`
	RefreshIntervalSeconds int             `json:"refresh_interval_seconds"`
	LowBalanceThreshold    float64         `json:"low_balance_threshold"`
	PollConcurrency        int             `json:"poll_concurrency"`
	LowBalance             bool            `json:"low_balance"`
	Available              float64         `json:"available"`
	SelectedSession        string          `json:"selected_session,omitempty"`
	SessionCount           int             `json:"session_count"`
	Sessions               []sessionReport `json:"sessions"`
	Wallet                 *walletData     `json:"wallet,omitempty"`
	Usage                  *usageData      `json:"usage,omitempty"`
	Panel                  *usagePanel     `json:"panel,omitempty"`
	Account                *accountData    `json:"account,omitempty"`
	Expiring               *expiringData   `json:"expiring,omitempty"`
	APIKeys                []apiKey        `json:"api_keys,omitempty"`
}

type cachedSessionReport struct {
	at     time.Time
	report sessionReport
}

type cacheEntry struct {
	mu    sync.Mutex
	epoch uint64
	byID  map[string]cachedSessionReport
}

var reportCache cacheEntry

func cookieHeader(session string) string {
	session = strings.TrimSpace(session)
	if session == "" {
		return ""
	}
	if strings.ContainsAny(session, "=;") {
		return session
	}
	return "tr_session=" + session
}

func cookieValue(header, name string) string {
	prefix := name + "="
	for _, part := range strings.Split(cookieHeader(header), ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(part, prefix))
		}
	}
	return ""
}

func csrfFromCookie(session string) string {
	return cookieValue(session, "tr_csrf")
}

func apiURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}

func isSafeHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

type hostInvoker func(method string, payload any) (json.RawMessage, error)

var hostCall hostInvoker

func invokeHost(method string, payload any) (json.RawMessage, error) {
	if hostCall == nil {
		return nil, fmt.Errorf("host is not available")
	}
	return hostCall(method, payload)
}

func doTrRequest(cfg pluginConfig, session, method, path string, body []byte) ([]byte, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	headers := map[string][]string{
		"Accept":          {"*/*"},
		"Accept-Language": {"zh-CN,zh;q=0.9,en;q=0.8"},
		"Cookie":          {cookieHeader(session)},
		"Origin":          {cfg.BaseURL},
		"Referer":         {apiURL(cfg.BaseURL, "/account/keys")},
		"User-Agent":      {defaultUserAgent},
	}
	if len(body) > 0 {
		headers["Content-Type"] = []string{"application/json"}
	}
	if csrf := csrfFromCookie(session); csrf != "" && !isSafeHTTPMethod(method) {
		headers["X-CSRF-Token"] = []string{csrf}
	}
	req := map[string]any{
		"method":  method,
		"url":     apiURL(cfg.BaseURL, path),
		"headers": headers,
	}
	if len(body) > 0 {
		req["body"] = body
	}
	result, errCall := invokeHost(pluginabi.MethodHostHTTPDo, req)
	if errCall != nil {
		return nil, errCall
	}
	var resp hostHTTPResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return nil, fmt.Errorf("decode host http response: %w", errUnmarshal)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyText := strings.TrimSpace(string(resp.Body))
		if len(bodyText) > 200 {
			bodyText = bodyText[:200]
		}
		return nil, fmt.Errorf("upstream returned HTTP %d: %s", resp.StatusCode, bodyText)
	}
	var env trEnvelope
	if errUnmarshal := json.Unmarshal(resp.Body, &env); errUnmarshal != nil {
		return nil, fmt.Errorf("decode tokenrhythm response: %w", errUnmarshal)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("tokenrhythm error %d: %s", env.Code, env.Message)
	}
	return env.Data, nil
}

func doTrGet(cfg pluginConfig, session, path string) ([]byte, error) {
	return doTrRequest(cfg, session, http.MethodGet, path, nil)
}

func parseFloat(raw string) float64 {
	value, errParse := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if errParse != nil {
		return 0
	}
	return value
}

func tryGetJSON(cfg pluginConfig, session, path string, out any) error {
	raw, errGet := doTrGet(cfg, session, path)
	if errGet != nil {
		return errGet
	}
	return json.Unmarshal(raw, out)
}

func fetchSessionReport(cfg pluginConfig, sess resolvedSession) sessionReport {
	now := time.Now().UTC().Format(time.RFC3339)
	report := sessionReport{
		sessionSummary: sessionSummary{
			ID:        sess.ID,
			Name:      sess.Name,
			FetchedAt: now,
			HasCSRF:   csrfFromCookie(sess.TRSession) != "",
		},
	}

	walletRaw, errWallet := doTrGet(cfg, sess.TRSession, "/api/wallet/summary")
	if errWallet != nil {
		report.Error = "wallet summary: " + errWallet.Error()
		return report
	}
	var wallet walletData
	if errUnmarshal := json.Unmarshal(walletRaw, &wallet); errUnmarshal != nil {
		report.Error = "decode wallet summary: " + errUnmarshal.Error()
		return report
	}

	var usage usageData
	var hasUsage bool
	if errUsage := tryGetJSON(cfg, sess.TRSession, "/api/usage-summary", &usage); errUsage == nil {
		hasUsage = true
	}

	var panel usagePanel
	var hasPanel bool
	if errPanel := tryGetJSON(cfg, sess.TRSession, "/api/usage/panel", &panel); errPanel == nil {
		hasPanel = true
	}

	var account accountData
	var hasAccount bool
	if errAccount := tryGetJSON(cfg, sess.TRSession, "/api/me", &account); errAccount == nil && account.Name != "" {
		hasAccount = true
	}

	var expiring expiringData
	var hasExpiring bool
	if errExpiring := tryGetJSON(cfg, sess.TRSession, "/api/wallet/expiring-credits?page=1&pageSize=20", &expiring); errExpiring == nil {
		hasExpiring = true
	}

	keys, _ := listAPIKeys(cfg, sess.TRSession)

	available := parseFloat(wallet.AvailableBalanceCny)
	if wallet.AvailableBalanceCny == "" {
		available = parseFloat(usage.AvailableBalanceCny)
	}
	name := sess.Name
	if hasAccount && account.Name != "" && (name == "" || name == sess.ID || name == "default") {
		name = account.Name
	}

	report.OK = true
	report.Name = name
	report.Available = available
	report.LowBalance = available < cfg.LowBalanceThreshold
	report.Wallet = &wallet
	report.APIKeys = keys
	report.APIKeyCount = len(keys)
	if hasAccount {
		report.Account = &account
		report.AccountName = account.Name
	}
	if hasUsage {
		report.Usage = &usage
	}
	if hasPanel {
		report.Panel = &panel
	}
	if hasExpiring {
		report.Expiring = &expiring
	}
	return report
}

func pollSessions(cfg pluginConfig, sessions []resolvedSession, force bool, epoch uint64) []sessionReport {
	reports := make([]sessionReport, len(sessions))
	if len(sessions) == 0 {
		return reports
	}
	ttl := time.Duration(cfg.RefreshIntervalSeconds) * time.Second
	concurrency := cfg.PollConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(sessions) {
		concurrency = len(sessions)
	}

	type job struct {
		index int
		sess  resolvedSession
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if !force {
					if cached, ok := cachedSession(epoch, item.sess.ID, ttl); ok {
						reports[item.index] = cached
						continue
					}
				}
				report := fetchSessionReport(cfg, item.sess)
				storeCachedSession(epoch, report)
				reports[item.index] = report
			}
		}()
	}
	for i, sess := range sessions {
		jobs <- job{index: i, sess: sess}
	}
	close(jobs)
	wg.Wait()
	return reports
}

func cachedSession(epoch uint64, id string, ttl time.Duration) (sessionReport, bool) {
	reportCache.mu.Lock()
	defer reportCache.mu.Unlock()
	if reportCache.epoch != epoch || reportCache.byID == nil {
		return sessionReport{}, false
	}
	cached, ok := reportCache.byID[id]
	if !ok || time.Since(cached.at) >= ttl {
		return sessionReport{}, false
	}
	return cached.report, true
}

func storeCachedSession(epoch uint64, report sessionReport) {
	reportCache.mu.Lock()
	defer reportCache.mu.Unlock()
	if reportCache.byID == nil || reportCache.epoch != epoch {
		reportCache.byID = make(map[string]cachedSessionReport)
		reportCache.epoch = epoch
	}
	reportCache.byID[report.ID] = cachedSessionReport{at: time.Now(), report: report}
}

func invalidateSessionCache(id string) {
	reportCache.mu.Lock()
	defer reportCache.mu.Unlock()
	if reportCache.byID != nil {
		delete(reportCache.byID, id)
	}
}

func assembleReport(cfg pluginConfig, sessions []sessionReport, selectedID string) balanceReport {
	now := time.Now().UTC().Format(time.RFC3339)
	report := balanceReport{
		FetchedAt:              now,
		BaseURL:                cfg.BaseURL,
		RefreshIntervalSeconds: cfg.RefreshIntervalSeconds,
		LowBalanceThreshold:    cfg.LowBalanceThreshold,
		PollConcurrency:        cfg.PollConcurrency,
		SessionCount:           len(sessions),
		Sessions:               sessions,
	}
	var selected *sessionReport
	var firstOK *sessionReport
	var available float64
	var anyOK bool
	var lastErr string
	for i := range sessions {
		sess := &sessions[i]
		if sess.OK {
			anyOK = true
			available += sess.Available
			if sess.LowBalance {
				report.LowBalance = true
			}
			if firstOK == nil {
				firstOK = sess
			}
			if selectedID != "" && sess.ID == selectedID {
				selected = sess
			}
		} else if sess.Error != "" {
			lastErr = sess.Error
		}
	}
	if selected == nil {
		selected = firstOK
	}
	if selected != nil {
		report.SelectedSession = selected.ID
		report.Wallet = selected.Wallet
		report.Usage = selected.Usage
		report.Panel = selected.Panel
		report.Account = selected.Account
		report.Expiring = selected.Expiring
		report.APIKeys = selected.APIKeys
	}
	report.Available = available
	if !anyOK {
		report.Error = lastErr
		if report.Error == "" {
			report.Error = "all sessions failed"
		}
		return report
	}
	report.OK = true
	if !report.LowBalance && available < cfg.LowBalanceThreshold {
		report.LowBalance = true
	}
	return report
}

func fetchReport(selectedID string, force bool) (balanceReport, error) {
	cfg, epoch := snapshotConfig()
	sessions := cfg.resolvedSessions()
	if len(sessions) == 0 {
		return balanceReport{}, fmt.Errorf("no Token Rhythm session is configured (set tr_session or sessions)")
	}
	if selectedID != "" {
		if _, ok := cfg.sessionByID(selectedID); !ok {
			return balanceReport{}, fmt.Errorf("unknown session %q", selectedID)
		}
	}
	polled := pollSessions(cfg, sessions, force, epoch)
	report := assembleReport(cfg, polled, selectedID)
	if !report.OK {
		return report, fmt.Errorf("%s", report.Error)
	}
	return report, nil
}

// balanceJSON returns the cached report JSON, refreshing when stale or forced.
func balanceJSON(force bool, selectedID string) ([]byte, error) {
	report, errFetch := fetchReport(selectedID, force)
	if errFetch != nil && report.Sessions == nil {
		cfg, _ := snapshotConfig()
		failure, _ := json.Marshal(balanceReport{
			OK:                     false,
			Error:                  errFetch.Error(),
			FetchedAt:              time.Now().UTC().Format(time.RFC3339),
			BaseURL:                cfg.BaseURL,
			RefreshIntervalSeconds: cfg.RefreshIntervalSeconds,
			LowBalanceThreshold:    cfg.LowBalanceThreshold,
			PollConcurrency:        cfg.PollConcurrency,
			Sessions:               []sessionReport{},
		})
		return failure, errFetch
	}
	payload, errMarshal := json.Marshal(report)
	if errMarshal != nil {
		return nil, errMarshal
	}
	if errFetch != nil {
		return payload, errFetch
	}
	return payload, nil
}

func balanceJSONString() string {
	payload, _ := balanceJSON(false, "")
	return string(payload)
}
