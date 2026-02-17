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
	"iter"
)

func logwf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprint(w, "▶ ")
	_, _ = fmt.Fprintf(w, format, a...) // #nosec G705
	_, _ = fmt.Fprintln(w)
}

func ForEach[T any](seq iter.Seq[T], action func(T)) {
	for v := range seq {
		action(v)
	}
}

func ForEachWithError[T any](seq iter.Seq[T], action func(T) error) error {
	for v := range seq {
		if err := action(v); err != nil {
			return err
		}
	}

	return nil
}
