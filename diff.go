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
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"gopkg.in/yaml.v3"
)

func showDiffInternal(_ context.Context, w io.Writer, path string, docs []*yaml.Node) error {
	// 1. Read original file
	origBytes, err := os.ReadFile(filepath.Clean(path)) //nolint:gosec // CLI tool reads user-provided paths
	if err != nil {
		return fmt.Errorf("read original file: %w", err)
	}

	orig := string(origBytes)

	// 2. Generate new content in memory
	var buf bytes.Buffer

	nodes := docs
	if len(docs) > 0 {
		first, comment := extractComment(docs[0])
		if comment != "" {
			if _, err = fmt.Fprintf(&buf, "%s\n---\n", comment); err != nil { //nolint:gosec // false positive
				return fmt.Errorf("write yaml comment to buffer: %w", err)
			}

			nodes = append([]*yaml.Node{first}, docs[1:]...)
		}
	}

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(yamlIndent)

	if err = encodeStream(enc, nodes); err != nil {
		return fmt.Errorf("encode yaml to buffer: %w", err)
	}

	if err = enc.Close(); err != nil {
		return fmt.Errorf("close encoder: %w", err)
	}

	newContent := buf.String()

	// 3. Compute diff
	edits := myers.ComputeEdits(span.URIFromPath(path), orig, newContent)

	// 4. Print diff
	// Use "a/" and "b/" prefixes to mimic git diff output
	diff := fmt.Sprint(gotextdiff.ToUnified("a/"+path, "b/"+path, orig, edits))
	fmt.Fprint(w, diff) //nolint:gosec // false positive for CLI tool

	return nil
}
