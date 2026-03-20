package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"pine-ai/config"

	"github.com/sashabaranov/go-openai"
)

type InferMeta struct {
	Model        string `json:"model"`
	Version      string `json:"version"`
	RuntimeID    string `json:"runtime_id"`
	BackendType  string `json:"backend_type"`
	Simulate     bool   `json:"simulate"`
	UpstreamName string `json:"upstream_model"`
}

type InferService struct {
	cfg       config.Config
	registry  *Registry
	openaiCli *openai.Client
}

func NewInferService(cfg config.Config, registry *Registry) *InferService {
	s := &InferService{
		cfg:      cfg,
		registry: registry,
	}
	if strings.TrimSpace(cfg.OpenAI.APIKey) != "" {
		s.openaiCli = openai.NewClient(cfg.OpenAI.APIKey)
	}
	return s
}

// StreamInfer streams tokens to onToken and emits onMeta once at the beginning.
// It does not deal with SSE formatting; handlers should convert tokens to SSE frames.
func (s *InferService) StreamInfer(
	ctx context.Context,
	model string,
	version string,
	input string,
	onMeta func(meta InferMeta) error,
	onToken func(token string) error,
) error {
	snap, release, err := s.registry.AcquireForInfer(model, version)
	if err != nil {
		return err
	}
	defer release()

	meta := InferMeta{
		Model:        model,
		Version:      version,
		RuntimeID:    snap.ID(),
		BackendType:  snap.BackendType(),
		Simulate:     snap.Simulate(),
		UpstreamName: snap.UpstreamModel(),
	}
	if onMeta != nil {
		if err := onMeta(meta); err != nil {
			return err
		}
	}

	// Decide backend.
	backend := snap.BackendType()
	useOpenAI := strings.EqualFold(backend, "openai") && !snap.Simulate() && s.openaiCli != nil
	if useOpenAI {
		return s.streamOpenAI(ctx, snap, input, onToken)
	}
	return streamSimulated(ctx, input, onToken)
}

func (s *InferService) streamOpenAI(
	ctx context.Context,
	snap *runtimeSnapshot,
	input string,
	onToken func(token string) error,
) error {
	modelName := snap.UpstreamModel()
	if strings.TrimSpace(modelName) == "" {
		modelName = s.cfg.OpenAI.ChatModel
	}

	req := openai.ChatCompletionRequest{
		Model:       modelName,
		Stream:      true,
		Messages:    []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: input}},
		Temperature: 1.0,
	}

	stream, err := s.openaiCli.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if len(response.Choices) == 0 {
			continue
		}
		token := response.Choices[0].Delta.Content
		if token == "" {
			continue
		}
		if err := onToken(token); err != nil {
			return err
		}
	}
}

func streamSimulated(ctx context.Context, input string, onToken func(token string) error) error {
	// Tokenize:
	// - if input contains spaces, use Fields()
	// - otherwise, split by rune as "token"
	// This gives stable streaming behavior for both English and Chinese.
	genText := "模拟回复：" + input
	tokens := tokenize(genText)
	if len(tokens) == 0 {
		tokens = []string{"..."}
	}

	for _, tk := range tokens {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := onToken(tk); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func tokenize(text string) []string {
	if strings.IndexFunc(text, func(r rune) bool { return r == ' ' || r == '\n' || r == '\t' || r == '\r' }) >= 0 {
		return strings.Fields(text)
	}
	out := make([]string, 0, len([]rune(text)))
	for _, r := range []rune(text) {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			continue
		}
		out = append(out, string(r))
	}
	return out
}
