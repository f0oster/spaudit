package mocks

import (
	"github.com/stretchr/testify/mock"

	"spaudit/domain/events"
)

// MockJobEventPublisher is a mock implementation of JobEventPublisher for testing
type MockJobEventPublisher struct {
	mock.Mock
}

func (m *MockJobEventPublisher) PublishJobCompleted(event events.JobCompletedEvent) {
	m.Called(event)
}

func (m *MockJobEventPublisher) PublishJobFailed(event events.JobFailedEvent) {
	m.Called(event)
}

func (m *MockJobEventPublisher) PublishJobCancelled(event events.JobCancelledEvent) {
	m.Called(event)
}

func (m *MockJobEventPublisher) PublishSiteAuditCompleted(event events.SiteAuditCompletedEvent) {
	m.Called(event)
}
