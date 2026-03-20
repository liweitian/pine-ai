package service

import (
	"context"
	"errors"
	"fmt"
	"pine-ai/dto"
	"sync"
	"sync/atomic"
	"time"

	"pine-ai/persistence"
)

type ModelVersionView struct {
	ModelName     string    `json:"model_name"`
	Version       string    `json:"version"`
	BackendType   string    `json:"backend_type"`
	Simulate      bool      `json:"simulate"`
	UpstreamModel string    `json:"upstream_model"`
	InUseCount    int64     `json:"in_use_count"`
	Deleted       bool      `json:"deleted"`
	State         string    `json:"state"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type runtimeSnapshot struct {
	id            string
	backendType   string
	upstreamModel string
	concurrency   int
	weight        int
	inFlight      int64
}

func (s *runtimeSnapshot) ID() string          { return s.id }
func (s *runtimeSnapshot) BackendType() string { return s.backendType }
func (s *runtimeSnapshot) UpstreamModel() string {
	return s.upstreamModel
}

type modelStore interface {
	CreateModel(ctx context.Context, rec *persistence.ModelRecord) error
	UpdateModel(ctx context.Context, rec *persistence.ModelRecord) error
	GetModel(ctx context.Context, modelName, version string) (*persistence.ModelRecord, error)
	ListModels(ctx context.Context) []*persistence.ModelRecord
}

type Registry struct {
	runtimeMu sync.RWMutex
	runtimes  map[string]map[string]*runtimeSnapshot
	store     modelStore
}

var ModelRegistry *Registry

func init() {
	ModelRegistry = &Registry{
		runtimes: make(map[string]map[string]*runtimeSnapshot),
		store:    persistence.Store,
	}
}

func (r *Registry) Register(req dto.RegisterModelRequest) error {
	rec := persistence.ModelRecord{
		ModelName:     req.ModelName,
		Version:       req.Version,
		BackendType:   persistence.BackendType(req.BackendType),
		UpstreamModel: req.UpstreamModel,
		Concurrency:   req.Concurrency,
		Weight:        req.Weight,
		Deleted:       false,
		State:         persistence.StateReady,
	}

	if err := r.store.CreateModel(context.Background(), &rec); err != nil {
		return err
	}

	snap := &runtimeSnapshot{
		id:            fmt.Sprintf("%s-%s-%d", req.ModelName, req.Version, time.Now().UnixNano()),
		backendType:   req.BackendType,
		upstreamModel: req.UpstreamModel,
		concurrency:   req.Concurrency,
		weight:        req.Weight,
	}

	r.runtimeMu.Lock()
	defer r.runtimeMu.Unlock()
	if _, ok := r.runtimes[req.ModelName]; !ok {
		r.runtimes[req.ModelName] = make(map[string]*runtimeSnapshot)
	}
	r.runtimes[req.ModelName][req.Version] = snap
	return nil
}

func (r *Registry) Update(name, version string, req dto.UpdateModelRequest) error {
	rec, err := r.store.GetModel(context.Background(), name, version)
	if err != nil {
		return err
	}
	if rec.Deleted {
		return errors.New("model version deleted")
	}

	newSnap := &runtimeSnapshot{
		id:            fmt.Sprintf("%s-%s-%d", name, version, time.Now().UnixNano()),
		backendType:   req.BackendType,
		upstreamModel: req.UpstreamModel,
		concurrency:   req.Concurrency,
		weight:        req.Weight,
	}
	r.runtimeMu.Lock()
	if _, ok := r.runtimes[name]; !ok {
		r.runtimes[name] = make(map[string]*runtimeSnapshot)
	}
	r.runtimes[name][version] = newSnap
	r.runtimeMu.Unlock()

	rec.BackendType = persistence.BackendType(req.BackendType)
	rec.UpstreamModel = req.UpstreamModel
	rec.Concurrency = req.Concurrency
	rec.Weight = req.Weight
	rec.Deleted = false
	rec.State = persistence.StateReady
	return r.store.UpdateModel(context.Background(), rec)
}

func (r *Registry) Delete(name, version string) error {
	rec, err := r.store.GetModel(context.Background(), name, version)
	if err != nil {
		return err
	}
	if rec.Deleted {
		return errors.New("model version already deleted")
	}

	rec.Deleted = true
	rec.State = persistence.StateDeleted
	return r.store.UpdateModel(context.Background(), rec)
}

func (r *Registry) List() []ModelVersionView {
	records := r.store.ListModels(context.Background())
	out := make([]ModelVersionView, 0, len(records))

	for _, rec := range records {
		var inUseCount int64
		r.runtimeMu.RLock()
		if snap, ok := r.runtimes[rec.ModelName][rec.Version]; ok && snap != nil {
			inUseCount = atomic.LoadInt64(&snap.inFlight)
		}
		r.runtimeMu.RUnlock()

		out = append(out, ModelVersionView{
			ModelName:     rec.ModelName,
			Version:       rec.Version,
			BackendType:   string(rec.BackendType),
			UpstreamModel: rec.UpstreamModel,
			InUseCount:    inUseCount,
			Deleted:       rec.Deleted,
			State:         string(rec.State),
			UpdatedAt:     rec.UpdatedAt,
		})
	}
	return out
}

func (r *Registry) AcquireForInfer(name, version string) (*runtimeSnapshot, func(), error) {
	rec, err := r.store.GetModel(context.Background(), name, version)
	if err != nil {
		return nil, nil, err
	}
	if rec.Deleted {
		return nil, nil, errors.New("model version deleted")
	}

	r.runtimeMu.RLock()
	snap := r.runtimes[name][version]
	r.runtimeMu.RUnlock()

	if snap == nil {
		snap = &runtimeSnapshot{
			id:            fmt.Sprintf("%s-%s-%d", name, version, time.Now().UnixNano()),
			backendType:   string(rec.BackendType),
			upstreamModel: rec.UpstreamModel,
			concurrency:   rec.Concurrency,
			weight:        rec.Weight,
		}
		r.runtimeMu.Lock()
		if _, ok := r.runtimes[name]; !ok {
			r.runtimes[name] = make(map[string]*runtimeSnapshot)
		}
		if existing := r.runtimes[name][version]; existing != nil {
			snap = existing
		} else {
			r.runtimes[name][version] = snap
		}
		r.runtimeMu.Unlock()
	}

	if snap.concurrency > 0 {
		current := atomic.LoadInt64(&snap.inFlight)
		for {
			if int(current) >= snap.concurrency {
				return nil, nil, fmt.Errorf("model version concurrency exceeded: limit=%d", snap.concurrency)
			}
			if atomic.CompareAndSwapInt64(&snap.inFlight, current, current+1) {
				break
			}
			current = atomic.LoadInt64(&snap.inFlight)
		}
	} else {
		atomic.AddInt64(&snap.inFlight, 1)
	}

	release := func() {
		atomic.AddInt64(&snap.inFlight, -1)
	}
	return snap, release, nil
}
