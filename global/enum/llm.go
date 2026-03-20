package enum

import "github.com/sashabaranov/go-openai"

type BackendType string

const (
	BackendTypeOpenAI BackendType = "openai"
	BackendTypeOllama BackendType = "ollama"
	BackendTypeQwen   BackendType = "qwen"
	BackendTypeMock   BackendType = "mock"
)

type Version string

const (
	GPT3Dot5Turbo      Version = openai.GPT3Dot5Turbo
	GPT4               Version = openai.GPT4
	VersionMockTimeout Version = "mock_timeout"
)
