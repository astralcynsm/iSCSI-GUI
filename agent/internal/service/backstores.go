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
	ErrBackstoreDriverUnavailable = errors.New("backstore driver unavailable")
	ErrInvalidBackstoreType       = errors.New("invalid backstore type")
	ErrInvalidBackstoreName       = errors.New("invalid backstore name")
	ErrInvalidBackstorePath       = errors.New("invalid backstore path")
	ErrInvalidBackstoreSize       = errors.New("invalid backstore size")
)

type Backstore struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type BackstoresService struct {
	driver driver.BackstoreDriver
	mu     sync.Mutex
}

func NewBackstoresService(d driver.BackstoreDriver) *BackstoresService {
	return &BackstoresService{driver: d}
}

func (s *BackstoresService) List(ctx context.Context) ([]Backstore, error) {
	if s.driver == nil || !s.driver.Available() {
		return nil, ErrBackstoreDriverUnavailable
	}

	items, err := s.driver.ListBackstores(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Backstore, 0, len(items))
	for _, it := range items {
		result = append(result, Backstore{Name: it.Name, Type: it.Type})
	}
	return result, nil
}

func (s *BackstoresService) Create(ctx context.Context, typ, name, path, size string) (bool, error) {
	typ = normalize(typ)
	name = strings.TrimSpace(name)
	path = strings.TrimSpace(path)
	size = strings.TrimSpace(size)

	if !iscsi.ValidBackstoreType(typ) {
		return false, ErrInvalidBackstoreType
	}
	if !iscsi.ValidBackstoreName(name) {
		return false, ErrInvalidBackstoreName
	}
	if !iscsi.ValidBackstorePath(path) {
		return false, ErrInvalidBackstorePath
	}
	if typ == "fileio" && size == "" {
		return false, ErrInvalidBackstoreSize
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrBackstoreDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.driver.CreateBackstore(ctx, typ, name, path, size); err != nil {
		if isBackstoreAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *BackstoresService) Delete(ctx context.Context, typ, name string) (bool, error) {
	typ = normalize(typ)
	name = strings.TrimSpace(name)

	if !iscsi.ValidBackstoreName(name) {
		return false, ErrInvalidBackstoreName
	}
	if typ != "" && !iscsi.ValidBackstoreType(typ) {
		return false, ErrInvalidBackstoreType
	}
	if s.driver == nil || !s.driver.Available() {
		return false, ErrBackstoreDriverUnavailable
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if typ != "" {
		if err := s.driver.DeleteBackstore(ctx, typ, name); err != nil {
			if isBackstoreNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	existing, err := s.driver.ListBackstores(ctx)
	if err != nil {
		return false, err
	}

	resolvedType := ""
	for _, it := range existing {
		if it.Name == name {
			resolvedType = it.Type
			break
		}
	}
	if resolvedType == "" {
		return false, nil
	}

	if err := s.driver.DeleteBackstore(ctx, resolvedType, name); err != nil {
		if isBackstoreNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isBackstoreAlreadyExists(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "exists:")
}

func isBackstoreNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no storage object named") || strings.Contains(msg, "does not exist")
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
