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
	"slices"
	"strconv"
	"strings"

	"github.com/BooleanCat/go-functional/v2/it"
)

// versionLess returns true if a < b using semantic versioning comparison.
func versionLess(a, b string) bool {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	limit := uint(max(len(as), len(bs)))

	seqA := it.Chain(slices.Values(as), it.Repeat("0"))
	seqB := it.Chain(slices.Values(bs), it.Repeat("0"))

	valA, valB, found := it.Find2(it.Take2(it.Zip(seqA, seqB), limit), func(a, b string) bool {
		return toInt(a) != toInt(b)
	})

	return found && toInt(valA) < toInt(valB)
}

func toInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
