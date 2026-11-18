//go:build !windows

package repository

func nativeGitEnvironment() ([]string, error) { return nil, nil }
