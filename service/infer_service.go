package service

import (
	"context"
	"errors"
	"pine-ai/global/constant"
	"pine-ai/persistence"
	service "pine-ai/service/llm_service"
	"strings"
	"time"
)

type InferProvider interface {
	Infer(ctx context.Context, model string, chanStream chan string) error
}

type inferService struct {
}

var InferService *inferService

func init() {
	InferService = &inferService{}
	releaseIdleModel()
}

func releaseIdleModel() {
	records := persistence.Store.ListModels(context.Background())
	for _, rec := range records {
		if time.Since(rec.LastUsedAt) > 1*time.Hour {
			rec.Deleted = true
			persistence.Store.UpdateModel(context.Background(), rec)
		}
	}
	defer time.AfterFunc(time.Hour, func() {
		releaseIdleModel()
	})
}

// StreamInfer streams tokens to onToken and emits onMeta once at the beginning.
// It does not deal with SSE formatting; handlers should convert tokens to SSE frames.
func (s *inferService) StreamInfer(
	ctx context.Context,
	model string,
	version string,
	input string,
	onToken func(token string) error,
) error {
	persistence.Store.UpdateLastUsedAt(ctx, model, version)
	snap, release, err := ModelRegistry.AcquireForInfer(model, version)
	if err != nil {
		return err
	}
	defer release()
	backend := snap.BackendType()
	chanStream := make(chan string, 32)
	var provider InferProvider
	switch backend {
	case constant.BackendTypeOpenAI:
		provider = service.OpenAIService
	case constant.BackendTypeOllama:
		provider = service.OllamaService
	case constant.BackendTypeQwen:
		provider = service.QwenService
	case constant.BackendTypeMock:
		provider = service.MockService
	case constant.BackendTypeMockTimeout:
		provider = service.MockService
	case constant.BackendTypeMockNoResponse:
		provider = service.MockService
	default:
		return errors.New("unknown backend type: " + backend)
	}

	if err := provider.Infer(ctx, snap.UpstreamModel(), chanStream); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-chanStream:
			if !ok {
				return nil
			}
			switch msg {
			case "<!finish>":
				return nil
			case "<!error>":
				return errors.New("infer stream backend error")
			default:
				if strings.HasPrefix(msg, "<!error>:") {
					return errors.New(strings.TrimPrefix(msg, "<!error>:"))
				}
				if err := onToken(msg); err != nil {
					return err
				}
			}
		}
	}
}
