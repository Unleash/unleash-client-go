package unleash_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
)

type NoOpListener struct{}

func (l *NoOpListener) OnReady()                                {}
func (l *NoOpListener) OnError(err error)                       {}
func (l *NoOpListener) OnWarning(warning error)                 {}
func (l *NoOpListener) OnCount(name string, enabled bool)       {}
func (l *NoOpListener) OnSent(payload unleash.MetricsData)      {}
func (l *NoOpListener) OnRegistered(payload unleash.ClientData) {}

func BenchmarkFeatureToggleEvaluation(b *testing.B) {
	unleash.Initialize(
		unleash.WithListener(&NoOpListener{}),
		unleash.WithAppName("go-benchmark"),
		unleash.WithUrl("https://app.unleash-hosted.com/demo/api/"),
		unleash.WithCustomHeaders(http.Header{"Authorization": {"Go-Benchmark:development.be6b5d318c8e77469efb58590022bb6416100261accf95a15046c04d"}}),
	)

	b.ResetTimer()
	startTime := time.Now()

	for i := 0; i < b.N; i++ {
		_ = unleash.IsEnabled("go-benchmark")
	}

	endTime := time.Now()
	b.StopTimer()

	// Calculate ns/op (nanoseconds per operation)
	nsPerOp := float64(endTime.Sub(startTime).Nanoseconds()) / float64(b.N)

	// Calculate operations per day
	opsPerSec := 1e9 / nsPerOp
	opsPerDay := opsPerSec * 60 * 60 * 24

	if b.N > 1000000 { // Only print if the number of iterations is large enough for a stable result
		opsPerDayBillions := opsPerDay / 1e9 // Convert to billions
		fmt.Printf("Final Estimated Operations Per Day: %.3f billion (%e)\n", opsPerDayBillions, opsPerDay)
	}
}
