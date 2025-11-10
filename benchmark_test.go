package unleash_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5"
	"github.com/Unleash/unleash-go-sdk/v5/api"
)

type mockStorage map[string]api.Feature

func (m mockStorage) Get(in string) (any, bool) {
	out, found := m[in]
	return out, found
}

func (m mockStorage) List() []any {
	res := make([]any, 0, len(m))
	for _, feature := range m {
		res = append(res, feature)
	}
	return res

}

func (m mockStorage) Init(backupPath string, appName string)        {}
func (m mockStorage) Load() error                                   { return nil }
func (m mockStorage) Persist() error                                { return nil }
func (m mockStorage) Reset(data map[string]any, persist bool) error { return nil }

func ptr[T any](v T) *T {
	return &v
}

func BenchmarkFeatureToggleEvaluation(b *testing.B) {
	err := unleash.Initialize(
		unleash.WithListener(&unleash.NoopListener{}),
		unleash.WithAppName("go-benchmark"),
		unleash.WithUrl("https://app.unleash-hosted.com/demo/api/"),
		unleash.WithCustomHeaders(http.Header{"Authorization": {"Go-Benchmark:development.be6b5d318c8e77469efb58590022bb6416100261accf95a15046c04d"}}),
		unleash.WithStorage(mockStorage{
			"foo": api.Feature{
				Name:    "foo",
				Enabled: true,
				Dependencies: &[]api.Dependency{
					{Feature: "bar", Enabled: ptr(true)},
				},
			},
			"bar": api.Feature{
				Name:    "bar",
				Enabled: true,
			},
		}),
	)
	if err != nil {
		b.Fatal(err)
	}

	startTime := time.Now()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = unleash.IsEnabled("foo")
	}

	endTime := time.Now()

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
