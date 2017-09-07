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

package api

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

func check(t *testing.T, name string, expected, got uint64) {
	if expected != got {
		t.Errorf("%s was not as expected.\nExpected: %d\nGot:      %d", name, expected, got)
	}
}

var tree = `Group 'root' [0..1099]
 ├─ [0..99] ───── Atoms [0..99]
 ├─ [100] ─────── Group 'Sub-group 0' [100..199]
 │                └─ [0..99] ───── Atoms [100..199]
 ├─ [101..200] ── Atoms [200..299]
 ├─ [201] ─────── Group 'Sub-group 1' [300..399]
 │                ├─ [0..9] ────── Atoms [310..319]
 │                ├─ [10] ──────── Group 'Sub-group 1.0' [340..359]
 │                │                └─ [0] ───────── Atoms [350..350]
 │                ├─ [11] ──────── Group 'Sub-group 1.1' [360..369]
 │                │                ├─ [0] ───────── Group 'Sub-group 1.1.0' [360..361]
 │                │                │                └─ [0..1] ────── Atoms [360..361]
 │                │                └─ [1] ───────── Group 'Sub-group 1.1.1' [362..364]
 │                │                                 └─ [0..2] ────── Atoms [362..364]
 │                └─ [12..21] ──── Atoms [370..379]
 ├─ [202..301] ── Atoms [400..499]
 ├─ [302] ─────── Group 'Sub-group 2' [500..599]
 │                └─ [0..99] ───── Atoms [500..599]
 └─ [303..702] ── Atoms [600..999]`

func buildTestGroup(end uint64) CmdIDGroup {
	return CmdIDGroup{
		"root", CmdIDRange{0, CmdID(end)}, Spans{
			&CmdIDRange{0, 100},
			&CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				&CmdIDRange{100, 200},
			}, nil},
			&CmdIDRange{200, 300},
			&CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				&CmdIDRange{310, 320},
				&CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					&CmdIDRange{350, 351},
				}, nil},
				&CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					&CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
						&CmdIDRange{360, 362},
					}, nil},
					&CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
						&CmdIDRange{362, 365},
					}, nil},
				}, nil},
				&CmdIDRange{370, 380},
			}, nil},
			&CmdIDRange{400, 500},
			&CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				&CmdIDRange{500, 600},
			}, nil},
			&CmdIDRange{600, CmdID(end - 100)},
		},
		nil}
}

func TestGroupFormat(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	assert.For(ctx, "string").ThatString("\n" + fmt.Sprintf("%+v", root)).Equals("\n" + tree)
}

func TestGroupCount(t *testing.T) {
	root := buildTestGroup(1100)

	check(t, "root count", 703, root.Count())
	check(t, "sub group 0 count", 100, root.Spans[1].(*CmdIDGroup).Count())
	check(t, "sub group 1 count", 22, root.Spans[3].(*CmdIDGroup).Count())
	check(t, "sub group 1.0 count", 1, root.Spans[3].(*CmdIDGroup).Spans[1].(*CmdIDGroup).Count())
	check(t, "sub group 2 count", 100, root.Spans[5].(*CmdIDGroup).Count())
}

func TestGroupIndex(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		index    uint64
		expected SpanItem
	}{
		{0, SubCmdIdx{0}},
		{1, SubCmdIdx{1}},
		{50, SubCmdIdx{50}},
		{100, *root.Spans[1].(*CmdIDGroup)},
		{101, SubCmdIdx{200}},
		{102, SubCmdIdx{201}},
		{151, SubCmdIdx{250}},
		{200, SubCmdIdx{299}},
		{201, *root.Spans[3].(*CmdIDGroup)},
		{202, SubCmdIdx{400}},
		{203, SubCmdIdx{401}},
		{252, SubCmdIdx{450}},
		{301, SubCmdIdx{499}},
		{302, *root.Spans[5].(*CmdIDGroup)},
		{303, SubCmdIdx{600}},
		{304, SubCmdIdx{601}},
		{353, SubCmdIdx{650}},
		{402, SubCmdIdx{699}},
	} {
		got := root.Index(test.index)
		assert.For(ctx, "root.Index(%v)", test.index).That(got).DeepEquals(test.expected)
	}
}

