package main

import (
	"testing"
)

func TestFindLatestStable(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     string
		found    bool
	}{
		{
			name:     "mixed versions",
			versions: []string{"1.0.0", "1.1.0-rc1", "1.0.1", "2.0.0-alpha"},
			want:     "1.0.1",
			found:    true,
		},
		{
			name:     "only stable versions",
			versions: []string{"1.0.0", "2.0.0", "1.5.0"},
			want:     "2.0.0",
			found:    true,
		},
		{
			name:     "only unstable versions",
			versions: []string{"1.0.0-rc1", "2.0.0-beta"},
			want:     "",
			found:    false,
		},
		{
			name:     "empty list",
			versions: []string{},
			want:     "",
			found:    false,
		},
		{
			name:     "semver ordering",
			versions: []string{"1.9.0", "1.10.0"},
			want:     "1.10.0",
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := findLatestStable(tt.versions)
			if found != tt.found {
				t.Errorf("findLatestStable() found = %v, want %v", found, tt.found)
			}
			if got != tt.want {
				t.Errorf("findLatestStable() = %v, want %v", got, tt.want)
			}
		})
	}
}
