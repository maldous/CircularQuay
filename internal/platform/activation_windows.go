//go:build windows

package platform

import (
	"context"
	"time"

	"github.com/maldous/CircularQuay/internal/fault"
	"github.com/maldous/CircularQuay/internal/model"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func activateNative(ctx context.Context, target model.Target) error {
	if target.Activation != "restart" {
		return fault.New(fault.Unsupported, "SCM reload is service specific")
	}
	m, err := mgr.Connect()
	if err != nil {
		return fault.New(fault.Apply, "connect service manager")
	}
	defer m.Disconnect()
	s, err := m.OpenService(target.Service)
	if err != nil {
		return fault.New(fault.Apply, "open declared service")
	}
	defer s.Close()
	return restartService(ctx, s)
}

func restartService(ctx context.Context, service *mgr.Service) error {
	status, err := service.Query()
	if err != nil {
		return fault.New(fault.Apply, "query declared service")
	}
	if status.State != svc.Running {
		if status.State == svc.Stopped {
			return nil
		}
		return fault.New(fault.Apply, "service transition already pending")
	}
	if err := stopService(ctx, service); err != nil {
		return err
	}
	return startWhenStopped(ctx, service)
}

func stopService(ctx context.Context, service *mgr.Service) error {
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "activation canceled")
	}
	if _, err := service.Control(svc.Stop); err != nil {
		return fault.New(fault.Apply, "stop declared service")
	}
	return nil
}

func startWhenStopped(ctx context.Context, service *mgr.Service) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	if err := waitForStopped(ctx, ticker, service); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "activation canceled")
	}
	if err := service.Start(); err != nil {
		return fault.New(fault.Apply, "start declared service")
	}
	return nil
}

func waitForStopped(ctx context.Context, ticker *time.Ticker, service *mgr.Service) error {
	for {
		select {
		case <-ctx.Done():
			return fault.New(fault.Apply, "service stop timeout")
		case <-ticker.C:
			status, err := service.Query()
			if err != nil {
				return fault.New(fault.Apply, "query stopping service")
			}
			if status.State == svc.Stopped {
				return nil
			}
		}
	}
}
