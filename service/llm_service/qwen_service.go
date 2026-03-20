package service

import (
	"context"
	"strings"
	"time"
)

var QwenService *qwenService

type qwenService struct{}

func init() {
	QwenService = &qwenService{}
}

func (m *qwenService) Infer(ctx context.Context, _ string, chanStream chan string) error {
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