func TestGroupIndexOf(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		id       CmdID
		expected uint64
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
		got := root.IndexOf(test.id)
		assert.For(ctx, "root.IndexOf(%v)", test.id).That(got).Equals(test.expected)
	}
}

func TestAddGroupTopDown(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{UserData: nil}
	got.Range = CmdIDRange{0, 1000}

	got.AddGroup(0, 1000, "R")

	got.AddGroup(100, 200, "A0")
	got.AddGroup(300, 400, "B0")
	got.AddGroup(500, 600, "C0")

	got.AddGroup(120, 180, "A1")
	got.AddGroup(310, 390, "B1")
	got.AddGroup(500, 600, "C1")

	got.AddGroup(140, 160, "A2")
	got.AddGroup(320, 380, "B2")
	got.AddGroup(500, 600, "C2")

	expected := CmdIDGroup{
		Range: CmdIDRange{0, 1000},
		Spans: Spans{
			&CmdIDGroup{
				Range: CmdIDRange{0, 1000},
				Name:  "R",
				Spans: Spans{
					&CmdIDGroup{
						Range: CmdIDRange{100, 200},
						Name:  "A0",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{120, 180},
								Name:  "A1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{140, 160},
										Name:     "A2",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
					&CmdIDGroup{
						Range: CmdIDRange{300, 400},
						Name:  "B0",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{310, 390},
								Name:  "B1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{320, 380},
										Name:     "B2",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
					&CmdIDGroup{
						Range: CmdIDRange{500, 600},
						Name:  "C2",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{500, 600},
								Name:  "C1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{500, 600},
										Name:     "C0",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
				},
				UserData: nil,
			},
		},
		UserData: nil,
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddGroupBottomUp(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{UserData: nil}
	got.Range = CmdIDRange{0, 1000}

	got.AddGroup(140, 160, "A2")
	got.AddGroup(320, 380, "B2")
	got.AddGroup(500, 600, "C2")

	got.AddGroup(120, 180, "A1")
	got.AddGroup(310, 390, "B1")
	got.AddGroup(500, 600, "C1")

	got.AddGroup(100, 200, "A0")
	got.AddGroup(300, 400, "B0")
	got.AddGroup(500, 600, "C0")

	got.AddGroup(0, 1000, "R")

	expected := CmdIDGroup{
		Range: CmdIDRange{0, 1000},
		Spans: Spans{
			&CmdIDGroup{
				Range: CmdIDRange{0, 1000},
				Name:  "R",
				Spans: Spans{
					&CmdIDGroup{
						Range: CmdIDRange{100, 200},
						Name:  "A0",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{120, 180},
								Name:  "A1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{140, 160},
										Name:     "A2",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
					&CmdIDGroup{
						Range: CmdIDRange{300, 400},
						Name:  "B0",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{310, 390},
								Name:  "B1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{320, 380},
										Name:     "B2",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
					&CmdIDGroup{
						Range: CmdIDRange{500, 600},
						Name:  "C0",
						Spans: Spans{
							&CmdIDGroup{
								Range: CmdIDRange{500, 600},
								Name:  "C1",
								Spans: Spans{
									&CmdIDGroup{
										Range:    CmdIDRange{500, 600},
										Name:     "C2",
										UserData: nil,
									},
								},
								UserData: nil,
							},
						},
						UserData: nil,
					},
				},
				UserData: nil,
			},
		},
		UserData: nil,
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddGroupMixed(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{UserData: nil}
	got.Range = CmdIDRange{0, 1000}

	got.AddGroup(100, 500, "A")
	got.AddGroup(400, 500, "C")
	got.AddGroup(200, 500, "B")

	expected := CmdIDGroup{
		Range: CmdIDRange{0, 1000},
		Spans: Spans{
			&CmdIDGroup{
				Range: CmdIDRange{100, 500},
				Name:  "A",
				Spans: Spans{
					&CmdIDGroup{
						Range: CmdIDRange{200, 500},
						Name:  "B",
						Spans: Spans{
							&CmdIDGroup{
								Range:    CmdIDRange{400, 500},
								Name:     "C",
								UserData: nil,
							},
						},
						UserData: nil,
					},
				},
				UserData: nil,
			},
		},
		UserData: nil,
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func shuffle(r CmdIDRange) []CmdID {
	const magic = 5329857283
	n := r.End - r.Start
	remaining := make([]CmdID, 0, n)
	for i := r.Start; i < r.End; i++ {
		remaining = append(remaining, i)
	}
	out := make([]CmdID, 0, n)
	for len(remaining) > 0 {
		i := magic % len(remaining)
		out = append(out, remaining[i])
		slice.RemoveAt(&remaining, i, 1)
	}
	return out
}

func addAtomsAndCluster(g *CmdIDGroup, maxChildren, maxNeighbours uint64, pred func(id CmdID) bool) {
	for _, id := range shuffle(g.Bounds()) {
		if pred(id) {
			g.AddCommand(id)
		}
	}
	for _, s := range g.Spans {
		if s, ok := s.(*CmdIDGroup); ok {
			addAtomsAndCluster(s, maxChildren, maxNeighbours, pred)
		}
	}
	g.Cluster(maxChildren, maxNeighbours)
}

func TestAddAtomsFill(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(1100)
	addAtomsAndCluster(&got, 0, 0, func(CmdID) bool { return true })

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 1100}, Spans{
			&CmdIDRange{0, 100},
			&CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				&CmdIDRange{100, 200},
			}, nil},
			&CmdIDRange{200, 300},
			&CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				&CmdIDRange{300, 340},
				&CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					&CmdIDRange{340, 360},
				}, nil},
				&CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					&CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
						&CmdIDRange{360, 362},
					}, nil},
					&CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
						&CmdIDRange{362, 365},
					}, nil},
					&CmdIDRange{365, 370},
				}, nil},
				&CmdIDRange{370, 400},
			}, nil},
			&CmdIDRange{400, 500},
			&CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				&CmdIDRange{500, 600},
			}, nil},
			&CmdIDRange{600, 1100},
		},
		nil}

	if !assert.With(ctx).That(got).DeepEquals(expected) {
		fmt.Printf("Got: %+v\n", got)
		fmt.Println()
		fmt.Printf("Want: %+v\n", expected)
	}
}

