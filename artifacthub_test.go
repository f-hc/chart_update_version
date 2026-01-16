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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArtifactHubLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		wantVer    string
		wantErr    bool
	}{
		{
			name: "valid response with multiple versions",
			response: `{
				"available_versions": [
					{"version": "1.0.0"},
					{"version": "2.0.0"},
					{"version": "1.5.0"}
				]
			}`,
			statusCode: http.StatusOK,
			wantVer:    "2.0.0",
		},
		{
			name: "skips pre-release versions",
			response: `{
				"available_versions": [
					{"version": "1.0.0"},
					{"version": "2.0.0-alpha"},
					{"version": "1.5.0"}
				]
			}`,
			statusCode: http.StatusOK,
			wantVer:    "1.5.0",
		},
		{
			name: "only pre-release versions",
			response: `{
				"available_versions": [
					{"version": "1.0.0-alpha"},
					{"version": "2.0.0-beta"}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "empty versions",
			response:   `{"available_versions": []}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:       "not found",
			response:   `{"error": "not found"}`,
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "invalid json",
			response:   `<html>error</html>`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runArtifactHubTest(t, tt.response, tt.statusCode, tt.wantVer, tt.wantErr)
		})
	}
}

func runArtifactHubTest(t *testing.T, response string, statusCode int, wantVer string, wantErr bool) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
		if _, err := w.Write([]byte(response)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Override the API URL for testing
	oldAPI := artifactHubAPI
	defer func() { setArtifactHubAPI(oldAPI) }()
	setArtifactHubAPI(server.URL)

	ver, err := artifactHubLatestVersion("test/repo")

	if wantErr {
		if err == nil {
			t.Error("artifactHubLatestVersion() error = nil, want error")
		}
		return
	}

	if err != nil {
		t.Errorf("artifactHubLatestVersion() error = %v", err)
		return
	}

	if ver != wantVer {
		t.Errorf("artifactHubLatestVersion() = %q, want %q", ver, wantVer)
	}
}

// setArtifactHubAPI is a test helper to override the API URL
func setArtifactHubAPI(url string) {
	artifactHubAPI = url
}
