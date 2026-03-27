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
	ErrCHAPDriverUnavailable = errors.New("chap driver unavailable")
	ErrCHAPTargetNotFound    = errors.New("chap target not found")
	ErrInvalidCHAPUser       = errors.New("invalid chap user")
	ErrInvalidCHAPPassword   = errors.New("invalid chap password")
)

type CHAPState struct {
	TargetIQN     string `json:"target_iqn"`
	Enabled       bool   `json:"enabled"`
	UserID        string `json:"userid,omitempty"`
	PasswordSet   bool   `json:"password_set"`
	MutualEnabled bool   `json:"mutual_enabled,omitempty"`
	MutualUserID  string `json:"mutual_userid,omitempty"`
	MutualPassSet bool   `json:"mutual_password_set,omitempty"`
}

type CHAPService struct {
	driver driver.CHAPDriver
	mu     sync.Mutex
}

func NewCHAPService(d driver.CHAPDriver) *CHAPService {
	return &CHAPService{driver: d}
}

func (s *CHAPService) Get(ctx context.Context, targetIQN string) (CHAPState, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return CHAPState{}, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return CHAPState{}, ErrCHAPDriverUnavailable
	}

	cfg, err := s.driver.GetCHAP(ctx, targetIQN)
	if err != nil {
		if isCHAPTargetPathMissing(err) {
			return CHAPState{}, ErrCHAPTargetNotFound
		}
		return CHAPState{}, err
	}
	return CHAPState{
		TargetIQN:     targetIQN,
		Enabled:       cfg.Enabled,
		UserID:        cfg.UserID,
		PasswordSet:   cfg.PasswordSet,
		MutualEnabled: cfg.MutualEnabled,
		MutualUserID:  cfg.MutualUserID,
		MutualPassSet: cfg.MutualPassSet,
	}, nil
}

func (s *CHAPService) Set(ctx context.Context, targetIQN string, enabled bool, userID, password string) (bool, CHAPState, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	userID = strings.TrimSpace(userID)
	password = strings.TrimSpace(password)

	if !iscsi.ValidIQN(targetIQN) {
		return false, CHAPState{}, ErrInvalidIQN
	}
	if enabled {
		if userID == "" {
			return false, CHAPState{}, ErrInvalidCHAPUser
		}
		if password == "" {
			return false, CHAPState{}, ErrInvalidCHAPPassword
		}
	}
	if s.driver == nil || !s.driver.Available() {
		return false, CHAPState{}, ErrCHAPDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.driver.GetCHAP(ctx, targetIQN)
	if err != nil {
		if isCHAPTargetPathMissing(err) {
			return false, CHAPState{}, ErrCHAPTargetNotFound
		}
		return false, CHAPState{}, err
	}

	noChange := !enabled && !current.Enabled
	if enabled && current.Enabled && current.UserID == userID && current.PasswordSet {
		noChange = true
	}
	if noChange {
		state := CHAPState{
			TargetIQN:     targetIQN,
			Enabled:       current.Enabled,
			UserID:        current.UserID,
			PasswordSet:   current.PasswordSet,
			MutualEnabled: current.MutualEnabled,
			MutualUserID:  current.MutualUserID,
			MutualPassSet: current.MutualPassSet,
		}
		return false, state, nil
	}

	if err := s.driver.SetCHAP(ctx, targetIQN, enabled, userID, password); err != nil {
		if isCHAPTargetPathMissing(err) {
			return false, CHAPState{}, ErrCHAPTargetNotFound
		}
		return false, CHAPState{}, err
	}

	updated, err := s.driver.GetCHAP(ctx, targetIQN)
	if err != nil {
		if isCHAPTargetPathMissing(err) {
			return false, CHAPState{}, ErrCHAPTargetNotFound
		}
		return false, CHAPState{}, err
	}

	state := CHAPState{
		TargetIQN:     targetIQN,
		Enabled:       updated.Enabled,
		UserID:        updated.UserID,
		PasswordSet:   updated.PasswordSet,
		MutualEnabled: updated.MutualEnabled,
		MutualUserID:  updated.MutualUserID,
		MutualPassSet: updated.MutualPassSet,
	}
	return true, state, nil
}

func isCHAPTargetPathMissing(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such path /iscsi/") || strings.Contains(msg, "invalid target")
}