func TestAddAtomsSparse(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{
		"root", CmdIDRange{0, 1100}, Spans{
			&CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{}, nil},
			&CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				&CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{}, nil},
				&CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					&CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{}, nil},
					&CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{}, nil},
				}, nil},
			}, nil},
			&CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{}, nil},
		}, nil,
	}

	addAtomsAndCluster(&got, 0, 0, func(id CmdID) bool { return (id/50)&1 == 0 })

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 1100}, Spans{
			&CmdIDRange{0, 50},
			&CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				&CmdIDRange{100, 150},
			}, nil},
			&CmdIDRange{200, 250},
			&CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				&CmdIDRange{300, 340},
				&CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					&CmdIDRange{340, 350},
				}, nil},
				&CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					&CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{}, nil},
					&CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{}, nil},
				}, nil},
			}, nil},
			&CmdIDRange{400, 450},
			&CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				&CmdIDRange{500, 550},
			}, nil},
			&CmdIDRange{600, 650},
			&CmdIDRange{700, 750},
			&CmdIDRange{800, 850},
			&CmdIDRange{900, 950},
			&CmdIDRange{1000, 1050},
		},
		nil}

	if !assert.With(ctx).That(got).DeepEquals(expected) {
		fmt.Printf("Got: %+v\n", got)
		fmt.Println()
		fmt.Printf("Want: %+v\n", expected)
	}
}

