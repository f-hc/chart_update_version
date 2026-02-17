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
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/BooleanCat/go-functional/v2/it"
	"gopkg.in/yaml.v3"
)

type UpdateStatus string

const (
	StatusUpToDate UpdateStatus = "up-to-date"
	StatusUpdated  UpdateStatus = "updated"
	StatusError    UpdateStatus = "error"
)

type UpdateResult struct {
	File    string
	Repo    string
	Current string
	Latest  string
	Status  UpdateStatus
	Error   error
}

type (
	YAMLReader func(path string) ([]*yaml.Node, error)
	YAMLWriter func(ctx context.Context, w io.Writer, path string, docs []*yaml.Node) error
)

func MakeChartUpdater(
	cfg Config,
	read YAMLReader,
	fetch VersionFetcher,
	write YAMLWriter,
) func(ctx context.Context, w io.Writer, file, repo string) UpdateResult {
	return func(ctx context.Context, w io.Writer, file, repo string) UpdateResult {
		path := filepath.Join(cfg.Dir, file)

		docs, err := read(path)
		if err != nil {
			return newErrorResult(file, repo, err)
		}

		current, found := findCurrentVersion(docs)
		if !found {
			return newErrorResult(file, repo, fmt.Errorf("failed to read current version in %s", file))
		}

		latest, err := fetch(ctx, repo)
		if err != nil {
			return newErrorResultWithCurrent(file, repo, current, err)
		}

		if !versionLess(current, latest) {
			return UpdateResult{
				File:    file,
				Repo:    repo,
				Current: current,
				Latest:  latest,
				Status:  StatusUpToDate,
				Error:   nil,
			}
		}

		updateDocuments(docs, latest)

		if writeErr := write(ctx, w, path, docs); writeErr != nil {
			return newErrorResultWithVersions(file, repo, current, latest, writeErr)
		}

		return UpdateResult{File: file, Repo: repo, Current: current, Latest: latest, Status: StatusUpdated, Error: nil}
	}
}

func findCurrentVersion(docs []*yaml.Node) (string, bool) {
	n, found := it.Find(slices.Values(docs), func(n *yaml.Node) bool {
		return kind(n) == KindApplication
	})

	if found {
		return getTargetRevision(n), true
	}

	return "", false
}

func updateDocuments(docs []*yaml.Node, version string) {
	appDocs := it.Filter(slices.Values(docs), func(n *yaml.Node) bool {
		return kind(n) == KindApplication
	})

	ForEach(appDocs, func(d *yaml.Node) {
		setTargetRevision(d, version)
	})
}

func newErrorResult(file, repo string, err error) UpdateResult {
	return UpdateResult{File: file, Repo: repo, Current: "", Latest: "", Status: StatusError, Error: err}
}

func newErrorResultWithCurrent(file, repo, current string, err error) UpdateResult {
	return UpdateResult{File: file, Repo: repo, Current: current, Latest: "", Status: StatusError, Error: err}
}

func newErrorResultWithVersions(file, repo, current, latest string, err error) UpdateResult {
	return UpdateResult{File: file, Repo: repo, Current: current, Latest: latest, Status: StatusError, Error: err}
}
