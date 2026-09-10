package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

func TestResolveReplacementPath(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	for _, tc := range []struct {
		name, target, want string
	}{
		{"sibling", "../dependency", filepath.Join(base, "..", "dependency")},
		{"child", "./dependency", filepath.Join(base, "dependency")},
		{"absolute", filepath.Join(base, "absolute"), filepath.Join(base, "absolute")},
		{"versioned fork", "github.com/aleksclark/fantasy v0.12.2-0.20260910105032-c654406e1947", "github.com/aleksclark/fantasy v0.12.2-0.20260910105032-c654406e1947"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			target := resolveReplacementPath(base, tc.target)
			require.Equal(t, tc.want, target)
			// Verify the exact generated replacement is valid Go module syntax.
			data := "module example.com/build\n\nreplace charm.land/fantasy => " + target + "\n"
			parsed, err := modfile.Parse("go.mod", []byte(data), nil)
			require.NoError(t, err)
			require.Len(t, parsed.Replace, 1)
		})
	}
}
