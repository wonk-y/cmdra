//go:build windows

package main

import (
	"context"

	"golang.org/x/sys/windows/svc"
)

type windowsService struct {
	name string
	cfg  Config
}

func runServiceHost(serviceName string, cfg Config) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return err
	}
	if !isService {
		return runForeground(cfg)
	}
	return svc.Run(serviceName, &windowsService{name: serviceName, cfg: cfg})
}

func (s *windowsService) Execute(_ []string, requests <-chan svc.ChangeRequest, statuses chan<- svc.Status) (bool, uint32) {
	statuses <- svc.Status{State: svc.StartPending}

	handle, err := startRuntime(s.cfg)
	if err != nil {
		return true, 1
	}
	defer func() { _ = handle.close() }()

	current := svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	statuses <- current

	for {
		select {
		case req := <-requests:
			switch req.Cmd {
			case svc.Interrogate:
				statuses <- current
			case svc.Stop, svc.Shutdown:
				current.State = svc.StopPending
				statuses <- current
				_ = handle.stop(context.Background())
				return false, 0
			default:
			}
		case err := <-handle.serveErr:
			if err == nil {
				return false, 0
			}
			return true, 1
		}
	}
}
