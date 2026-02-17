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

const (
	testAppFile    = "app.yaml"
	testChartRepo  = "org/chart"
	testAppContent = "# artifacthub: " + testChartRepo + "\nkind: Application"
	testFlagDir    = "flag/dir"
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
				testAppFile: testAppContent + "\nmetadata:\n  name: test",
			},
			wantCount: 1,
			wantCharts: []ChartInfo{
				{File: testAppFile, Repo: testChartRepo},
			},
		},
		{
			name: "multiple charts",
			files: map[string]string{
				"app1.yaml": "# artifacthub: org1/chart1\nkind: Application",
				"app2.yaml": "# artifacthub: org2/chart2\nkind: Application",
			},
			wantCount:  2,
			wantCharts: nil,
		},
		{
			name: "file without comment is skipped",
			files: map[string]string{
				testAppFile: "kind: Application\nmetadata:\n  name: test",
			},
			wantCount:  0,
			wantCharts: nil,
		},
		{
			name: "non-Application kind is skipped",
			files: map[string]string{
				"deploy.yaml": "# artifacthub: org/chart\nkind: Deployment",
			},
			wantCount:  0,
			wantCharts: nil,
		},
		{
			name: "mixed files",
			files: map[string]string{
				testAppFile:   testAppContent,
				"deploy.yaml": "kind: Deployment",
				"secret.yaml": "kind: Secret",
			},
			wantCount:  1,
			wantCharts: nil,
		},
		{
			name: "yml extension supported",
			files: map[string]string{
				"app.yml": testAppContent,
			},
			wantCount:  1,
			wantCharts: nil,
		},
		{
			name: "multi-document with Application",
			files: map[string]string{
				testAppFile: testAppContent + "\n---\nkind: Secret",
			},
			wantCount:  1,
			wantCharts: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.Mkdir(testDir, 0o750); err != nil {
				t.Fatal(err)
			}

			createTestFiles(t, testDir, tt.files)

			discover := MakeChartDiscoverer(os.Stat, os.ReadDir, readYAMLDocuments)

			charts, err := discover(testDir)
			if err != nil {
				t.Errorf("discoverCharts() error = %v", err)
				return
			}

			checkDiscoveredCharts(t, charts, tt.wantCount, tt.wantCharts)
		})
	}
}

func createTestFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func checkDiscoveredCharts(t *testing.T, got []ChartInfo, wantCount int, wantCharts []ChartInfo) {
	t.Helper()

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
	discover := MakeChartDiscoverer(os.Stat, os.ReadDir, readYAMLDocuments)

	t.Run("nonexistent directory", func(t *testing.T) {
		_, err := discover("/nonexistent/path")
		if err == nil {
			t.Error("discoverCharts() error = nil, want error")
		}
	})

	t.Run("path is a file", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "file.yaml")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}

		_, err := discover(tmpFile)
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
			content: testAppContent,
			want:    testChartRepo,
		},
		{
			name:    "no comment",
			content: "kind: Application",
			want:    "",
		},
		{
			name:    "comment with extra spaces",
			content: "# artifacthub:   org/chart  \nkind: Application",
			want:    testChartRepo,
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
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := extractArtifactHubRepo(readYAMLDocuments, path)
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

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "defaults",
			args: []string{},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "env var override",
			env: map[string]string{
				argoAppsDirEnvVar: "custom/dir",
			},
			args: []string{},
			want: Config{
				Dir:       "custom/dir",
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "flag override",
			args: []string{"--dir", testFlagDir},
			env:  nil,
			want: Config{
				Dir:       testFlagDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "flag overrides env var",
			env: map[string]string{
				argoAppsDirEnvVar: "env/dir",
			},
			args: []string{"--dir", testFlagDir},
			want: Config{
				Dir:       testFlagDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "dry run short",
			args: []string{"-n"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    true,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "dry run long",
			args: []string{"--dry-run"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    true,
				CheckOnly: false,
			},
			wantErr: false,
		},
		{
			name: "check short",
			args: []string{"-C"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: true,
			},
			wantErr: false,
		},
		{
			name: "check long",
			args: []string{"--check"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: true,
			},
			wantErr: false,
		},
		{
			name: "dry run and check incompatible",
			args: []string{"--dry-run", "--check"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    true,
				CheckOnly: true,
			},
			wantErr: true,
		},
		{
			name: "missing dir argument",
			args: []string{"--dir"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: true,
		},
		{
			name: "unknown flag",
			args: []string{"--unknown"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: true,
		},
		{
			name: "ignore test flags",
			args: []string{"-test.v"},
			env:  nil,
			want: Config{
				Dir:       defaultArgoAppsDir,
				DryRun:    false,
				CheckOnly: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getEnv := func(key string) string {
				if tt.env == nil {
					return ""
				}

				return tt.env[key]
			}

			got, err := ParseConfig(tt.args, getEnv)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseConfig() = %+v, want %+v", got, tt.want)
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