func TestAddAtomsWithSplitting(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(700)

	addAtomsAndCluster(&got, 45, 0, func(CmdID) bool { return true })

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 700}, Spans{
			&CmdIDGroup{"Sub Group 1", CmdIDRange{0, 45}, Spans{&CmdIDRange{0, 45}}, nil},
			&CmdIDGroup{"Sub Group 2", CmdIDRange{45, 90}, Spans{&CmdIDRange{45, 90}}, nil},
			&CmdIDGroup{"Sub Group 3", CmdIDRange{90, 234}, Spans{
				&CmdIDRange{90, 100},
				&CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
					&CmdIDGroup{"Sub Group 1", CmdIDRange{100, 145}, Spans{&CmdIDRange{100, 145}}, nil},
					&CmdIDGroup{"Sub Group 2", CmdIDRange{145, 190}, Spans{&CmdIDRange{145, 190}}, nil},
					&CmdIDGroup{"Sub Group 3", CmdIDRange{190, 200}, Spans{&CmdIDRange{190, 200}}, nil},
				}, nil},
				&CmdIDRange{200, 234},
			}, nil},
			&CmdIDGroup{"Sub Group 4", CmdIDRange{234, 279}, Spans{&CmdIDRange{234, 279}}, nil},
			&CmdIDGroup{"Sub Group 5", CmdIDRange{279, 423}, Spans{
				&CmdIDRange{279, 300},
				&CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
					&CmdIDGroup{"Sub Group 1", CmdIDRange{300, 373}, Spans{
						&CmdIDRange{300, 340},
						&CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
							&CmdIDRange{340, 360},
						}, nil},
						&CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
							&CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
								&CmdIDRange{360, 362},
							}, nil},
							&CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
								&CmdIDRange{362, 365},
							}, nil},
							&CmdIDRange{365, 370},
						}, nil},
						&CmdIDRange{370, 373},
					}, nil},
					&CmdIDGroup{"Sub Group 2", CmdIDRange{373, 400}, Spans{&CmdIDRange{373, 400}}, nil},
				}, nil},
				&CmdIDRange{400, 423},
			}, nil},
			&CmdIDGroup{"Sub Group 6", CmdIDRange{423, 468}, Spans{&CmdIDRange{423, 468}}, nil},
			&CmdIDGroup{"Sub Group 7", CmdIDRange{468, 612}, Spans{
				&CmdIDRange{468, 500},
				&CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
					&CmdIDGroup{"Sub Group 1", CmdIDRange{500, 545}, Spans{&CmdIDRange{500, 545}}, nil},
					&CmdIDGroup{"Sub Group 2", CmdIDRange{545, 590}, Spans{&CmdIDRange{545, 590}}, nil},
					&CmdIDGroup{"Sub Group 3", CmdIDRange{590, 600}, Spans{&CmdIDRange{590, 600}}, nil},
				}, nil},
				&CmdIDRange{600, 612},
			}, nil},
			&CmdIDGroup{"Sub Group 8", CmdIDRange{612, 657}, Spans{&CmdIDRange{612, 657}}, nil},
			&CmdIDGroup{"Sub Group 9", CmdIDRange{657, 700}, Spans{&CmdIDRange{657, 700}}, nil},
		},
		nil}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsWithNeighbours(t *testing.T) {
	ctx := log.Testing(t)
	{
		got := CmdIDGroup{
			"root", CmdIDRange{0, 52}, Spans{
				&CmdIDGroup{"Child 1", CmdIDRange{10, 20}, Spans{&CmdIDRange{10, 20}}, nil},
				&CmdIDGroup{"Child 2", CmdIDRange{31, 50}, Spans{&CmdIDRange{31, 50}}, nil},
			},
			nil,
		}

		addAtomsAndCluster(&got, 0, 10, func(CmdID) bool { return true })

		expected := CmdIDGroup{
			"root", CmdIDRange{0, 52}, Spans{
				&CmdIDRange{0, 10},
				&CmdIDGroup{"Child 1", CmdIDRange{10, 20}, Spans{&CmdIDRange{10, 20}}, nil},
				&CmdIDGroup{"Sub Group", CmdIDRange{20, 31}, Spans{&CmdIDRange{20, 31}}, nil},
				&CmdIDGroup{"Child 2", CmdIDRange{31, 50}, Spans{&CmdIDRange{31, 50}}, nil},
				&CmdIDRange{50, 52},
			},
			nil,
		}

		assert.With(ctx).That(got).DeepEquals(expected)
	}
}

