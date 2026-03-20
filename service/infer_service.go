package service

import (
	"context"
	"errors"
	"io"
	client "pine-ai/client/llm"
	"strings"
	"time"

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

type inferService struct {
	// cfg       config.Config
	// registry  *Registry
	// openaiCli *openai.Client
}

var InferService *inferService

func init() {
	InferService = &inferService{
		// cfg:      cfg,
		// registry: registry,
	}
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
	snap, release, err := ModelRegistry.AcquireForInfer(model, version)
	if err != nil {
		return err
	}
	defer release()
	if snap.Simulate() {
		return streamSimulated(ctx, input, onToken)
	}
	backend := snap.BackendType()
	switch backend {
	case "openai":
		return s.streamOpenAI(ctx, snap, input, onToken)
	default:
		return streamSimulated(ctx, input, onToken)
	}
}

func (s *inferService) streamOpenAI(
	ctx context.Context,
	snap *runtimeSnapshot,
	input string,
	onToken func(token string) error,
) error {
	modelName := snap.UpstreamModel()

	req := openai.ChatCompletionRequest{
		Model:       modelName,
		Stream:      true,
		Messages:    []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: input}},
		Temperature: 1.0,
	}

	stream, err := client.OpenAI.CreateChatCompletionStream(ctx, req)
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
