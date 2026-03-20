package service

import (
	"context"
	"errors"
	"pine-ai/global/constant"
	"strings"
	"time"
)

var MockService *mockService

type mockService struct{}

func init() {
	MockService = &mockService{}
}

func (m *mockService) Infer(ctx context.Context, model string, chanStream chan string) error {
	mode := strings.ToLower(model)
	if strings.Contains(mode, constant.BackendTypeMockTimeout) {
		time.Sleep(3 * time.Second)
		chanStream <- "<!error>:timeout"
		return errors.New("mock timeout")
	}
	if strings.Contains(mode, constant.BackendTypeMockNoResponse) {
		<-ctx.Done()
		chanStream <- "<!error>:no_response"
		return errors.New("mock no response")
	}
	go func() {
		defer close(chanStream)
		text := "mock service streaming response"
		tokens := strings.Fields(text)
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
