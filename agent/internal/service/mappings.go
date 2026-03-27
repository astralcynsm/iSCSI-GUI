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
	ErrMappingDriverUnavailable = errors.New("mapping driver unavailable")
	ErrInvalidLunID             = errors.New("invalid lun id")
)

type Mapping struct {
	TargetIQN     string `json:"target_iqn"`
	LunID         int    `json:"lun_id"`
	BackstoreType string `json:"backstore_type,omitempty"`
	BackstoreName string `json:"backstore_name,omitempty"`
}

type MappingsService struct {
	driver driver.MappingDriver
	mu     sync.Mutex
}

func NewMappingsService(d driver.MappingDriver) *MappingsService {
	return &MappingsService{driver: d}
}

func (s *MappingsService) List(ctx context.Context, targetIQN string) ([]Mapping, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return nil, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrMappingDriverUnavailable
	}

	items, err := s.driver.ListMappings(ctx, targetIQN)
	if err != nil {
		return nil, err
	}
	result := make([]Mapping, 0, len(items))
	for _, it := range items {
		result = append(result, Mapping{
			TargetIQN:     it.TargetIQN,
			LunID:         it.LunID,
			BackstoreType: it.BackstoreType,
			BackstoreName: it.BackstoreName,
		})
	}
	return result, nil
}

func (s *MappingsService) Create(ctx context.Context, targetIQN, backstoreType, backstoreName string, lunID *int) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	backstoreType = strings.ToLower(strings.TrimSpace(backstoreType))
	backstoreName = strings.TrimSpace(backstoreName)

	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidBackstoreType(backstoreType) {
		return false, ErrInvalidBackstoreType
	}
	if !iscsi.ValidBackstoreName(backstoreName) {
		return false, ErrInvalidBackstoreName
	}
	if lunID != nil && !iscsi.ValidLunID(*lunID) {
		return false, ErrInvalidLunID
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrMappingDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.CreateMapping(ctx, targetIQN, backstoreType, backstoreName, lunID); err != nil {
		if isMappingAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *MappingsService) Delete(ctx context.Context, targetIQN string, lunID int) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidLunID(lunID) {
		return false, ErrInvalidLunID
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrMappingDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.DeleteMapping(ctx, targetIQN, lunID); err != nil {
		if isMappingNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isMappingAlreadyExists(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "already in use") || strings.Contains(msg, "exists")
}

func isMappingNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no lun") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "invalid lun")
}
