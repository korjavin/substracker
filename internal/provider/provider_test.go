package provider

import (
	"testing"
	"time"
)

func TestUsageInfo(t *testing.T) {
	info := UsageInfo{
		ResetDate:           time.Now(),
		CurrentUsageSeconds: 10800, // 3 hours
		TotalLimitSeconds:   18000, // 5 hours
		IsBlocked:           false,
	}

	if info.CurrentUsageSeconds != 10800 {
		t.Errorf("expected 10800, got %d", info.CurrentUsageSeconds)
	}
	if info.TotalLimitSeconds != 18000 {
		t.Errorf("expected 18000, got %d", info.TotalLimitSeconds)
	}
	if info.IsBlocked != false {
		t.Errorf("expected false, got %v", info.IsBlocked)
	}
}
