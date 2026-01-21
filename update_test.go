package main

import (
	"errors"
	"slices"
	"testing"

	"github.com/BooleanCat/go-functional/v2/it"
	"gopkg.in/yaml.v3"
)

type testCase struct {
	name        string
	read        func() ([]*yaml.Node, error)
	fetch       func() (string, error)
	write       func() error
	wantStatus  UpdateStatus
	wantCurrent string
	wantLatest  string
	wantErr     string
}

func TestUpdateChart(t *testing.T) {
	cfg := Config{Dir: "."}

	tests := []testCase{
		{
			name: "successful update",
			read: func() ([]*yaml.Node, error) {
				return []*yaml.Node{createMockAppNode("1.0.0")}, nil
			},
			fetch:       func() (string, error) { return "1.1.0", nil },
			write:       func() error { return nil },
			wantStatus:  StatusUpdated,
			wantCurrent: "1.0.0",
			wantLatest:  "1.1.0",
		},
		{
			name: "already up to date",
			read: func() ([]*yaml.Node, error) {
				return []*yaml.Node{createMockAppNode("1.1.0")}, nil
			},
			fetch:      func() (string, error) { return "1.1.0", nil },
			write:      func() error { return errors.New("write should not be called") },
			wantStatus: StatusUpToDate,
		},
		{
			name: "read error",
			read: func() ([]*yaml.Node, error) {
				return nil, errors.New("read failed")
			},
			fetch:      func() (string, error) { return "", nil },
			write:      func() error { return nil },
			wantStatus: StatusError,
			wantErr:    "read failed",
		},
		{
			name: "fetch error",
			read: func() ([]*yaml.Node, error) {
				return []*yaml.Node{createMockAppNode("1.0.0")}, nil
			},
			fetch: func() (string, error) {
				return "", errors.New("fetch failed")
			},
			write:      func() error { return nil },
			wantStatus: StatusError,
			wantErr:    "fetch failed",
		},
		{
			name: "write error",
			read: func() ([]*yaml.Node, error) {
				return []*yaml.Node{createMockAppNode("1.0.0")}, nil
			},
			fetch:      func() (string, error) { return "1.1.0", nil },
			write:      func() error { return errors.New("write failed") },
			wantStatus: StatusError,
			wantErr:    "write failed",
		},
		{
			name: "current version not found",
			read: func() ([]*yaml.Node, error) {
				// Return a node that doesn't have the Application/spec/source/targetRevision structure
				return []*yaml.Node{{Kind: yaml.DocumentNode, Content: []*yaml.Node{}}}, nil
			},
			fetch:      func() (string, error) { return "1.1.0", nil },
			write:      func() error { return nil },
			wantStatus: StatusError,
			wantErr:    "failed to read current version in app.yaml",
		},
	}

	it.ForEach(slices.Values(tests), func(tc testCase) {
		t.Run(tc.name, runUpdateChartTest(cfg, tc))
	})
}

func runUpdateChartTest(cfg Config, tc testCase) func(t *testing.T) {
	return func(t *testing.T) {
		mockRead := func(path string) ([]*yaml.Node, error) { return tc.read() }
		mockFetch := func(repo string) (string, error) { return tc.fetch() }
		mockWrite := func(path string, docs []*yaml.Node) error { return tc.write() }

		updater := MakeChartUpdater(cfg, mockRead, mockFetch, mockWrite)
		result := updater("app.yaml", "org/repo")

		assertStatus(t, tc.wantStatus, result.Status)
		assertString(t, "current", tc.wantCurrent, result.Current)
		assertString(t, "latest", tc.wantLatest, result.Latest)
		assertError(t, tc.wantErr, result.Error)
	}
}

func assertStatus(t *testing.T, want, got UpdateStatus) {
	if want != got {
		t.Errorf("expected status %v, got %v", want, got)
	}
}

func assertString(t *testing.T, name, want, got string) {
	if want != "" && want != got {
		t.Errorf("expected %s %s, got %s", name, want, got)
	}
}

func assertError(t *testing.T, want string, got error) {
	if want != "" {
		if got == nil || got.Error() != want {
			t.Errorf("expected error %q, got %v", want, got)
		}
	} else if got != nil {
		t.Errorf("unexpected error: %v", got)
	}
}

// Helper to create a minimal node structure that satisfies the lookup
func createMockAppNode(version string) *yaml.Node {
	// Construction of a minimal YAML AST for:
	// kind: Application
	// spec:
	//   source:
	//     targetRevision: <version>

	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "kind"},
					{Kind: yaml.ScalarNode, Value: "Application"},
					{Kind: yaml.ScalarNode, Value: "spec"},
					{
						Kind: yaml.MappingNode,
						Content: []*yaml.Node{
							{Kind: yaml.ScalarNode, Value: "source"},
							{
								Kind: yaml.MappingNode,
								Content: []*yaml.Node{
									{Kind: yaml.ScalarNode, Value: "targetRevision"},
									{Kind: yaml.ScalarNode, Value: version},
								},
							},
						},
					},
				},
			},
		},
	}
}
