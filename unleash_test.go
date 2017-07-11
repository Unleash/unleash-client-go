package unleash

import (
	"github.com/stretchr/testify/mock"
)

type MockedListener struct {
	mock.Mock
}

func (l *MockedListener) OnError(err error) {
	l.Called(err)
}

func (l *MockedListener) OnWarning(warning error) {
	l.Called(warning)
}

func (l *MockedListener) OnReady() {
	l.Called()
}

func (l *MockedListener) OnCount(name string, enabled bool) {
	l.Called(name, enabled)
}

func (l *MockedListener) OnSent(payload MetricsData) {
	l.Called(payload)
}

func (l *MockedListener) OnRegistered(payload ClientData) {
	l.Called(payload)
}
