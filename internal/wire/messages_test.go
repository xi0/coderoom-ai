package wire

import "testing"

func TestContextUsagePercent(t *testing.T) {
	tests := []struct {
		name  string
		usage *ContextUsage
		want  int
	}{
		{
			name:  "nil usage",
			usage: nil,
			want:  0,
		},
		{
			name:  "zero limit",
			usage: &ContextUsage{Used: 100, Limit: 0},
			want:  0,
		},
		{
			name:  "negative limit",
			usage: &ContextUsage{Used: 100, Limit: -1},
			want:  0,
		},
		{
			name:  "half",
			usage: &ContextUsage{Used: 50, Limit: 100},
			want:  50,
		},
		{
			name:  "rounds down",
			usage: &ContextUsage{Used: 1, Limit: 3},
			want:  33,
		},
		{
			name:  "rounds up",
			usage: &ContextUsage{Used: 2, Limit: 3},
			want:  67,
		},
		{
			name:  "full",
			usage: &ContextUsage{Used: 100, Limit: 100},
			want:  100,
		},
		{
			name:  "over the limit",
			usage: &ContextUsage{Used: 120, Limit: 100},
			want:  120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.usage.Percent(); got != tt.want {
				t.Fatalf("Percent() = %d, want %d", got, tt.want)
			}
		})
	}
}
