package dto

import "pine-ai/global/enum"

type RegisterModelRequest struct {
	ModelName   string `json:"model_name" binding:"required"`
	Version     string `json:"version" binding:"required"`
	BackendType string `json:"backend_type" binding:"required"`
	Concurrency int    `json:"concurrency"`
	Weight      int    `json:"weight"`
}

type UpdateModelRequest struct {
	ModelName   string           `json:"model_name" binding:"required"`
	Version     string           `json:"version" binding:"required"`
	BackendType enum.BackendType `json:"backend_type" binding:"required"`
	Concurrency int              `json:"concurrency"`
	Weight      int              `json:"weight"`
}

type InferRequest struct {
	Model   string `json:"model" binding:"required"`
	Version string `json:"version" binding:"required"`
	Input   string `json:"input" binding:"required"`
}
