package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"pine-ai/persistence"
)

type RegisterModelRequest struct {
	ModelName     string `json:"model_name" binding:"required"`
	Version       string `json:"version" binding:"required"`
	BackendType   string `json:"backend_type" binding:"required"`
	Simulate      bool   `json:"simulate"`
	UpstreamModel string `json:"upstream_model"`
}

type UpdateModelRequest struct {
	BackendType   string `json:"backend_type" binding:"required"`
	Simulate      bool   `json:"simulate"`
	UpstreamModel string `json:"upstream_model"`
}

type InferRequest struct {
	Model   string `json:"model" binding:"required"`
	Version string `json:"version" binding:"required"`
	Input   string `json:"input" binding:"required"`
}

type ModelVersionView struct {
	ModelName     string    `json:"model_name"`
	Version       string    `json:"version"`
	BackendType   string    `json:"backend_type"`
	Simulate      bool      `json:"simulate"`
	UpstreamModel string    `json:"upstream_model"`
	Available     bool      `json:"available"`
	InUse         bool      `json:"in_use"`
	InUseCount    int64     `json:"in_use_count"`
	Deleted       bool      `json:"deleted"`
	State         string    `json:"state"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type runtimeSnapshot struct {
	id            string
	backendType   string
	simulate      bool
	upstreamModel string
	inFlight      int64
}

func (s *runtimeSnapshot) ID() string {
	return s.id
}

func (s *runtimeSnapshot) BackendType() string {
	return s.backendType
}

func (s *runtimeSnapshot) Simulate() bool {
	return s.simulate
}

func (s *runtimeSnapshot) UpstreamModel() string {
	return s.upstreamModel
}

type Registry struct {
	runtimeMu sync.RWMutex
	runtimes  map[string]map[string]*runtimeSnapshot
	store     *persistence.Store
}

func NewRegistry() *Registry {
	return &Registry{
		runtimes: make(map[string]map[string]*runtimeSnapshot),
		store:    persistence.Persistence,
	}
}

func (r *Registry) Register(req RegisterModelRequest) error {
	snap := &runtimeSnapshot{
		id:            fmt.Sprintf("%s-%s-%d", req.ModelName, req.Version, time.Now().UnixNano()),
		backendType:   req.BackendType,
		simulate:      req.Simulate,
		upstreamModel: req.UpstreamModel,
	}

	rec := persistence.ModelRecord{
		ModelName:     req.ModelName,
		Version:       req.Version,
		BackendType:   req.BackendType,
		IsMock:        req.Simulate,
		UpstreamModel: req.UpstreamModel,
		Available:     true,
		Deleted:       false,
		State:         "ready",
	}
	if err := r.store.CreateModel(context.Background(), rec); err != nil {
		if errors.Is(err, persistence.ErrModelVersionExists) {
			return fmt.Errorf("model %s version %s already exists", req.ModelName, req.Version)
		}
		return err
	}

	r.runtimeMu.Lock()
	defer r.runtimeMu.Unlock()
	if _, ok := r.runtimes[req.ModelName]; !ok {
		r.runtimes[req.ModelName] = make(map[string]*runtimeSnapshot)
	}
	r.runtimes[req.ModelName][req.Version] = snap
	return nil
}

func (r *Registry) Update(name, version string, req UpdateModelRequest) error {
	rec, err := r.store.GetModel(context.Background(), name, version)
	if err != nil {
		if errors.Is(err, persistence.ErrModelNotFound) || errors.Is(err, persistence.ErrModelVersionNotFound) {
			return errors.New("model version not found")
		}
		return err
	}
	if rec.Deleted {
		return errors.New("model version deleted")
	}
	// Hot update strategy:
	// 1) Build a new runtime snapshot first.
	// 2) Atomically replace pointer for new requests.
	// 3) Existing in-flight requests keep old snapshot and finish safely.
	newSnap := &runtimeSnapshot{
		id:            fmt.Sprintf("%s-%s-%d", name, version, time.Now().UnixNano()),
		backendType:   req.BackendType,
		simulate:      req.Simulate,
		upstreamModel: req.UpstreamModel,
	}

	r.runtimeMu.Lock()
	if _, ok := r.runtimes[name]; !ok {
		r.runtimes[name] = make(map[string]*runtimeSnapshot)
	}
	r.runtimes[name][version] = newSnap
	r.runtimeMu.Unlock()

	rec.BackendType = req.BackendType
	rec.IsMock = req.Simulate
	rec.UpstreamModel = req.UpstreamModel
	rec.Available = true
	rec.State = "ready"
	return r.store.UpdateModel(context.Background(), rec)
}

func (r *Registry) List() []ModelVersionView {
	records := r.store.ListModels(context.Background())
	out := make([]ModelVersionView, 0)
	for _, rec := range records {
		snap := r.getRuntime(rec.ModelName, rec.Version)
		var inUseCount int64
		if snap != nil {
			inUseCount = atomic.LoadInt64(&snap.inFlight)
		}
		view := ModelVersionView{
			ModelName:     rec.ModelName,
			Version:       rec.Version,
			BackendType:   rec.BackendType,
			Simulate:      rec.IsMock,
			UpstreamModel: rec.UpstreamModel,
			Available:     rec.Available,
			Deleted:       rec.Deleted,
			State:         rec.State,
			UpdatedAt:     rec.UpdatedAt,
			InUseCount:    inUseCount,
			InUse:         inUseCount > 0,
		}
		out = append(out, view)
	}
	return out
}

func (r *Registry) AcquireForInfer(name, version string) (*runtimeSnapshot, func(), error) {
	rec, err := r.store.GetModel(context.Background(), name, version)
	if err != nil {
		if errors.Is(err, persistence.ErrModelNotFound) || errors.Is(err, persistence.ErrModelVersionNotFound) {
			return nil, nil, errors.New("model version not found")
		}
		return nil, nil, err
	}
	if rec.Deleted {
		return nil, nil, errors.New("model version deleted")
	}
	if !rec.Available {
		return nil, nil, errors.New("model version unavailable")
	}

	snap := r.getRuntime(name, version)
	if snap == nil {
		return nil, nil, errors.New("model runtime unavailable")
	}
	atomic.AddInt64(&snap.inFlight, 1)
	release := func() {
		atomic.AddInt64(&snap.inFlight, -1)
	}
	return snap, release, nil
}

func (r *Registry) getRuntime(name, version string) *runtimeSnapshot {
	r.runtimeMu.RLock()
	defer r.runtimeMu.RUnlock()
	versions, ok := r.runtimes[name]
	if !ok {
		return nil
	}
	snap, ok := versions[version]
	if !ok {
		return nil
	}
	return snap
}
