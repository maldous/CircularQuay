package main

import (
	"context"
	"testing"
	"time"
)

func TestSignalBoundaryContractParentCancelStopsOwnedRegistration(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, stop := rootSignalNotifyContext(parent)
	defer stop()
	cancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("entry-point signal context lost parent cancellation")
	}
}
