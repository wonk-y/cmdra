package cmdraclient

import (
	"context"
	"strings"
	"testing"
)

func TestDialRequiresAddress(t *testing.T) {
	t.Parallel()

	_, err := Dial(context.Background(), DialConfig{})
	if err == nil {
		t.Fatal("expected error for missing address")
	}
	if !strings.Contains(err.Error(), "address is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
