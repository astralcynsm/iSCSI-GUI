package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"iscsi-gui/agent/internal/driver"
	"iscsi-gui/agent/internal/iscsi"
)

var (
	ErrDriverUnavailable = errors.New("target driver unavailable")
	ErrInvalidIQN        = errors.New("invalid iqn")
)

type Target struct {
	IQN string `json:"iqn"`
}

type TargetsService struct {
	driver driver.TargetDriver
	mu     sync.Mutex
}

func NewTargetsService(d driver.TargetDriver) *TargetsService {
	return &TargetsService{driver: d}
}

func (s *TargetsService) List(ctx context.Context) ([]Target, error) {
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrDriverUnavailable
	}

	items, err := s.driver.ListTargets(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Target, 0, len(items))
	for _, iqn := range items {
		res = append(res, Target{IQN: iqn})
	}
	return res, nil
}

func (s *TargetsService) Create(ctx context.Context, iqn string) (bool, error) {
	iqn = strings.TrimSpace(iqn)
	if !iscsi.ValidIQN(iqn) {
		return false, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.driver.ListTargets(ctx)
	if err != nil {
		return false, err
	}
	for _, it := range existing {
		if it == iqn {
			return false, nil
		}
	}

	if err := s.driver.CreateTarget(ctx, iqn); err != nil {
		return false, err
	}
	return true, nil
}

func (s *TargetsService) Delete(ctx context.Context, iqn string) (bool, error) {
	iqn = strings.TrimSpace(iqn)
	if !iscsi.ValidIQN(iqn) {
		return false, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, err := s.driver.ListTargets(ctx)
	if err != nil {
		return false, err
	}
	found := false
	for _, it := range existing {
		if it == iqn {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	if err := s.driver.DeleteTarget(ctx, iqn); err != nil {
		return false, err
	}
	return true, nil
}
