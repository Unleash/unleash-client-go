package unleash

import (
	"fmt"
	"testing"
	"time"
)

func TestFailoverStrategy_ServerHintPollingFailsOver(t *testing.T) {
	strategy := newFailoverStrategy(3, 10*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	event := &failoverRequest{
		baseFailEvent: baseFailEvent{
			occurredAt: now,
			message:    "Unleash has requested failover to polling",
		},
	}

	if got := strategy.shouldFailover(event, now); !got {
		t.Fatalf("expected failover on server polling hint")
	}
}

func TestFailoverStrategy_HardHttpStatusCodesFailoverImmediately(t *testing.T) {
	strategy := newFailoverStrategy(3, 10*time.Second)
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for _, statusCode := range []int{401, 403, 404, 429} {
		event := &httpStatusErrorEvent{
			baseFailEvent: baseFailEvent{
				occurredAt: now,
				message:    fmt.Sprintf("HTTP status error with status code %d", statusCode),
			},
			statusCode: statusCode,
		}

		if got := strategy.shouldFailover(event, now); !got {
			t.Fatalf("expected failover on hard status code %d", statusCode)
		}
	}
}

func TestFailoverStrategy_SoftErrorsAccumulateWithinRelaxWindow(t *testing.T) {
	strategy := newFailoverStrategy(3, 60*time.Second)
	base := time.Date(1867, 11, 7, 0, 0, 0, 0, time.UTC)

	mkEvent := func(offsetMs int) *httpStatusErrorEvent {
		occurredAt := base.Add(time.Duration(offsetMs) * time.Millisecond)
		return &httpStatusErrorEvent{
			baseFailEvent: baseFailEvent{
				occurredAt: occurredAt,
				message:    "HTTP status error with status code 503",
			},
			statusCode: 503,
		}
	}

	if got := strategy.shouldFailover(mkEvent(0), base); got {
		t.Fatalf("expected no failover on first soft error")
	}

	if got := strategy.shouldFailover(mkEvent(1000), base.Add(1000*time.Millisecond)); got {
		t.Fatalf("expected no failover on second soft error")
	}

	if got := strategy.shouldFailover(mkEvent(2000), base.Add(2000*time.Millisecond)); !got {
		t.Fatalf("expected failover on third soft error within relax window")
	}
}

func TestFailoverStrategy_SoftErrorsPrunedByWindow(t *testing.T) {
	strategy := newFailoverStrategy(3, 1*time.Second)
	base := time.Date(1867, 11, 7, 0, 0, 0, 0, time.UTC)

	mkEvent := func(offsetMs int) *httpStatusErrorEvent {
		occurredAt := base.Add(time.Duration(offsetMs) * time.Millisecond)
		return &httpStatusErrorEvent{
			baseFailEvent: baseFailEvent{
				occurredAt: occurredAt,
				message:    "HTTP status error with status code 502",
			},
			statusCode: 502,
		}
	}

	if got := strategy.shouldFailover(mkEvent(0), base); got {
		t.Fatalf("expected no failover on first soft error")
	}

	if got := strategy.shouldFailover(mkEvent(2000), base.Add(2000*time.Millisecond)); got {
		t.Fatalf("expected no failover on second soft error after window")
	}

	if got := strategy.shouldFailover(mkEvent(3000), base.Add(3000*time.Millisecond)); got {
		t.Fatalf("expected no failover on third soft error after pruning")
	}
}

func TestFailoverStrategy_UnhandledStatusCodesNeverFailover(t *testing.T) {
	strategy := newFailoverStrategy(1, 10*time.Second)
	base := time.Date(1867, 11, 7, 0, 0, 0, 0, time.UTC)

	mkEvent := func(code int, offsetMs int) *httpStatusErrorEvent {
		occurredAt := base.Add(time.Duration(offsetMs) * time.Millisecond)
		return &httpStatusErrorEvent{
			baseFailEvent: baseFailEvent{
				occurredAt: occurredAt,
				message:    fmt.Sprintf("HTTP status error with status code %d", code),
			},
			statusCode: code,
		}
	}

	for _, statusCode := range []int{418, 418, 418, 418} {
		if got := strategy.shouldFailover(mkEvent(statusCode, 0), base); got {
			t.Fatalf("expected no failover on unhandled status code %d", statusCode)
		}
	}
}
