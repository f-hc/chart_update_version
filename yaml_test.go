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
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReadYAMLDocuments(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantDocs int
		wantErr  bool
	}{
		{
			name:     "single document",
			content:  "key: value",
			wantDocs: 1,
			wantErr:  false,
		},
		{
			name:     "multiple documents",
			content:  "---\nkey1: value1\n---\nkey2: value2",
			wantDocs: 2,
			wantErr:  false,
		},
		{
			name:     "empty file",
			content:  "",
			wantDocs: 0,
			wantErr:  false,
		},
		{
			name:     "invalid yaml",
			content:  "key: [invalid",
			wantDocs: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".yaml")
			runReadYAMLDocumentsTest(t, path, tt.content, tt.wantDocs, tt.wantErr)
		})
	}
}

func runReadYAMLDocumentsTest(t *testing.T, path, content string, wantDocs int, wantErr bool) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	docs, err := readYAMLDocuments(path)

	if wantErr {
		if err == nil {
			t.Error("readYAMLDocuments() error = nil, want error")
		}

		return
	}

	if err != nil {
		t.Errorf("readYAMLDocuments() error = %v", err)
		return
	}

	if len(docs) != wantDocs {
		t.Errorf("readYAMLDocuments() got %d docs, want %d", len(docs), wantDocs)
	}
}

func TestReadYAMLDocumentsFileNotFound(t *testing.T) {
	_, err := readYAMLDocuments("/nonexistent/file.yaml")
	if err == nil {
		t.Error("readYAMLDocuments() error = nil, want error")
	}
}

func TestWriteYAMLDocuments(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.yaml")

	// Create a simple document
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "key"},
					{Kind: yaml.ScalarNode, Value: "value"},
				},
			},
		},
	}

	err := writeYAMLDocuments(context.Background(), io.Discard, path, []*yaml.Node{doc})
	if err != nil {
		t.Errorf("writeYAMLDocuments() error = %v", err)
		return
	}

	// Verify file was written
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("failed to read written file: %v", err)
		return
	}

	if len(content) == 0 {
		t.Error("writeYAMLDocuments() wrote empty file")
	}
}

func TestWriteYAMLDocumentsWithArtifactHub(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output_ah.yaml")

	// Create a document with artifacthub comment
	doc := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{
						Kind:        yaml.ScalarNode,
						Value:       "apiVersion",
						HeadComment: "# artifacthub: org/repo",
					},
					{Kind: yaml.ScalarNode, Value: "v1"},
				},
			},
		},
	}

	err := writeYAMLDocuments(context.Background(), io.Discard, path, []*yaml.Node{doc})
	if err != nil {
		t.Fatalf("writeYAMLDocuments failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	expectedPrefix := "# artifacthub: org/repo\n---\n"
	if !strings.HasPrefix(string(content), expectedPrefix) {
		t.Errorf("Expected prefix %q, got:\n%s", expectedPrefix, string(content))
	}
}

func TestGetAndSetTargetRevision(t *testing.T) {
	yamlContent := `apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  source:
    targetRevision: 1.0.0`

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &doc); err != nil {
		t.Fatal(err)
	}

	// Test getTargetRevision
	got := getTargetRevision(&doc)
	if got != "1.0.0" {
		t.Errorf("getTargetRevision() = %q, want %q", got, "1.0.0")
	}

	// Test setTargetRevision
	setTargetRevision(&doc, "2.0.0")

	got = getTargetRevision(&doc)
	if got != "2.0.0" {
		t.Errorf("after setTargetRevision(), getTargetRevision() = %q, want %q", got, "2.0.0")
	}
}

func TestKind(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "Application kind",
			content: "kind: Application",
			want:    "Application",
		},
		{
			name:    "other kind",
			content: "kind: Deployment",
			want:    "Deployment",
		},
		{
			name:    "no kind",
			content: "foo: bar",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc yaml.Node
			if err := yaml.Unmarshal([]byte(tt.content), &doc); err != nil {
				t.Fatal(err)
			}

			got := kind(&doc)
			if got != tt.want {
				t.Errorf("kind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocRoot(t *testing.T) {
	// Test with DocumentNode
	docNode := &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode},
		},
	}

	root := docRoot(docNode)
	if root.Kind != yaml.MappingNode {
		t.Errorf("docRoot() on DocumentNode returned kind %v, want MappingNode", root.Kind)
	}

	// Test with non-DocumentNode
	mappingNode := &yaml.Node{Kind: yaml.MappingNode}

	root = docRoot(mappingNode)
	if root != mappingNode {
		t.Error("docRoot() on non-DocumentNode should return the same node")
	}
}

func TestLookup(t *testing.T) {
	content := `level1:
  level2:
    level3: value`

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatal(err)
	}

	root := docRoot(&doc)

	tests := []struct {
		path []string
		want string
	}{
		{[]string{"level1", "level2", "level3"}, "value"},
		{[]string{"level1", "level2"}, ""},
		{[]string{"nonexistent"}, ""},
		{[]string{"level1", "nonexistent"}, ""},
	}

	for _, tt := range tests {
		got := lookup(root, tt.path...)
		if got != tt.want {
			t.Errorf("lookup(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMapGet(t *testing.T) {
	content := `key1: value1
key2: value2`

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatal(err)
	}

	root := docRoot(&doc)

	// Test existing key
	node := mapGet(root, "key1")
	if node == nil || node.Value != "value1" {
		t.Errorf("mapGet(key1) failed")
	}

	// Test non-existing key
	node = mapGet(root, "nonexistent")
	if node != nil {
		t.Errorf("mapGet(nonexistent) = %v, want nil", node)
	}

	// Test on non-mapping node
	scalarNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}

	node = mapGet(scalarNode, "key")
	if node != nil {
		t.Errorf("mapGet on scalar node = %v, want nil", node)
	}
}

func TestGetArtifactHubRepo(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "comment on document node",
			content: "# artifacthub: org/chart\nkind: Application",
			want:    "org/chart",
		},
		{
			name:    "no comment",
			content: "kind: Application",
			want:    "",
		},
		{
			name:    "comment with spaces",
			content: "# artifacthub:   org/chart  \nkind: Application",
			want:    "org/chart",
		},
		{
			name:    "different comment",
			content: "# some other comment\nkind: Application",
			want:    "",
		},
		{
			name:    "nested org/repo",
			content: "# artifacthub: cloudnative-pg/cloudnative-pg\nkind: Application",
			want:    "cloudnative-pg/cloudnative-pg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var doc yaml.Node
			if err := yaml.Unmarshal([]byte(tt.content), &doc); err != nil {
				t.Fatal(err)
			}

			got := getArtifactHubRepo(&doc)
			if got != tt.want {
				t.Errorf("getArtifactHubRepo() = %q, want %q", got, tt.want)
			}
		})
	}
}
