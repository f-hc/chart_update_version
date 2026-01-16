// SPDX-License-Identifier: GPL-3.0-only
//
// Copyright (C) 2026 f-hc <207619282+f-hc@users.noreply.github.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 of the License.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverCharts(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		files      map[string]string
		wantCount  int
		wantCharts []ChartInfo
	}{
		{
			name: "single chart with comment",
			files: map[string]string{
				"app.yaml": "# artifacthub: org/chart\napiVersion: argoproj.io/v1alpha1\nkind: Application\nmetadata:\n  name: test",
			},
			wantCount: 1,
			wantCharts: []ChartInfo{
				{File: "app.yaml", Repo: "org/chart"},
			},
		},
		{
			name: "multiple charts",
			files: map[string]string{
				"app1.yaml": "# artifacthub: org1/chart1\nkind: Application",
				"app2.yaml": "# artifacthub: org2/chart2\nkind: Application",
			},
			wantCount: 2,
		},
		{
			name: "file without comment is skipped",
			files: map[string]string{
				"app.yaml": "kind: Application\nmetadata:\n  name: test",
			},
			wantCount: 0,
		},
		{
			name: "non-Application kind is skipped",
			files: map[string]string{
				"deploy.yaml": "# artifacthub: org/chart\nkind: Deployment",
			},
			wantCount: 0,
		},
		{
			name: "mixed files",
			files: map[string]string{
				"app.yaml":    "# artifacthub: org/chart\nkind: Application",
				"deploy.yaml": "kind: Deployment",
				"secret.yaml": "kind: Secret",
			},
			wantCount: 1,
		},
		{
			name: "yml extension supported",
			files: map[string]string{
				"app.yml": "# artifacthub: org/chart\nkind: Application",
			},
			wantCount: 1,
		},
		{
			name: "multi-document with Application",
			files: map[string]string{
				"app.yaml": "# artifacthub: org/chart\nkind: Application\n---\nkind: Secret",
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.Mkdir(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			createTestFiles(t, testDir, tt.files)

			charts, err := discoverCharts(testDir)
			if err != nil {
				t.Errorf("discoverCharts() error = %v", err)
				return
			}

			checkDiscoveredCharts(t, charts, tt.wantCount, tt.wantCharts)
		})
	}
}

func createTestFiles(t *testing.T, dir string, files map[string]string) {
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func checkDiscoveredCharts(t *testing.T, got []ChartInfo, wantCount int, wantCharts []ChartInfo) {
	if len(got) != wantCount {
		t.Errorf("discoverCharts() found %d charts, want %d", len(got), wantCount)
	}

	for _, want := range wantCharts {
		found := false
		for _, g := range got {
			if g.File == want.File && g.Repo == want.Repo {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("discoverCharts() missing expected chart %+v", want)
		}
	}
}

func TestDiscoverChartsErrors(t *testing.T) {
	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := discoverCharts("/nonexistent/path")
		if err == nil {
			t.Error("discoverCharts() error = nil, want error")
		}
	})

	t.Run("path is a file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "file.yaml")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := discoverCharts(tmpFile)
		if err == nil {
			t.Error("discoverCharts() error = nil, want error for file path")
		}
		if !contains(err.Error(), "not a directory") {
			t.Errorf("discoverCharts() error = %q, want error mentioning 'not a directory'", err.Error())
		}
	})
}

func TestExtractArtifactHubRepo(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "comment at start",
			content: "# artifacthub: org/chart\nkind: Application",
			want:    "org/chart",
		},
		{
			name:    "no comment",
			content: "kind: Application",
			want:    "",
		},
		{
			name:    "comment with extra spaces",
			content: "# artifacthub:   org/chart  \nkind: Application",
			want:    "org/chart",
		},
		{
			name:    "wrong comment prefix",
			content: "# other: org/chart\nkind: Application",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := extractArtifactHubRepo(path)
			if err != nil {
				t.Errorf("extractArtifactHubRepo() error = %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("extractArtifactHubRepo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
