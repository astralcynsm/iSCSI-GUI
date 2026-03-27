package service

import (
	"context"
	"errors"
	"strings"

	"iscsi-gui/agent/internal/driver"
	"iscsi-gui/agent/internal/iscsi"
)

var (
	ErrSessionDriverUnavailable = errors.New("session driver unavailable")
)

type Session struct {
	SID          string `json:"sid,omitempty"`
	TargetIQN    string `json:"target_iqn,omitempty"`
	InitiatorIQN string `json:"initiator_iqn,omitempty"`
	ClientIP     string `json:"client_ip,omitempty"`
	State        string `json:"state,omitempty"`
}

type SessionsService struct {
	driver driver.SessionDriver
}

func NewSessionsService(d driver.SessionDriver) *SessionsService {
	return &SessionsService{driver: d}
}

func (s *SessionsService) List(ctx context.Context, targetIQN string) ([]Session, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if targetIQN != "" && !iscsi.ValidIQN(targetIQN) {
		return nil, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrSessionDriverUnavailable
	}

	items, err := s.driver.ListSessions(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]Session, 0, len(items))
	for _, it := range items {
		row := Session{
			SID:          it.SID,
			TargetIQN:    it.TargetIQN,
			InitiatorIQN: it.InitiatorIQN,
			ClientIP:     it.ClientIP,
			State:        it.State,
		}
		// Best-effort filtering: keep rows with unknown target instead of hard-dropping,
		// because targetcli output format differs by distro/version and may omit target IQN.
		if targetIQN != "" && row.TargetIQN != "" && row.TargetIQN != targetIQN {
			continue
		}
		result = append(result, row)
	}
	return result, nil
}
