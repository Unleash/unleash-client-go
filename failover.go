package unleash

import (
	"time"
)

type failEvent interface {
	OccurredAt() time.Time
	Message() string
}

type baseFailEvent struct {
	occurredAt time.Time
	message    string
}

func (b *baseFailEvent) OccurredAt() time.Time { return b.occurredAt }
func (b *baseFailEvent) Message() string       { return b.message }

type networkErrorEvent struct {
	baseFailEvent
	err error
}

type httpStatusErrorEvent struct {
	baseFailEvent
	statusCode int
}

type failoverRequest struct {
	baseFailEvent
}

type failoverStrategy struct {
	failures  []failEvent
	maxFails  int
	relaxTime time.Duration
}

func newFailoverStrategy(maxFails int, relaxTime time.Duration) *failoverStrategy {
	return &failoverStrategy{
		failures:  make([]failEvent, 0, maxFails),
		maxFails:  maxFails,
		relaxTime: relaxTime,
	}
}

func (f *failoverStrategy) shouldFailover(event failEvent, now time.Time) bool {
	f.pruneOldFailures(now)

	switch e := event.(type) {
	case *networkErrorEvent:
		return f.hasTooManyFails(event)
	case *httpStatusErrorEvent:
		switch e.statusCode {
		case 401, 403, 404, 429, 501:
			return true
		case 408, 500, 502, 503, 504:
			return f.hasTooManyFails(event)
		default:
			return false
		}
	case *failoverRequest:
		return true
	default:
		return false
	}
}

func (f *failoverStrategy) pruneOldFailures(now time.Time) {
	cutoff := now.Add(-f.relaxTime)

	write := 0
	for _, fail := range f.failures {
		if !fail.OccurredAt().Before(cutoff) {
			f.failures[write] = fail
			write++
		}
	}
	f.failures = f.failures[:write]
}

func (f *failoverStrategy) hasTooManyFails(event failEvent) bool {
	f.failures = append(f.failures, event)
	return len(f.failures) >= f.maxFails
}
