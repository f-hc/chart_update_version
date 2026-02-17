package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func TestShowDiffInternal(t *testing.T) {
	// Create a temporary file with original content
	content := `# artifacthub: org/repo
kind: Application
spec:
  source:
    targetRevision: 1.0.0
`

	tmpfile, err := os.CreateTemp(t.TempDir(), "test-diff-*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}

	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Parse the content to get nodes
	// We use readYAMLDocuments to ensure we get the structure exactly as the app does
	docs, err := readYAMLDocuments(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Modify the nodes to simulate an update
	updateDocuments(docs, "1.1.0")

	// Capture output
	r, w, _ := os.Pipe()

	err = showDiffInternal(context.Background(), w, tmpfile.Name(), docs)
	if err != nil {
		w.Close()

		t.Fatalf("showDiffInternal failed: %v", err)
	}

	w.Close()

	out, _ := io.ReadAll(r)
	output := string(out)

	// Check if output contains expected diff parts
	expectedParts := []string{
		"--- a/" + tmpfile.Name(),
		"+++ b/" + tmpfile.Name(),
		"-    targetRevision: 1.0.0",
		"+    targetRevision: 1.1.0",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Diff output missing %q. Got:\n%s", part, output)
		}
	}
}
