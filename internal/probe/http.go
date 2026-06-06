package probe

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

// probeHTTP issues a single request and records only the status code. It never
// reads the response body and never sends auth headers.
func probeHTTP(h config.HTTPProbe, timeout time.Duration) model.HTTPProbeResult {
	method := strings.ToUpper(strings.TrimSpace(h.Method))
	if method == "" {
		method = http.MethodHead
	}
	expected := h.ExpectedStatus
	if expected == 0 {
		expected = http.StatusOK
	}

	r := model.HTTPProbeResult{
		Name:           h.Name,
		URL:            h.URL,
		Method:         method,
		Required:       h.Required,
		ExpectedStatus: expected,
	}

	client := &http.Client{
		Timeout: timeout,
		// Do not follow into bodies; we only need status. Redirects are fine.
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, h.URL, nil)
	if err != nil {
		r.Error = err.Error()
		return r
	}

	resp, err := client.Do(req)
	if err != nil {
		r.Error = err.Error()
		return r
	}
	// Close without reading the body.
	_ = resp.Body.Close()

	r.GotStatus = resp.StatusCode
	r.OK = resp.StatusCode == expected
	return r
}
