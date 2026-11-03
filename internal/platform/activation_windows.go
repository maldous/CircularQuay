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
	status, err := s.Query()
	if err != nil {
		return fault.New(fault.Apply, "query declared service")
	}
	if status.State == svc.Stopped {
		return nil
	}
	if status.State != svc.Running {
		return fault.New(fault.Apply, "service transition already pending")
	}
	if ctx.Err() != nil {
		return fault.New(fault.Apply, "activation canceled")
	}
	if _, err = s.Control(svc.Stop); err != nil {
		return fault.New(fault.Apply, "stop declared service")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fault.New(fault.Apply, "service stop timeout")
		case <-ticker.C:
			status, err = s.Query()
			if err != nil {
				return fault.New(fault.Apply, "query stopping service")
			}
			if status.State == svc.Stopped {
				if ctx.Err() != nil {
					return fault.New(fault.Apply, "activation canceled")
				}
				if err = s.Start(); err != nil {
					return fault.New(fault.Apply, "start declared service")
				}
				return nil
			}
		}
	}
}
