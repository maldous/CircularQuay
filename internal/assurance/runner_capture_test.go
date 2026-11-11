package assurance

import (
	"context"
	"testing"
)

func TestSourceBlobCaptureContextIsNarrowlyBounded(t *testing.T) {
	if got := commandCaptureLimit(context.Background()); got != defaultCommandCaptureBytes {
		t.Fatalf("default capture limit = %d", got)
	}
	sourceContext := sourceBlobCaptureContext(context.Background(), maxSourceRasterBytes)
	if got := commandCaptureLimit(sourceContext); got != maxSourceRasterBytes {
		t.Fatalf("source blob capture limit = %d", got)
	}
	for _, candidate := range []int64{0, defaultCommandCaptureBytes, maxSourceBlobCaptureBytes + 1} {
		if got := commandCaptureLimit(sourceBlobCaptureContext(context.Background(), candidate)); got != defaultCommandCaptureBytes {
			t.Fatalf("candidate %d widened capture to %d", candidate, got)
		}
	}
}

func TestBoundedBufferRejectsOutputBeyondScopedSourceBlobLimit(t *testing.T) {
	buffer := boundedBuffer{maximum: maxSourceBlobCaptureBytes}
	if _, err := buffer.Write(make([]byte, maxSourceBlobCaptureBytes)); err != nil || buffer.overflow {
		t.Fatalf("bounded source blob capture rejected its exact limit: overflow=%t error=%v", buffer.overflow, err)
	}
	if _, err := buffer.Write([]byte{0}); err != nil || !buffer.overflow {
		t.Fatalf("bounded source blob capture accepted data beyond its limit: overflow=%t error=%v", buffer.overflow, err)
	}
}
