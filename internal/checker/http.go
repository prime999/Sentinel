package checker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
	"github.com/sentinel-monitoring/sentinel/internal/safehost"
)

type timingInfo struct {
	dnsStart, dnsDone         time.Time
	connectStart, connectDone time.Time
	tlsStart, tlsDone         time.Time
	gotFirstResponseByte      time.Time
}

func (c *Checker) probeHTTP(ctx context.Context, m *models.Monitor) *models.CheckResult {
	start := time.Now()
	result := &models.CheckResult{
		MonitorID: m.ID,
		Status:    models.StatusDown,
		CheckedAt: start,
	}

	var timings timingInfo
	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) { timings.dnsStart = time.Now() },
		DNSDone:  func(_ httptrace.DNSDoneInfo) { timings.dnsDone = time.Now() },
		ConnectStart: func(_, _ string) {
			if timings.connectStart.IsZero() {
				timings.connectStart = time.Now()
			}
		},
		ConnectDone: func(_, _ string, _ error) {
			if timings.connectDone.IsZero() {
				timings.connectDone = time.Now()
			}
		},
		TLSHandshakeStart: func() { timings.tlsStart = time.Now() },
		TLSHandshakeDone:  func(_ tls.ConnectionState, _ error) { timings.tlsDone = time.Now() },
		GotFirstResponseByte: func() {
			if timings.gotFirstResponseByte.IsZero() {
				timings.gotFirstResponseByte = time.Now()
			}
		},
	}

	timeout := time.Duration(m.TimeoutMs) * time.Millisecond
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	reqCtx = httptrace.WithClientTrace(reqCtx, trace)

	if err := safehost.ValidateHTTPURL(m.URL); err != nil {
		result.Error = err.Error()
		result.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return result
	}

	method := m.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	if m.RequestBody != "" {
		body = strings.NewReader(m.RequestBody)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, m.URL, body)
	if err != nil {
		result.Error = fmt.Sprintf("invalid request: %v", err)
		result.ResponseTimeMs = int(time.Since(start).Milliseconds())
		return result
	}

	if m.RequestHeaders != "" {
		for _, line := range strings.Split(m.RequestHeaders, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:         safehost.ControlDialContext,
			TLSHandshakeTimeout: timeout,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: false},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if !m.FollowRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if err := safehost.ValidateHTTPURL(req.URL.String()); err != nil {
				return err
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	result.ResponseTimeMs = int(time.Since(start).Milliseconds())
	fillTimings(result, timings, start)

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	result.StatusCode = &code

	if !statusMatches(m, code) {
		result.Error = fmt.Sprintf("expected status %s, got %d", expectedStatusLabel(m), code)
		return result
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.Error = fmt.Sprintf("read body: %v", err)
		return result
	}
	bodyStr := string(bodyBytes)

	if m.KeywordMustExist != "" && !strings.Contains(bodyStr, m.KeywordMustExist) {
		result.Error = fmt.Sprintf("keyword not found: %q", m.KeywordMustExist)
		return result
	}
	if m.KeywordMustNotExist != "" && strings.Contains(bodyStr, m.KeywordMustNotExist) {
		result.Error = fmt.Sprintf("forbidden keyword found: %q", m.KeywordMustNotExist)
		return result
	}

	if m.SlowThresholdMs > 0 && result.ResponseTimeMs > m.SlowThresholdMs {
		result.Status = models.StatusDegraded
	} else {
		result.Status = models.StatusUp
	}
	return result
}

func fillTimings(result *models.CheckResult, t timingInfo, start time.Time) {
	if !t.dnsStart.IsZero() && !t.dnsDone.IsZero() {
		v := int(t.dnsDone.Sub(t.dnsStart).Milliseconds())
		result.DNSMs = &v
	}
	if !t.connectStart.IsZero() && !t.connectDone.IsZero() {
		v := int(t.connectDone.Sub(t.connectStart).Milliseconds())
		result.TCPMs = &v
	}
	if !t.tlsStart.IsZero() && !t.tlsDone.IsZero() {
		v := int(t.tlsDone.Sub(t.tlsStart).Milliseconds())
		result.TLSMs = &v
	}
	if !t.gotFirstResponseByte.IsZero() {
		v := int(t.gotFirstResponseByte.Sub(start).Milliseconds())
		result.TTFBMs = &v
	}
}

func statusMatches(m *models.Monitor, code int) bool {
	if m.ExpectedStatusMin != nil && m.ExpectedStatusMax != nil {
		return code >= *m.ExpectedStatusMin && code <= *m.ExpectedStatusMax
	}
	if m.ExpectedStatus == 0 {
		return code >= 200 && code < 300
	}
	if m.ExpectedStatus == 2 {
		return code >= 200 && code < 300
	}
	return code == m.ExpectedStatus
}

func expectedStatusLabel(m *models.Monitor) string {
	if m.ExpectedStatusMin != nil && m.ExpectedStatusMax != nil {
		return fmt.Sprintf("%d-%d", *m.ExpectedStatusMin, *m.ExpectedStatusMax)
	}
	if m.ExpectedStatus == 2 {
		return "2xx"
	}
	return fmt.Sprintf("%d", m.ExpectedStatus)
}
