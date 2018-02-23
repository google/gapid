// Copyright (C) 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package data

import (
	"bytes"
	"sort"
)

// Dedupe returns a new byte slice containing the concatenated bytes of slices
// with duplicates removed, along with indices that map slices into the new
// slice.
func Dedupe(slices [][]byte) ([]byte, []int) {
	if len(slices) == 0 {
		return nil, nil
	}

	type lenidx struct {
		len, idx int
	}

	lis := make([]lenidx, len(slices))
	for i, slice := range slices {
		lis[i] = lenidx{len(slice), i}
	}
	sort.Slice(lis, func(i, j int) bool { return lis[i].len > lis[j].len })

	idxs := make([]int, len(slices))

	buf := bytes.Buffer{}
	buf.Write(slices[lis[0].idx])
	idxs[lis[0].idx] = 0

	for _, li := range lis[1:] {
		s := slices[li.idx]
		idx := bytes.Index(buf.Bytes(), s)
		if idx < 0 {
			idxs[li.idx] = buf.Len()
			buf.Write(s)
		} else {
			idxs[li.idx] = idx
		}
	}

	return buf.Bytes(), idxs
}
