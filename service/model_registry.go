package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"pine-ai/dto"
	"pine-ai/global/enum"
	"sync"
	"sync/atomic"
	"time"

	"pine-ai/persistence"

	"github.com/google/uuid"
)

type ModelVersionView struct {
	ModelName   string    `json:"model_name"`
	Version     string    `json:"version"`
	BackendType string    `json:"backend_type"`
	InUseCount  int64     `json:"in_use_count"`
	Deleted     bool      `json:"deleted"`
	State       string    `json:"state"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type runtimeSnapshot struct {
	id          string
	backendType enum.BackendType
	version     string
	concurrency int
	weight      int
	inFlight    int64
}

func (s *runtimeSnapshot) ID() string                    { return s.id }
func (s *runtimeSnapshot) BackendType() enum.BackendType { return s.backendType }
func (s *runtimeSnapshot) Version() string               { return s.version }

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
		ModelName:   req.ModelName,
		Version:     req.Version,
		BackendType: enum.BackendType(req.BackendType),
		Concurrency: req.Concurrency,
		Weight:      req.Weight,
		Deleted:     false,
		State:       persistence.StateReady,
	}

	if err := r.store.CreateModel(context.Background(), &rec); err != nil {
		return err
	}

	snap := &runtimeSnapshot{
		id:          fmt.Sprintf("%s-%s-%d", req.ModelName, req.Version, time.Now().UnixNano()),
		backendType: enum.BackendType(req.BackendType),
		concurrency: req.Concurrency,
		version:     req.Version,
		weight:      req.Weight,
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
		id:          fmt.Sprintf("%s-%s-%d", name, version, time.Now().UnixNano()),
		backendType: enum.BackendType(rec.BackendType),
		concurrency: req.Concurrency,
		weight:      req.Weight,
	}
	r.runtimeMu.Lock()
	if _, ok := r.runtimes[name]; !ok {
		r.runtimes[name] = make(map[string]*runtimeSnapshot)
	}
	r.runtimes[name][version] = newSnap
	r.runtimeMu.Unlock()

	rec.BackendType = enum.BackendType(req.BackendType)
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
			ModelName:   rec.ModelName,
			Version:     rec.Version,
			BackendType: string(rec.BackendType),
			InUseCount:  inUseCount,
			Deleted:     rec.Deleted,
			State:       string(rec.State),
			UpdatedAt:   rec.UpdatedAt,
		})
	}
	return out
}

func (r *Registry) AcquireForInfer(name, version string) (*runtimeSnapshot, func(), error) {
	if version == "" {
		versions := r.runtimes[name]
		if len(versions) == 0 {
			return nil, nil, errors.New("no model version found")
		}
		totalWeight := 0
		targetVersion := ""
		for _, version := range versions {
			totalWeight += version.weight
		}
		randomWeight := rand.Intn(totalWeight)
		for _, version := range versions {
			randomWeight -= version.weight
			if randomWeight <= 0 {
				targetVersion = version.Version()
				break
			}
		}
		if targetVersion == "" {
			return nil, nil, errors.New("no model version found")
		}
		return r.AcquireForInfer(name, targetVersion)
	}
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
			id:          fmt.Sprintf("%s-%s-%s", name, version, uuid.New().String()),
			backendType: enum.BackendType(rec.BackendType),
			version:     version,
			concurrency: rec.Concurrency,
			weight:      rec.Weight,
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