func TestSpansSplit(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{
		"root", CmdIDRange{0, 22}, Spans{
			&CmdIDRange{0, 3},
			&CmdIDRange{3, 4},
			&CmdIDRange{4, 7},
			&CmdIDRange{7, 9},
			&CmdIDRange{9, 11},
			&CmdIDRange{11, 14},
			&CmdIDRange{14, 15},
			&CmdIDGroup{"Child 1", CmdIDRange{15, 16}, Spans{&CmdIDRange{15, 16}}, nil},
			&CmdIDGroup{"Child 2", CmdIDRange{16, 17}, Spans{&CmdIDRange{16, 17}}, nil},
			&CmdIDGroup{"Child 3", CmdIDRange{17, 18}, Spans{&CmdIDRange{17, 18}}, nil},
			&CmdIDGroup{"Child 4", CmdIDRange{18, 19}, Spans{&CmdIDRange{18, 19}}, nil},
			&CmdIDGroup{"Child 5", CmdIDRange{19, 20}, Spans{&CmdIDRange{19, 20}}, nil},
			&CmdIDGroup{"Child 6", CmdIDRange{20, 21}, Spans{&CmdIDRange{20, 21}}, nil},
			&CmdIDGroup{"Child 7", CmdIDRange{21, 22}, Spans{&CmdIDRange{21, 22}}, nil},
		},
		nil}

	got.Spans = got.Spans.split(3)

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 22}, Spans{
			&CmdIDGroup{"Sub Group 1", CmdIDRange{0, 3}, Spans{&CmdIDRange{0, 3}}, nil},
			&CmdIDGroup{"Sub Group 2", CmdIDRange{3, 6}, Spans{&CmdIDRange{3, 4}, &CmdIDRange{4, 6}}, nil},
			&CmdIDGroup{"Sub Group 3", CmdIDRange{6, 9}, Spans{&CmdIDRange{6, 7}, &CmdIDRange{7, 9}}, nil},
			&CmdIDGroup{"Sub Group 4", CmdIDRange{9, 12}, Spans{&CmdIDRange{9, 11}, &CmdIDRange{11, 12}}, nil},
			&CmdIDGroup{"Sub Group 5", CmdIDRange{12, 15}, Spans{&CmdIDRange{12, 14}, &CmdIDRange{14, 15}}, nil},
			&CmdIDGroup{"Sub Group 6", CmdIDRange{15, 18}, Spans{
				&CmdIDGroup{"Child 1", CmdIDRange{15, 16}, Spans{&CmdIDRange{15, 16}}, nil},
				&CmdIDGroup{"Child 2", CmdIDRange{16, 17}, Spans{&CmdIDRange{16, 17}}, nil},
				&CmdIDGroup{"Child 3", CmdIDRange{17, 18}, Spans{&CmdIDRange{17, 18}}, nil},
			}, nil},
			&CmdIDGroup{"Sub Group 7", CmdIDRange{18, 21}, Spans{
				&CmdIDGroup{"Child 4", CmdIDRange{18, 19}, Spans{&CmdIDRange{18, 19}}, nil},
				&CmdIDGroup{"Child 5", CmdIDRange{19, 20}, Spans{&CmdIDRange{19, 20}}, nil},
				&CmdIDGroup{"Child 6", CmdIDRange{20, 21}, Spans{&CmdIDRange{20, 21}}, nil},
			}, nil},
			&CmdIDGroup{"Sub Group 8", CmdIDRange{21, 22}, Spans{
				&CmdIDGroup{"Child 7", CmdIDRange{21, 22}, Spans{&CmdIDRange{21, 22}}, nil},
			}, nil},
		},
		nil}

	assert.With(ctx).That(got).DeepEquals(expected)
}

type idxAndGroupOrID struct {
	idx  uint64
	item SpanItem
}

const stop = fault.Const("stop")

