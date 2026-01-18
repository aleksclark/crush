package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilterBannedCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		banned   []string
		allowed  []string
		expected []string
	}{
		{
			name:     "no allowed commands",
			banned:   []string{"curl", "wget", "ssh"},
			allowed:  nil,
			expected: []string{"curl", "wget", "ssh"},
		},
		{
			name:     "empty allowed list",
			banned:   []string{"curl", "wget", "ssh"},
			allowed:  []string{},
			expected: []string{"curl", "wget", "ssh"},
		},
		{
			name:     "allow single command",
			banned:   []string{"curl", "wget", "ssh"},
			allowed:  []string{"curl"},
			expected: []string{"wget", "ssh"},
		},
		{
			name:     "allow multiple commands",
			banned:   []string{"curl", "wget", "ssh", "nc"},
			allowed:  []string{"curl", "wget"},
			expected: []string{"ssh", "nc"},
		},
		{
			name:     "allow command not in banned list",
			banned:   []string{"curl", "wget"},
			allowed:  []string{"ls"},
			expected: []string{"curl", "wget"},
		},
		{
			name:     "allow all banned commands",
			banned:   []string{"curl", "wget"},
			allowed:  []string{"curl", "wget"},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := filterBannedCommands(tt.banned, tt.allowed)
			require.Equal(t, tt.expected, result)
		})
	}
}
