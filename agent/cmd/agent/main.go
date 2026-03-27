package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"iscsi-gui/agent/internal/api"
	"iscsi-gui/agent/internal/audit"
	"iscsi-gui/agent/internal/config"
	"iscsi-gui/agent/internal/driver"
	"iscsi-gui/agent/internal/service"
	"iscsi-gui/agent/internal/systemd"
)

func main() {
	cfg := config.Load()
	targetDriver := driver.NewTargetCLI()
	targetsSvc := service.NewTargetsService(targetDriver)
	backstoresSvc := service.NewBackstoresService(targetDriver)
	mappingsSvc := service.NewMappingsService(targetDriver)
	aclsSvc := service.NewACLsService(targetDriver)
	portalsSvc := service.NewPortalsService(targetDriver)
	chapSvc := service.NewCHAPService(targetDriver)
	sessionsSvc := service.NewSessionsService(targetDriver)
	auditLog := audit.NewLogger(500)

	h := api.NewRouter(api.Dependencies{
		AgentListen: cfg.Listen,
		Targets:     targetsSvc,
		Backstores:  backstoresSvc,
		Mappings:    mappingsSvc,
		ACLs:        aclsSvc,
		Portals:     portalsSvc,
		CHAP:        chapSvc,
		Sessions:    sessionsSvc,
		Audit:       auditLog,
	})

	ln, cleanup, err := listen(cfg.Listen)
	if err != nil {
		log.Fatalf("agent listen failed: %v", err)
	}
	defer cleanup()

	srv := &http.Server{
		Handler: h,
	}

	errCh := make(chan error, 1)
	go func() {
		err := srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	if strings.HasPrefix(cfg.Listen, "/") {
		log.Printf("iscsi-agent listening on unix socket %s", cfg.Listen)
	} else {
		log.Printf("iscsi-agent listening on tcp %s", cfg.Listen)
	}

	if targetDriver.Available() {
		log.Printf("target backend ready: targetcli")
	} else {
		log.Printf("target backend unavailable: targetcli/targetcli-fb not found")
	}

	if err := systemd.Notify("READY=1"); err != nil {
		log.Printf("systemd notify READY failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Fatalf("agent server failed: %v", err)
		}
		return
	}

	if err := systemd.Notify("STOPPING=1"); err != nil {
		log.Printf("systemd notify STOPPING failed: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if closeErr := srv.Close(); closeErr != nil {
			log.Printf("forced close failed: %v", closeErr)
		}
	}

	if err := <-errCh; err != nil {
		log.Fatalf("agent server failed: %v", err)
	}
}

func listen(addr string) (net.Listener, func(), error) {
	if !strings.HasPrefix(addr, "/") {
		ln, err := net.Listen("tcp", addr)
		return ln, func() {}, err
	}

	dir := filepath.Dir(addr)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}

	if err := os.Remove(addr); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, err
	}

	ln, err := net.Listen("unix", addr)
	if err != nil {
		return nil, nil, err
	}

	if err := os.Chmod(addr, 0o660); err != nil {
		_ = ln.Close()
		return nil, nil, err
	}

	cleanup := func() {
		_ = ln.Close()
		if err := os.Remove(addr); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("failed to remove socket %s: %v", addr, err)
		}
	}
	return ln, cleanup, nil
}
