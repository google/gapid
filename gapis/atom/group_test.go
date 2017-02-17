// Copyright (C) 2017 Google Inc.
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

package atom

import (
	"reflect"
	"testing"
)

func check(t *testing.T, name string, expected, got uint64) {
	if expected != got {
		t.Errorf("%s was not as expected.\nExpected: %d\nGot:      %d", name, expected, got)
	}
}

// Root-group
//   │
//   ├─ [0..99] ───── Atoms[0..99]
//   │
//   ├─ [100] ─────── Sub-group 0
//   │                  │
//   │                  └─ [0..99] ── Atoms[100..199]
//   │
//   ├─ [101..200] ── Atoms[200..299]
//   │
//   ├─ [201] ─────── Sub-group 1
//   │                  │
//   │                  ├─ [0..39] ── Atoms[300..339]
//   │                  │
//   │                  ├─ [40] ───── Sub-group 0
//   │                  │               │
//   │                  │               └─ [0..19] Atoms[340..359]
//   │                  │
//   │                  └─ [41..81] ─ Atoms[360..399]
//   │
//   ├─ [202..301] ── Atoms[400..499]
//   │
//   ├─ [302] ─────── Sub-group 2
//   │                  │
//   │                  └─ [0..100) ─ Atoms[500..599]
//   │
//   └─ [303..9702] ─ Atoms[600..9999]
//
func buildTestGroup() Group {
	return Group{
		Range: Range{Start: 0, End: 10000},
		SubGroups: GroupList{
			Group{Range: Range{Start: 100, End: 200}},
			Group{
				Range: Range{Start: 300, End: 400},
				SubGroups: GroupList{
					Group{Range: Range{Start: 340, End: 360}},
				},
			},
			Group{Range: Range{Start: 500, End: 600}},
		},
	}
}

func TestGroupCount(t *testing.T) {
	root := buildTestGroup()

	check(t, "root count", 10000-300+3, root.Count())
	check(t, "sub group 0 count", 100, root.SubGroups[0].Count())
	check(t, "sub group 1 count", 40+1+40, root.SubGroups[1].Count())
	check(t, "sub group 1's sub group count", 20, root.SubGroups[1].SubGroups[0].Count())
	check(t, "sub group 2 count", 100, root.SubGroups[2].Count())
}

func TestGroupIndex(t *testing.T) {
	root := buildTestGroup()
	for _, test := range []struct {
		index                 uint64
		expectedBaseAtomIndex uint64
		expectedSubGroup      *Group
	}{
		{0, 0, nil},
		{1, 1, nil},
		{50, 50, nil},
		{100, 100, &root.SubGroups[0]},
		{101, 200, nil},
		{102, 201, nil},
		{151, 250, nil},
		{200, 299, nil},
		{201, 300, &root.SubGroups[1]},
		{202, 400, nil},
		{203, 401, nil},
		{252, 450, nil},
		{301, 499, nil},
		{302, 500, &root.SubGroups[2]},
		{303, 600, nil},
		{304, 601, nil},
		{353, 650, nil},
		{402, 699, nil},
	} {
		gotBaseAtomIndex, gotSubGroup := root.Index(test.index)
		if test.expectedBaseAtomIndex != gotBaseAtomIndex {
			t.Errorf("base atom id was not as expected for index %d.\nExpected: %d\nGot:      %d",
				test.index, test.expectedBaseAtomIndex, gotBaseAtomIndex)
		}
		if test.expectedSubGroup != gotSubGroup {
			t.Errorf("sub group was not as expected for index %d.\nExpected: %+v\nGot:      %+v",
				test.index, test.expectedSubGroup, gotSubGroup)
		}
	}
}

func TestGroupIndexOf(t *testing.T) {
	root := buildTestGroup()
	for _, test := range []struct {
		atomIndex uint64
		expected  uint64
	}{
		{0, 0},
		{1, 1},
		{50, 50},
		{100, 100},
		{101, 100},
		{150, 100},
		{199, 100},
		{200, 101},
		{201, 102},
		{250, 151},
		{299, 200},
		{300, 201},
		{301, 201},
		{350, 201},
		{399, 201},
		{400, 202},
		{401, 203},
		{450, 252},
		{499, 301},
		{500, 302},
		{501, 302},
		{550, 302},
		{599, 302},
		{600, 303},
		{601, 304},
		{650, 353},
		{699, 402},
	} {
		got := root.IndexOf(test.atomIndex)
		if test.expected != got {
			t.Errorf("IndexOf(%d) returned unexpected atom index.\nExpected: %+v\nGot:      %+v",
				test.atomIndex, test.expected, got)
		}
	}
}

