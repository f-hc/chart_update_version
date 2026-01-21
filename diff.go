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
	"os/exec"

	"gopkg.in/yaml.v3"
)

func showDiffInternal(path string, docs []*yaml.Node) (err error) {
	tmp, err := os.CreateTemp("", "update-version-*.yaml")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := os.Remove(tmp.Name()); removeErr != nil && err == nil {
			err = removeErr
		}
	}()

	enc := yaml.NewEncoder(tmp)
	enc.SetIndent(2)

	if err := encodeStream(enc, docs); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	cmd := exec.Command("git", "diff", "--no-index", "--", path, tmp.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil // git diff returns 1 when files differ
		}
		return err
	}
	return nil
}
