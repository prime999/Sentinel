package checker

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sentinel-monitoring/sentinel/internal/models"
)

func TestIsTimeoutErr(t *testing.T) {
	if !isTimeoutErr(context.DeadlineExceeded) {
		t.Fatal("DeadlineExceeded should be timeout")
	}
	if !isTimeoutErr(fmt.Errorf("Get %q: %w", "https://x", context.DeadlineExceeded)) {
		t.Fatal("wrapped deadline should be timeout")
	}
	if !isTimeoutErr(errors.New("Client.Timeout exceeded while awaiting headers")) {
		t.Fatal("client timeout string should match")
	}
	if isTimeoutErr(errors.New("connection refused")) {
		t.Fatal("connection refused is not a timeout")
	}
}

func TestApplyHTTPProbeErrorTimeoutReachedIsUp(t *testing.T) {
	result := &models.CheckResult{Status: models.StatusDown}
	applyHTTPProbeError(result, context.DeadlineExceeded, true)
	if result.Status != models.StatusUp {
		t.Fatalf("status=%s want up", result.Status)
	}
	if result.Error != "" {
		t.Fatalf("error=%q want empty", result.Error)
	}
}

func TestApplyHTTPProbeErrorTimeoutUnreachableIsDown(t *testing.T) {
	result := &models.CheckResult{Status: models.StatusDown}
	applyHTTPProbeError(result, context.DeadlineExceeded, false)
	if result.Status != models.StatusDown {
		t.Fatalf("status=%s want down", result.Status)
	}
	if !strings.Contains(result.Error, "timed out") {
		t.Fatalf("error=%q", result.Error)
	}
}

func TestApplyHTTPProbeErrorHardFailureIsDown(t *testing.T) {
	result := &models.CheckResult{Status: models.StatusDown}
	applyHTTPProbeError(result, errors.New("dial tcp: connection refused"), true)
	if result.Status != models.StatusDown {
		t.Fatalf("status=%s want down", result.Status)
	}
	if result.Error == "" {
		t.Fatal("expected error")
	}
}

func TestTimingInfoReachedHost(t *testing.T) {
	var empty timingInfo
	if empty.reachedHost() {
		t.Fatal("empty timings should not count as reached")
	}
	connected := timingInfo{connectDone: time.Now()}
	if !connected.reachedHost() {
		t.Fatal("connectDone should count as reached")
	}
}

func TestFormatHTTPProbeErrorStripsGetPrefix(t *testing.T) {
	err := fmt.Errorf(`Get "https://www.2hatslogic.com/": %w`, context.DeadlineExceeded)
	got := formatHTTPProbeError(err)
	if strings.Contains(got, "Get ") {
		t.Fatalf("got %q, still has Get prefix", got)
	}
	if !strings.Contains(got, "timed out") {
		t.Fatalf("got %q", got)
	}
}
