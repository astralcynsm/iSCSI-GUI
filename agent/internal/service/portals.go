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
	ErrPortalDriverUnavailable = errors.New("portal driver unavailable")
	ErrInvalidPortalIP         = errors.New("invalid portal ip")
	ErrInvalidPortalPort       = errors.New("invalid portal port")
	ErrPortalTargetNotFound    = errors.New("portal target not found")
)

type Portal struct {
	TargetIQN string `json:"target_iqn"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
}

type PortalsService struct {
	driver driver.PortalDriver
	mu     sync.Mutex
}

func NewPortalsService(d driver.PortalDriver) *PortalsService {
	return &PortalsService{driver: d}
}

func (s *PortalsService) List(ctx context.Context, targetIQN string) ([]Portal, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	if !iscsi.ValidIQN(targetIQN) {
		return nil, ErrInvalidIQN
	}
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrPortalDriverUnavailable
	}

	items, err := s.driver.ListPortals(ctx, targetIQN)
	if err != nil {
		return nil, err
	}
	result := make([]Portal, 0, len(items))
	for _, it := range items {
		result = append(result, Portal{
			TargetIQN: it.TargetIQN,
			IP:        it.IP,
			Port:      it.Port,
		})
	}
	return result, nil
}

func (s *PortalsService) Create(ctx context.Context, targetIQN, ip string, port int) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	ip = strings.TrimSpace(ip)
	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidPortalIP(ip) {
		return false, ErrInvalidPortalIP
	}
	if !iscsi.ValidPortalPort(port) {
		return false, ErrInvalidPortalPort
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrPortalDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.CreatePortal(ctx, targetIQN, ip, port); err != nil {
		if isPortalAlreadyExists(err) {
			return false, nil
		}
		if isPortalTargetPathMissing(err) {
			return false, ErrPortalTargetNotFound
		}
		return false, err
	}
	return true, nil
}

func (s *PortalsService) Delete(ctx context.Context, targetIQN, ip string, port int) (bool, error) {
	targetIQN = strings.TrimSpace(targetIQN)
	ip = strings.TrimSpace(ip)
	if !iscsi.ValidIQN(targetIQN) {
		return false, ErrInvalidIQN
	}
	if !iscsi.ValidPortalIP(ip) {
		return false, ErrInvalidPortalIP
	}
	if !iscsi.ValidPortalPort(port) {
		return false, ErrInvalidPortalPort
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrPortalDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.DeletePortal(ctx, targetIQN, ip, port); err != nil {
		if isPortalNotFound(err) {
			return false, nil
		}
		if isPortalTargetPathMissing(err) {
			return false, ErrPortalTargetNotFound
		}
		return false, err
	}
	return true, nil
}

func isPortalAlreadyExists(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "exists")
}

func isPortalNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "does not exist") || strings.Contains(msg, "invalid portal")
}

func isPortalTargetPathMissing(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such path /iscsi/") || strings.Contains(msg, "invalid target")
}
