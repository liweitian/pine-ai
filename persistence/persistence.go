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

type ModelRecord struct {
	ModelName     string    `json:"model_name"`
	Version       string    `json:"version"`
	BackendType   string    `json:"backend_type"`
	IsMock        bool      `json:"is_mock"`
	UpstreamModel string    `json:"upstream_model"`
	Available     bool      `json:"available"`
	Deleted       bool      `json:"deleted"`
	State         string    `json:"state"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Store struct {
	mu     sync.RWMutex
	models map[string]map[string]ModelRecord
}

var Persistence = New()

func New() *Store {
	return &Store{
		models: make(map[string]map[string]ModelRecord),
	}
}

func (p *Store) CreateModel(ctx context.Context, rec ModelRecord) error {
	_ = ctx
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.models[rec.ModelName]; !ok {
		p.models[rec.ModelName] = make(map[string]ModelRecord)
	}
	if _, exists := p.models[rec.ModelName][rec.Version]; exists {
		return ErrModelVersionExists
	}
	rec.UpdatedAt = time.Now()
	p.models[rec.ModelName][rec.Version] = rec
	return nil
}

func (p *Store) UpdateModel(ctx context.Context, rec ModelRecord) error {
	_ = ctx
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

func (p *Store) GetModel(ctx context.Context, modelName, version string) (ModelRecord, error) {
	_ = ctx
	p.mu.RLock()
	defer p.mu.RUnlock()
	versions, ok := p.models[modelName]
	if !ok {
		return ModelRecord{}, ErrModelNotFound
	}
	rec, ok := versions[version]
	if !ok {
		return ModelRecord{}, ErrModelVersionNotFound
	}
	return rec, nil
}

func (p *Store) ListModels(ctx context.Context) []ModelRecord {
	_ = ctx
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]ModelRecord, 0)
	for _, versions := range p.models {
		for _, rec := range versions {
			out = append(out, rec)
		}
	}
	return out
}

func (p *Store) Save(ctx context.Context, key string, value any) error {
	_ = ctx
	_ = key
	_ = value
	return nil
}

func (p *Store) Get(ctx context.Context, key string) (any, error) {
	_ = ctx
	_ = key
	return nil, nil
}

func (p *Store) Delete(ctx context.Context, key string) error {
	_ = ctx
	_ = key
	return nil
}
