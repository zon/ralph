package comment

import (
	"io"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/output"
)

type mockAIClient struct {
	runAgentFunc  func(string) error
	runAgentCalls []string
}

func (m *mockAIClient) RunAgent(prompt string) error {
	m.runAgentCalls = append(m.runAgentCalls, prompt)
	if m.runAgentFunc != nil {
		return m.runAgentFunc(prompt)
	}
	return nil
}

type mockServicesClient struct {
	startFunc   func([]config.Service) error
	startCalled bool
	stopCalled  bool
	services    []config.Service
}

func (m *mockServicesClient) Start(services []config.Service) error {
	m.startCalled = true
	m.services = services
	if m.startFunc != nil {
		return m.startFunc(services)
	}
	return nil
}

func (m *mockServicesClient) Stop() {
	m.stopCalled = true
}

type commentOption func(*CommentCmd)

func withAI(ac AIClient) commentOption {
	return func(c *CommentCmd) { c.ai = ac }
}

func withServices(sc ServicesClient) commentOption {
	return func(c *CommentCmd) { c.services = sc }
}

func withMocks(opts ...commentOption) *CommentCmd {
	c := &CommentCmd{
		ai:       &mockAIClient{},
		services: &mockServicesClient{},
		out:      output.NewClient(io.Discard, io.Discard, false),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func aiRunAgentCalls(cmd *CommentCmd) []string {
	if m, ok := cmd.ai.(*mockAIClient); ok {
		return m.runAgentCalls
	}
	return nil
}

func servicesStartCalled(cmd *CommentCmd) bool {
	if m, ok := cmd.services.(*mockServicesClient); ok {
		return m.startCalled
	}
	return false
}

func servicesStopCalled(cmd *CommentCmd) bool {
	if m, ok := cmd.services.(*mockServicesClient); ok {
		return m.stopCalled
	}
	return false
}

var errMock = &mockError{"mock error"}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
