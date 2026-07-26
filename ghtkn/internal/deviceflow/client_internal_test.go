package deviceflow

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	t.Parallel()
	input := newMockInput()
	client := NewClient(input)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.input.Stderr == nil {
		t.Error("stderr not set")
	}
}
