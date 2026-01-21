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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/BooleanCat/go-functional/v2/it"
)

func main() {
	if err := run(os.Args, os.Getenv, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "❌", err)
		os.Exit(1)
	}
}

func run(args []string, getEnv func(string) string, stderr io.Writer) error {
	programName := filepath.Base(args[0])
	flags := args[1:]

	cfg, err := ParseConfig(flags, getEnv)
	if err != nil {
		if err.Error() == "help requested" {
			printUsage(stderr, programName)
			return nil
		}
		return err
	}

	return runApp(cfg, stderr)
}

func runApp(cfg Config, w io.Writer) error {
	discover := MakeChartDiscoverer(os.Stat, os.ReadDir, readYAMLDocuments)
	charts, err := discover(cfg.Dir)
	if err != nil {
		return err
	}

	if len(charts) == 0 {
		return fmt.Errorf("no charts with artifacthub comments found in %s", cfg.Dir)
	}

	if cfg.CheckOnly {
		runCheck(charts, w)
		return nil
	}

	return runUpdate(cfg, charts, w)
}

func runCheck(charts []ChartInfo, w io.Writer) {
	logw(w, "discovered %d chart(s) with artifacthub comments:", len(charts))
	ForEach(slices.Values(charts), func(c ChartInfo) {
		logw(w, "  %s → %s", c.File, c.Repo)
	})
}

func runUpdate(cfg Config, charts []ChartInfo, w io.Writer) error {
	const apiURL = "https://artifacthub.io/api/v1/packages/helm"
	client := &http.Client{Timeout: 60 * time.Second}

	fetcher := MakeArtifactHubFetcher(apiURL, client)

	var writer YAMLWriter = writeYAMLDocuments
	if cfg.DryRun {
		writer = showDiffInternal
	}

	updater := MakeChartUpdater(cfg, readYAMLDocuments, fetcher, writer)

	// Pipeline: Iterate -> Map(process) -> ForEach(log)
	process := func(c ChartInfo) UpdateResult {
		return updater(c.File, c.Repo)
	}

	return ForEachWithError(it.Map(slices.Values(charts), process), func(result UpdateResult) error {
		return logResult(result, w)
	})
}

func logResult(r UpdateResult, w io.Writer) error {
	if r.Error != nil {
		return r.Error
	}

	switch r.Status {
	case StatusUpdated:
		logw(w, "%s: %s → %s", r.File, r.Current, r.Latest)
	case StatusUpToDate:
		logw(w, "%s: already up to date (%s)", r.File, r.Current)
	case StatusError:
		if r.Error != nil {
			return r.Error
		} else {
			return fmt.Errorf("%s: unknown error", r.File)
		}
	}
	return nil
}

func printUsage(w io.Writer, exe string) {
	_, _ = fmt.Fprintf(w, `Usage:
  %s [flags]

Description:
  Updates Argo CD Application Helm chart versions by scanning for manifests
  with "# artifacthub: org/repo" comments and fetching the latest version
  from ArtifactHub.

License:
  GNU GPL v3.0 only - https://spdx.org/licenses/GPL-3.0-only.html

Flags:
  -d, --dir <path>    Path to argoapps directory (default: %s)
  -n, --dry-run       Show git diff without modifying files
  -C, --check         Discover charts and show what would be updated
  -h, --help          Show this help message

Environment:
  %s    Directory path (used if --dir is not provided)

Exit codes:
  0  Success
  1  Error

Examples:
  %s
  %s --dir ./my-apps
  %s --dry-run
  %s=./my-apps %s --check

`, exe, defaultArgoAppsDir, argoAppsDirEnvVar, exe, exe, exe, argoAppsDirEnvVar, exe)
}