func TestIterateForwards(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for ti, test := range []struct {
		from     uint64
		count    int
		expected []idxAndGroupOrID
	}{
		{0, 3, []idxAndGroupOrID{
			{0, SubCmdIdx{0}},
			{1, SubCmdIdx{1}},
			{2, SubCmdIdx{2}},
		}},
		{98, 5, []idxAndGroupOrID{
			{98, SubCmdIdx{98}},
			{99, SubCmdIdx{99}},
			{100, *root.Spans[1].(*CmdIDGroup)},
			{101, SubCmdIdx{200}},
			{102, SubCmdIdx{201}},
		}},
		{199, 5, []idxAndGroupOrID{
			{199, SubCmdIdx{298}},
			{200, SubCmdIdx{299}},
			{201, *root.Spans[3].(*CmdIDGroup)},
			{202, SubCmdIdx{400}},
			{203, SubCmdIdx{401}},
		}},
		{300, 5, []idxAndGroupOrID{
			{300, SubCmdIdx{498}},
			{301, SubCmdIdx{499}},
			{302, *root.Spans[5].(*CmdIDGroup)},
			{303, SubCmdIdx{600}},
			{304, SubCmdIdx{601}},
		}},
		{700, 3, []idxAndGroupOrID{
			{700, SubCmdIdx{997}},
			{701, SubCmdIdx{998}},
			{702, SubCmdIdx{999}},
			{0xdead, nil}, // Not reached
		}},
	} {
		i := 0
		err := root.IterateForwards(test.from, func(childIdx uint64, item SpanItem) error {
			got, expected := idxAndGroupOrID{childIdx, item}, test.expected[i]
			assert.For(ctx, "root.IterateForwards(%v) callback %v", test.from, i).That(got).DeepEquals(expected)
			i++
			if i == test.count {
				return stop
			}
			return nil
		})
		if err != stop {
			t.Errorf("Traverse returned %v (%d callbacks) for test %d.", err, i, ti)
		}
	}
}

func TestIterateBackwards(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for ti, test := range []struct {
		from     uint64
		count    int
		expected []idxAndGroupOrID
	}{
		{2, 3, []idxAndGroupOrID{
			{2, SubCmdIdx{2}},
			{1, SubCmdIdx{1}},
			{0, SubCmdIdx{0}},
			{0xdead, nil}, // Not reached
		}},
		{102, 5, []idxAndGroupOrID{
			{102, SubCmdIdx{201}},
			{101, SubCmdIdx{200}},
			{100, *root.Spans[1].(*CmdIDGroup)},
			{99, SubCmdIdx{99}},
			{98, SubCmdIdx{98}},
		}},
		{203, 5, []idxAndGroupOrID{
			{203, SubCmdIdx{401}},
			{202, SubCmdIdx{400}},
			{201, *root.Spans[3].(*CmdIDGroup)},
			{200, SubCmdIdx{299}},
			{199, SubCmdIdx{298}},
		}},
		{304, 5, []idxAndGroupOrID{
			{304, SubCmdIdx{601}},
			{303, SubCmdIdx{600}},
			{302, *root.Spans[5].(*CmdIDGroup)},
			{301, SubCmdIdx{499}},
			{300, SubCmdIdx{498}},
		}},
		{702, 3, []idxAndGroupOrID{
			{702, SubCmdIdx{999}},
			{701, SubCmdIdx{998}},
			{700, SubCmdIdx{997}},
		}},
	} {
		i := 0
		err := root.IterateBackwards(test.from, func(childIdx uint64, item SpanItem) error {
			got, expected := idxAndGroupOrID{childIdx, item}, test.expected[i]
			assert.For(ctx, "root.IterateBackwards(%v) callback %v", test.from, i).That(got).DeepEquals(expected)
			i++
			if i == test.count {
				return stop
			}
			return nil
		})
		if err != stop {
			t.Errorf("Traverse returned %v (%d callbacks) for test %d.", err, i, ti)
		}
	}
}

type indicesAndGroupOrID struct {
	Indices []uint64
	Item    SpanItem
}

func I(v ...uint64) []uint64 { return v }

