package ai

import (
	"errors"
	"strings"
	"testing"
)

func TestFallbackErrorDescribesSingleDecision(t *testing.T) {
	err := (&FallbackError{Provider: "test", Cause: errors.New("failed")}).Error()
	if !strings.Contains(err, "本次决策") || strings.Contains(err, "已切换本地 AI") {
		t.Fatalf("fallback message implies a persistent downgrade: %q", err)
	}
}
