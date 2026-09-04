//go:build !windows

package main

import "context"

func acquireSingleInstance() (func(), bool, error) {
	return func() {}, true, nil
}

func prepareSingleInstanceWake() error {
	return nil
}

func requestExistingInstanceWindow() error {
	return nil
}

func watchSingleInstanceWake(ctx context.Context, onRequest func()) {}
