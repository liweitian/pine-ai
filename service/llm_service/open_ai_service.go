package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	client "pine-ai/client/llm"

	"strings"

	"github.com/sashabaranov/go-openai"
)

var OpenAIService *openAIService

type openAIService struct {
}

const (
	ChatTemperature  = 1.0
	MaxTokenLimit    = 512
	chanStreamError  = "<!error>"
	chanStreamFinish = "<!finish>"
)

var (
	defaultSystemMessage = openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "You are a helpful assistant.",
	}
)

func init() {
	OpenAIService = &openAIService{}
}

// 以防模型传入有问题 model字段实际并未使用
func (o *openAIService) Infer(ctx context.Context, model string, chanStream chan string) error {
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4o,
		MaxTokens:   MaxTokenLimit,
		Temperature: ChatTemperature,
		Messages:    []openai.ChatCompletionMessage{defaultSystemMessage},
		Stream:      true,
	}
	stream, err := client.OpenAI.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return err
	}

	go func() {
		defer stream.Close()
		defer close(chanStream)
		var sb strings.Builder
		for {

			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				chanStream <- chanStreamFinish
				return
			}

			if err != nil {
				chanStream <- chanStreamError
				return
			}
			if len(response.Choices) == 0 {
				chanStream <- chanStreamError
				return
			}
			sb.WriteString(response.Choices[0].Delta.Content)
			data, _ := json.Marshal(response.Choices[0])
			chanStream <- string(data)
		}
	}()
	return nil
}
