package service

import (
	"context"
	"errors"
	"pine-ai/global/enum"
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
	case enum.BackendTypeOpenAI:
		provider = service.OpenAIService
	case enum.BackendTypeOllama:
		provider = service.OllamaService
	case enum.BackendTypeQwen:
		provider = service.QwenService
	case enum.BackendTypeMock:
		provider = service.MockService
	default:
		return errors.New("unknown backend type: " + string(backend))
	}
	asyncCtx := context.Background()
	if err := provider.Infer(asyncCtx, snap.Version(), chanStream); err != nil {
		return err
	}

	for {
		select {
		case <-asyncCtx.Done():
			return asyncCtx.Err()
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
