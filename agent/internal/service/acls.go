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
	ErrACLDriverUnavailable = errors.New("acl driver unavailable")
)

type ACL struct {
	TargetIQN    string `json:"target_iqn"`
	InitiatorIQN string `json:"initiator_iqn"`
}

type ACLsService struct {
	driver driver.ACLDriver
	mu     sync.Mutex
}

func NewACLsService(d driver.ACLDriver) *ACLsService {
	return &ACLsService{driver: d}
}

func (s *ACLsService) List(ctx context.Context, targetIQN string) ([]ACL, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return nil, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrACLDriverUnavailable
	}

	items, err := s.driver.ListACLs(ctx, targetIQN)
	if err != nil {
		return nil, err
	}
	result := make([]ACL, 0, len(items))
	for _, it := range items {
		result = append(result, ACL{TargetIQN: it.TargetIQN, InitiatorIQN: it.InitiatorIQN})
	}
	return result, nil
}

func (s *ACLsService) Create(ctx context.Context, targetIQN, initiatorIQN string) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	initiatorIQN = strings.TrimSpace(initiatorIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidIQN(initiatorIQN) {
		return false, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrACLDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.CreateACL(ctx, targetIQN, initiatorIQN); err != nil {
		if isACLAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *ACLsService) Delete(ctx context.Context, targetIQN, initiatorIQN string) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	initiatorIQN = strings.TrimSpace(initiatorIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidIQN(initiatorIQN) {
		return false, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrACLDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.DeleteACL(ctx, targetIQN, initiatorIQN); err != nil {
		if isACLNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isACLAlreadyExists(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "exists")
}

func isACLNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "invalid acl") || strings.Contains(msg, "no such nodeacl")
}
