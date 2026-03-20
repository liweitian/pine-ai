package dto

type RegisterModelRequest struct {
	ModelName     string `json:"model_name" binding:"required"`
	Version       string `json:"version" binding:"required"`
	BackendType   string `json:"backend_type" binding:"required"`
	Simulate      bool   `json:"simulate"`
	UpstreamModel string `json:"upstream_model"`
}

type UpdateModelRequest struct {
	ModelName     string `json:"model_name" binding:"required"`
	Version       string `json:"version" binding:"required"`
	BackendType   string `json:"backend_type" binding:"required"`
	Simulate      bool   `json:"simulate"`
	UpstreamModel string `json:"upstream_model"`
}

type InferRequest struct {
	Model   string `json:"model" binding:"required"`
	Version string `json:"version" binding:"required"`
	Input   string `json:"input" binding:"required"`
}
