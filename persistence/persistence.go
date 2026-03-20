package persistence

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrModelVersionExists   = errors.New("model version already exists")
	ErrModelNotFound        = errors.New("model not found")
	ErrModelVersionNotFound = errors.New("model version not found")
)

type BackendType string
type State string

const (
	BackendTypeOpenAI BackendType = "openai"
	StateReady        State       = "ready"
	StateDeleted      State       = "deleted"
	StateUnavailable  State       = "unavailable"
)

type ModelRecord struct {
	ModelName     string      `json:"model_name"`
	Version       string      `json:"version"`
	BackendType   BackendType `json:"backend_type"`
	UpstreamModel string      `json:"upstream_model"`
	Concurrency   int         `json:"concurrency"`
	Weight        int         `json:"weight"`
	Deleted       bool        `json:"deleted"`
	State         State       `json:"state"`
	LastUsedAt    time.Time   `json:"last_used_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type store struct {
	mu     sync.RWMutex
	models map[string]map[string]*ModelRecord
}

var Store *store

func init() {
	Store = &store{
		models: make(map[string]map[string]*ModelRecord),
	}
}

func (p *store) CreateModel(ctx context.Context, rec *ModelRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.models[rec.ModelName]; !ok {
		p.models[rec.ModelName] = make(map[string]*ModelRecord)
	}
	if _, exists := p.models[rec.ModelName][rec.Version]; exists {
		return ErrModelVersionExists
	}
	rec.UpdatedAt = time.Now()
	rec.LastUsedAt = time.Now()
	p.models[rec.ModelName][rec.Version] = rec
	return nil
}

func (p *store) UpdateModel(ctx context.Context, rec *ModelRecord) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	versions, ok := p.models[rec.ModelName]
	if !ok {
		return ErrModelNotFound
	}
	if _, ok := versions[rec.Version]; !ok {
		return ErrModelVersionNotFound
	}
	rec.UpdatedAt = time.Now()
	versions[rec.Version] = rec
	return nil
}

func (p *store) GetModel(ctx context.Context, modelName, version string) (*ModelRecord, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	versions, ok := p.models[modelName]
	if !ok {
		return nil, ErrModelNotFound
	}
	rec, ok := versions[version]
	if !ok {
		return nil, ErrModelVersionNotFound
	}
	return rec, nil
}

func (p *store) ListModels(ctx context.Context) []*ModelRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]*ModelRecord, 0)
	for _, versions := range p.models {
		for _, rec := range versions {
			out = append(out, rec)
		}
	}
	return out
}

func (p *store) UpdateLastUsedAt(ctx context.Context, modelName, version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	versions, ok := p.models[modelName]
	if !ok {
		return ErrModelNotFound
	}
	if _, ok := versions[version]; !ok {
		return ErrModelVersionNotFound
	}
	versions[version].LastUsedAt = time.Now()
	return nil
}
