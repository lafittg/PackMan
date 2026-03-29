package types

import "testing"

func TestUsageLevelFromCount(t *testing.T) {
	tests := []struct {
		count int
		want  UsageLevel
	}{
		{0, UsageUnused},
		{1, UsageLow},
		{2, UsageLow},
		{3, UsageNormal},
		{10, UsageNormal},
		{11, UsageHeavy},
		{100, UsageHeavy},
	}

	for _, tt := range tests {
		got := UsageLevelFromCount(tt.count)
		if got != tt.want {
			t.Errorf("UsageLevelFromCount(%d) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestUsageLevelString(t *testing.T) {
	tests := []struct {
		level UsageLevel
		want  string
	}{
		{UsageUnused, "UNUSED"},
		{UsageLow, "Low"},
		{UsageNormal, "Normal"},
		{UsageHeavy, "Heavy"},
	}

	for _, tt := range tests {
		got := tt.level.String()
		if got != tt.want {
			t.Errorf("UsageLevel(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}
