package client

import (
	"os"

	"github.com/sashabaranov/go-openai"
)

var OpenAI *openai.Client

func init() {
	initOpenAIClient()

}

func initOpenAIClient() {
	authToken := os.Getenv("OPENAI_API_KEY")
	c := openai.NewClient(authToken)
	OpenAI = c
}
