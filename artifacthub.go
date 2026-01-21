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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/BooleanCat/go-functional/v2/it"
)

// ArtifactHubVersion represents a version entry in the API response.
type ArtifactHubVersion struct {
	Version string `json:"version"`
}

// ArtifactHubResponse represents the API response structure.
type ArtifactHubResponse struct {
	AvailableVersions []ArtifactHubVersion `json:"available_versions"`
}

// VersionFetcher is a function that retrieves the latest version for a repository.
type VersionFetcher func(repo string) (string, error)

// MakeArtifactHubFetcher creates a VersionFetcher that uses the ArtifactHub API.
func MakeArtifactHubFetcher(apiURL string, client *http.Client) VersionFetcher {
	return func(repo string) (string, error) {
		versions, err := fetchVersions(apiURL, client, repo)
		if err != nil {
			return "", err
		}

		latest, ok := findLatestStable(versions)
		if !ok {
			return "", errors.New("no stable versions found")
		}

		return latest, nil
	}
}

func fetchVersions(apiURL string, client *http.Client, repo string) (versions []string, err error) {
	resp, err := client.Get(apiURL + "/" + repo)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifacthub HTTP %d", resp.StatusCode)
	}

	var data ArtifactHubResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return slices.Collect(it.Map(slices.Values(data.AvailableVersions), func(v ArtifactHubVersion) string {
		return v.Version
	})), nil
}

func findLatestStable(versions []string) (string, bool) {
	stable := slices.Collect(it.Filter(slices.Values(versions), isStable))

	if len(stable) == 0 {
		return "", false
	}

	latest := slices.MaxFunc(stable, func(a, b string) int {
		if versionLess(a, b) {
			return -1
		}
		if versionLess(b, a) {
			return 1
		}
		return 0
	})

	return latest, true
}

func isStable(v string) bool {
	return !strings.Contains(v, "-")
}
