package service

import (
	"context"
	"errors"
	"pine-ai/global/enum"
	"strings"
	"time"
)

var MockService *mockService

type mockService struct{}

func init() {
	MockService = &mockService{}
}

const (
	MockResponse = "this is a very long mock response, this is a very long mock response, this is a very long mock response, this is a very long mock response, this is a very long mock response"
)

func (m *mockService) Infer(ctx context.Context, version string, chanStream chan string) error {
	if strings.Contains(version, string(enum.VersionMockTimeout)) {
		time.Sleep(3 * time.Second)
		chanStream <- "<!error>:timeout"
		return errors.New("mock timeout")
	}
	go func() {
		defer close(chanStream)
		tokens := strings.Fields(MockResponse)
		if len(tokens) == 0 {
			tokens = []string{"..."}
		}

		for _, tk := range tokens {
			select {
			case <-ctx.Done():
				chanStream <- "<!finish>"
				return
			default:
			}
			chanStream <- tk
			time.Sleep(200 * time.Millisecond)
		}
		chanStream <- "<!finish>"
	}()
	return nil
}
