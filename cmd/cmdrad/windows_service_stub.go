//go:build !windows

package main

func runServiceHost(_ string, cfg Config) error {
	return runForeground(cfg)
}