func TestGroupAddTopDown(t *testing.T) {
	root := Group{}
	root.Range = Range{Start: 0, End: 1000}

	root.SubGroups.Add(0, 1000, "R")

	root.SubGroups.Add(100, 200, "A0")
	root.SubGroups.Add(300, 400, "B0")
	root.SubGroups.Add(500, 600, "C0")

	root.SubGroups.Add(120, 180, "A1")
	root.SubGroups.Add(310, 390, "B1")
	root.SubGroups.Add(500, 600, "C1")

	root.SubGroups.Add(140, 160, "A2")
	root.SubGroups.Add(320, 380, "B2")
	root.SubGroups.Add(500, 600, "C2")

	expected := Group{
		Range: Range{Start: 0, End: 1000},
		SubGroups: GroupList{
			Group{
				Range: Range{Start: 0, End: 1000},
				Name:  "R",
				SubGroups: GroupList{
					Group{
						Range: Range{Start: 100, End: 200},
						Name:  "A0",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 120, End: 180},
								Name:  "A1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 140, End: 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{Start: 300, End: 400},
						Name:  "B0",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 310, End: 390},
								Name:  "B1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 320, End: 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{Start: 500, End: 600},
						Name:  "C0",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 500, End: 600},
								Name:  "C1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 500, End: 600},
										Name:  "C2",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(expected, root) {
		t.Errorf("built group was not as expected.\nExpected: %+v\nGot:      %+v",
			expected, root)
	}
}

func TestGroupAddBottomUp(t *testing.T) {
	root := Group{}
	root.Range = Range{Start: 0, End: 1000}

	root.SubGroups.Add(140, 160, "A2")
	root.SubGroups.Add(320, 380, "B2")
	root.SubGroups.Add(500, 600, "C2")

	root.SubGroups.Add(120, 180, "A1")
	root.SubGroups.Add(310, 390, "B1")
	root.SubGroups.Add(500, 600, "C1")

	root.SubGroups.Add(100, 200, "A0")
	root.SubGroups.Add(300, 400, "B0")
	root.SubGroups.Add(500, 600, "C0")

	root.SubGroups.Add(0, 1000, "R")

	expected := Group{
		Range: Range{Start: 0, End: 1000},
		SubGroups: GroupList{
			Group{
				Range: Range{Start: 0, End: 1000},
				Name:  "R",
				SubGroups: GroupList{
					Group{
						Range: Range{Start: 100, End: 200},
						Name:  "A0",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 120, End: 180},
								Name:  "A1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 140, End: 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{Start: 300, End: 400},
						Name:  "B0",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 310, End: 390},
								Name:  "B1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 320, End: 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{Start: 500, End: 600},
						Name:  "C2",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 500, End: 600},
								Name:  "C1",
								SubGroups: GroupList{
									Group{
										Range: Range{Start: 500, End: 600},
										Name:  "C0",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(expected, root) {
		t.Errorf("built group was not as expected.\nExpected: %+v\nGot:      %+v",
			expected, root)
	}
}

func TestGroupAddMixed(t *testing.T) {
	root := Group{}
	root.Range = Range{Start: 0, End: 1000}

	root.SubGroups.Add(100, 500, "A")
	root.SubGroups.Add(400, 500, "C")
	root.SubGroups.Add(200, 500, "B")

	expected := Group{
		Range: Range{Start: 0, End: 1000},
		SubGroups: GroupList{
			Group{
				Range: Range{Start: 100, End: 500},
				Name:  "A",
				SubGroups: GroupList{
					Group{
						Range: Range{Start: 200, End: 500},
						Name:  "B",
						SubGroups: GroupList{
							Group{
								Range: Range{Start: 400, End: 500},
								Name:  "C",
							},
						},
					},
				},
			},
		},
	}

	if !reflect.DeepEqual(expected, root) {
		t.Errorf("built group was not as expected.\nExpected: %+v\nGot:      %+v",
			expected, root)
	}
}