func TestTraverseForwards(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for ti, test := range []struct {
		from     []uint64
		expected []indicesAndGroupOrID
	}{
		{I(), []indicesAndGroupOrID{
			{I(0), SubCmdIdx{0}},
			{I(1), SubCmdIdx{1}},
			{I(2), SubCmdIdx{2}},
		}},
		{I(98), []indicesAndGroupOrID{
			{I(98), SubCmdIdx{98}},
			{I(99), SubCmdIdx{99}},
			{I(100), *root.Spans[1].(*CmdIDGroup)},
			{I(100, 0), SubCmdIdx{100}},
			{I(100, 1), SubCmdIdx{101}},
			{I(100, 2), SubCmdIdx{102}},
		}},
		{I(199), []indicesAndGroupOrID{
			{I(199), SubCmdIdx{298}},
			{I(200), SubCmdIdx{299}},
			{I(201), *root.Spans[3].(*CmdIDGroup)},
			{I(201, 0), SubCmdIdx{310}},
			{I(201, 1), SubCmdIdx{311}},
		}},
		{I(201, 8), []indicesAndGroupOrID{
			{I(201, 8), SubCmdIdx{318}},
			{I(201, 9), SubCmdIdx{319}},
			{I(201, 10), *root.Spans[3].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 10, 0), SubCmdIdx{350}},
			{I(201, 11), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup)},
			{I(201, 11, 0), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[0].(*CmdIDGroup)},
			{I(201, 11, 0, 0), SubCmdIdx{360}},
			{I(201, 11, 0, 1), SubCmdIdx{361}},
			{I(201, 11, 1), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 11, 1, 0), SubCmdIdx{362}},
			{I(201, 11, 1, 1), SubCmdIdx{363}},
			{I(201, 11, 1, 2), SubCmdIdx{364}},
			{I(201, 12), SubCmdIdx{370}},
		}},
		{I(300), []indicesAndGroupOrID{
			{I(300), SubCmdIdx{498}},
			{I(301), SubCmdIdx{499}},
			{I(302), *root.Spans[5].(*CmdIDGroup)},
			{I(302, 0), SubCmdIdx{500}},
			{I(302, 1), SubCmdIdx{501}},
			{I(302, 2), SubCmdIdx{502}},
		}},
		{I(700), []indicesAndGroupOrID{
			{I(700), SubCmdIdx{997}},
			{I(701), SubCmdIdx{998}},
			{I(702), SubCmdIdx{999}},
		}},
	} {
		i := 0
		err := root.Traverse(false, test.from, func(indices []uint64, item SpanItem) error {
			got, expected := indicesAndGroupOrID{indices, item}, test.expected[i]
			assert.For(ctx, "root.Traverse(false, %v) callback %v", test.from, i).That(got).DeepEquals(expected)
			i++
			if i == len(test.expected) {
				return stop
			}
			return nil
		})
		if err != stop {
			t.Errorf("Traverse returned %v (%d callbacks) for test %d.", err, i, ti)
		}
	}
}

func TestTraverseBackwards(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	root.Name = "testGroup"
	overflowTest := CmdIDGroup{
		"overflowTest", CmdIDRange{0, 10}, Spans{
			&CmdIDGroup{"Frame 1", CmdIDRange{0, 5}, Spans{
				&CmdIDGroup{"Draw 1", CmdIDRange{0, 2}, Spans{
					&CmdIDRange{0, 2},
				}, nil},
				&CmdIDGroup{"Draw 2", CmdIDRange{2, 4}, Spans{
					&CmdIDRange{2, 4},
				}, nil},
				&CmdIDRange{4, 5},
			}, nil},
			&CmdIDGroup{"Frame 2", CmdIDRange{5, 10}, Spans{
				&CmdIDGroup{"Draw 1", CmdIDRange{5, 7}, Spans{
					&CmdIDRange{5, 7},
				}, nil},
				&CmdIDGroup{"Draw 2", CmdIDRange{7, 9}, Spans{
					&CmdIDRange{7, 9},
				}, nil},
				&CmdIDRange{9, 10},
			}, nil},
		},
		nil}

	for ti, test := range []struct {
		root     CmdIDGroup
		from     []uint64
		expected []indicesAndGroupOrID
	}{
		{root, I(), []indicesAndGroupOrID{
			{I(702), SubCmdIdx{999}},
			{I(701), SubCmdIdx{998}},
			{I(700), SubCmdIdx{997}},
		}},
		{root, I(100, 2), []indicesAndGroupOrID{
			{I(100, 2), SubCmdIdx{102}},
			{I(100, 1), SubCmdIdx{101}},
			{I(100, 0), SubCmdIdx{100}},
			{I(100), *root.Spans[1].(*CmdIDGroup)},
			{I(99), SubCmdIdx{99}},
			{I(98), SubCmdIdx{98}},
		}},
		{root, I(201, 1), []indicesAndGroupOrID{
			{I(201, 1), SubCmdIdx{311}},
			{I(201, 0), SubCmdIdx{310}},
			{I(201), *root.Spans[3].(*CmdIDGroup)},
			{I(200), SubCmdIdx{299}},
			{I(199), SubCmdIdx{298}},
		}},
		{root, I(201, 13), []indicesAndGroupOrID{
			{I(201, 13), SubCmdIdx{371}},
			{I(201, 12), SubCmdIdx{370}},
			{I(201, 11, 1, 2), SubCmdIdx{364}},
			{I(201, 11, 1, 1), SubCmdIdx{363}},
			{I(201, 11, 1, 0), SubCmdIdx{362}},
			{I(201, 11, 1), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 11, 0, 1), SubCmdIdx{361}},
			{I(201, 11, 0, 0), SubCmdIdx{360}},
			{I(201, 11, 0), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[0].(*CmdIDGroup)},
			{I(201, 11), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup)},
			{I(201, 10, 0), SubCmdIdx{350}},
			{I(201, 10), *root.Spans[3].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 9), SubCmdIdx{319}},
			{I(201, 8), SubCmdIdx{318}},
		}},
		{root, I(201, 11, 1, 1), []indicesAndGroupOrID{
			{I(201, 11, 1, 1), SubCmdIdx{363}},
			{I(201, 11, 1, 0), SubCmdIdx{362}},
			{I(201, 11, 1), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 11, 0, 1), SubCmdIdx{361}},
			{I(201, 11, 0, 0), SubCmdIdx{360}},
			{I(201, 11, 0), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup).Spans[0].(*CmdIDGroup)},
			{I(201, 11), *root.Spans[3].(*CmdIDGroup).Spans[2].(*CmdIDGroup)},
			{I(201, 10, 0), SubCmdIdx{350}},
			{I(201, 10), *root.Spans[3].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(201, 9), SubCmdIdx{319}},
		}},
		{root, I(302, 2), []indicesAndGroupOrID{
			{I(302, 2), SubCmdIdx{502}},
			{I(302, 1), SubCmdIdx{501}},
			{I(302, 0), SubCmdIdx{500}},
			{I(302), *root.Spans[5].(*CmdIDGroup)},
			{I(301), SubCmdIdx{499}},
			{I(300), SubCmdIdx{498}},
		}},
		{root, I(702), []indicesAndGroupOrID{
			{I(702), SubCmdIdx{999}},
			{I(701), SubCmdIdx{998}},
			{I(700), SubCmdIdx{997}},
		}},
		{overflowTest, I(1, 1, 1), []indicesAndGroupOrID{
			{I(1, 1, 1), SubCmdIdx{8}},
			{I(1, 1, 0), SubCmdIdx{7}},
			{I(1, 1), *overflowTest.Spans[1].(*CmdIDGroup).Spans[1].(*CmdIDGroup)},
			{I(1, 0, 1), SubCmdIdx{6}},
			{I(1, 0, 0), SubCmdIdx{5}},
			{I(1, 0), *overflowTest.Spans[1].(*CmdIDGroup).Spans[0].(*CmdIDGroup)},
			{I(1), *overflowTest.Spans[1].(*CmdIDGroup)},
			{I(0, 2), SubCmdIdx{4}},
			{I(0, 1, 1), SubCmdIdx{3}},
			{I(0, 1, 0), SubCmdIdx{2}},
		}},
		// This test should pass, given the previous test (it's a subrange), but
		// it used to cause an unsinged int overflow and thus fail (see 3c90b4c).
		{overflowTest, I(1, 0, 1), []indicesAndGroupOrID{
			{I(1, 0, 1), SubCmdIdx{6}},
			{I(1, 0, 0), SubCmdIdx{5}},
			{I(1, 0), *overflowTest.Spans[1].(*CmdIDGroup).Spans[0].(*CmdIDGroup)},
			{I(1), *overflowTest.Spans[1].(*CmdIDGroup)},
			{I(0, 2), SubCmdIdx{4}},
			{I(0, 1, 1), SubCmdIdx{3}},
			{I(0, 1, 0), SubCmdIdx{2}},
		}},
	} {
		i := 0
		err := test.root.Traverse(true, test.from, func(indices []uint64, item SpanItem) error {
			got, expected := indicesAndGroupOrID{indices, item}, test.expected[i]
			assert.For(ctx, "%s.Traverse(true, %v) callback %v", test.root.Name, test.from, i).That(got).DeepEquals(expected)
			i++
			if i == len(test.expected) {
				return stop
			}
			return nil
		})
		if err != stop {
			t.Errorf("Traverse returned %v (%d callbacks) for test %d.", err, i, ti)
		}
	}
}
