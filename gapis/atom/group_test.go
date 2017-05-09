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
	"testing"

	"fmt"

	"github.com/google/gapid/core/assert"
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
 │                └─ [11..20] ──── Atoms [370..379]
 ├─ [202..301] ── Atoms [400..499]
 ├─ [302] ─────── Group 'Sub-group 2' [500..599]
 │                └─ [0..99] ───── Atoms [500..599]
 └─ [303..702] ── Atoms [600..999]`

func buildTestGroup(end uint64) Group {
	return Group{
		"root", Range{0, ID(end)}, Spans{
			Range{0, 100},
			Group{"Sub-group 0", Range{100, 200}, Spans{
				Range{100, 200},
			}},
			Range{200, 300},
			Group{"Sub-group 1", Range{300, 400}, Spans{
				Range{310, 320},
				Group{"Sub-group 1.0", Range{340, 360}, Spans{
					Range{350, 351},
				}},
				Range{370, 380},
			}},
			Range{400, 500},
			Group{"Sub-group 2", Range{500, 600}, Spans{
				Range{500, 600},
			}},
			Range{600, ID(end-100)},
		},
	}
}

func TestGroupFormat(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	assert.For(ctx, "string").ThatString("\n" + fmt.Sprintf("%+v", root)).Equals("\n" + tree)
}

func TestGroupCount(t *testing.T) {
	root := buildTestGroup(1100)

	check(t, "root count", 703, root.Count())
	check(t, "sub group 0 count", 100, root.Spans[1].(Group).Count())
	check(t, "sub group 1 count", 21, root.Spans[3].(Group).Count())
	check(t, "sub group 1.0 count", 1, root.Spans[3].(Group).Spans[1].(Group).Count())
	check(t, "sub group 2 count", 100, root.Spans[5].(Group).Count())
}

func TestGroupIndex(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		index    uint64
		expected GroupOrID
	}{
		{0, ID(0)},
		{1, ID(1)},
		{50, ID(50)},
		{100, root.Spans[1].(Group)},
		{101, ID(200)},
		{102, ID(201)},
		{151, ID(250)},
		{200, ID(299)},
		{201, root.Spans[3].(Group)},
		{202, ID(400)},
		{203, ID(401)},
		{252, ID(450)},
		{301, ID(499)},
		{302, root.Spans[5].(Group)},
		{303, ID(600)},
		{304, ID(601)},
		{353, ID(650)},
		{402, ID(699)},
	} {
		got := root.Index(test.index)
		assert.For(ctx, "root.Index(%v)", test.index).That(got).DeepEquals(test.expected)
	}
}

func TestGroupIndexOf(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		id       ID
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
	got := Group{}
	got.Range = Range{0, 1000}

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

	expected := Group{
		Range: Range{0, 1000},
		Spans: Spans{
			Group{
				Range: Range{0, 1000},
				Name:  "R",
				Spans: Spans{
					Group{
						Range: Range{100, 200},
						Name:  "A0",
						Spans: Spans{
							Group{
								Range: Range{120, 180},
								Name:  "A1",
								Spans: Spans{
									Group{
										Range: Range{140, 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{300, 400},
						Name:  "B0",
						Spans: Spans{
							Group{
								Range: Range{310, 390},
								Name:  "B1",
								Spans: Spans{
									Group{
										Range: Range{320, 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{500, 600},
						Name:  "C0",
						Spans: Spans{
							Group{
								Range: Range{500, 600},
								Name:  "C1",
								Spans: Spans{
									Group{
										Range: Range{500, 600},
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

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddGroupBottomUp(t *testing.T) {
	ctx := log.Testing(t)
	got := Group{}
	got.Range = Range{0, 1000}

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

	expected := Group{
		Range: Range{0, 1000},
		Spans: Spans{
			Group{
				Range: Range{0, 1000},
				Name:  "R",
				Spans: Spans{
					Group{
						Range: Range{100, 200},
						Name:  "A0",
						Spans: Spans{
							Group{
								Range: Range{120, 180},
								Name:  "A1",
								Spans: Spans{
									Group{
										Range: Range{140, 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{300, 400},
						Name:  "B0",
						Spans: Spans{
							Group{
								Range: Range{310, 390},
								Name:  "B1",
								Spans: Spans{
									Group{
										Range: Range{320, 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					Group{
						Range: Range{500, 600},
						Name:  "C2",
						Spans: Spans{
							Group{
								Range: Range{500, 600},
								Name:  "C1",
								Spans: Spans{
									Group{
										Range: Range{500, 600},
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

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddGroupMixed(t *testing.T) {
	ctx := log.Testing(t)
	got := Group{}
	got.Range = Range{0, 1000}

	got.AddGroup(100, 500, "A")
	got.AddGroup(400, 500, "C")
	got.AddGroup(200, 500, "B")

	expected := Group{
		Range: Range{0, 1000},
		Spans: Spans{
			Group{
				Range: Range{100, 500},
				Name:  "A",
				Spans: Spans{
					Group{
						Range: Range{200, 500},
						Name:  "B",
						Spans: Spans{
							Group{
								Range: Range{400, 500},
								Name:  "C",
							},
						},
					},
				},
			},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsFill(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(1100)

	got.AddAtoms(func(ID) bool { return true }, 0)

	expected := Group{
		"root", Range{0, 1100}, Spans{
			Range{0, 100},
			Group{"Sub-group 0", Range{100, 200}, Spans{
				Range{100, 200},
			}},
			Range{200, 300},
			Group{"Sub-group 1", Range{300, 400}, Spans{
				Range{300, 340},
				Group{"Sub-group 1.0", Range{340, 360}, Spans{
					Range{340, 360},
				}},
				Range{360, 400},
			}},
			Range{400, 500},
			Group{"Sub-group 2", Range{500, 600}, Spans{
				Range{500, 600},
			}},
			Range{600, 1100},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsSparse(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(1100)

	got.AddAtoms(func(id ID) bool { return (id/50)&1 == 0 }, 0)

	expected := Group{
		"root", Range{0, 1100}, Spans{
			Range{0, 50},
			Group{"Sub-group 0", Range{100, 200}, Spans{
				Range{100, 150},
			}},
			Range{200, 250},
			Group{"Sub-group 1", Range{300, 400}, Spans{
				Range{300, 340},
				Group{"Sub-group 1.0", Range{340, 360}, Spans{
					Range{340, 350},
				}},
			}},
			Range{400, 450},
			Group{"Sub-group 2", Range{500, 600}, Spans{
				Range{500, 550},
			}},
			Range{600, 650},
			Range{700, 750},
			Range{800, 850},
			Range{900, 950},
			Range{1000, 1050},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsWithSplitting(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(700)

	got.AddAtoms(func(ID) bool { return true }, 45)

	expected := Group{
		"root", Range{0, 700}, Spans{
			Group{"Sub Group 1", Range{0, 45}, Spans{Range{0, 45}}},
			Group{"Sub Group 2", Range{45, 90}, Spans{Range{45, 90}}},
			Group{"Sub Group 3", Range{90, 234}, Spans{
				Range{90, 100},
				Group{"Sub-group 0", Range{100, 200}, Spans{
					Group{"Sub Group 1", Range{100, 145}, Spans{Range{100, 145}}},
					Group{"Sub Group 2", Range{145, 190}, Spans{Range{145, 190}}},
					Group{"Sub Group 3", Range{190, 200}, Spans{Range{190, 200}}},
				}},
				Range{200, 234},
			}},
			Group{"Sub Group 4", Range{234, 279}, Spans{Range{234, 279}}},
			Group{"Sub Group 5", Range{279, 423}, Spans{
				Range{279, 300},
				Group{"Sub-group 1", Range{300, 400}, Spans{
					Group{"Sub Group 1", Range{300, 364}, Spans{
						Range{300, 340},
						Group{"Sub-group 1.0", Range{340, 360}, Spans{
							Range{340, 360},
						}},
						Range{360, 364},
					}},
					Group{"Sub Group 2", Range{364, 400}, Spans{Range{364, 400}}},
				}},
				Range{400, 423},
			}},
			Group{"Sub Group 6", Range{423, 468}, Spans{Range{423, 468}}},
			Group{"Sub Group 7", Range{468, 612}, Spans{
				Range{468, 500},
				Group{"Sub-group 2", Range{500, 600}, Spans{
					Group{"Sub Group 1", Range{500, 545}, Spans{Range{500, 545}}},
					Group{"Sub Group 2", Range{545, 590}, Spans{Range{545, 590}}},
					Group{"Sub Group 3", Range{590, 600}, Spans{Range{590, 600}}},
				}},
				Range{600, 612},
			}},
			Group{"Sub Group 8", Range{612, 657}, Spans{Range{612, 657}}},
			Group{"Sub Group 9", Range{657, 700}, Spans{Range{657, 700}}},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestSpansSplit(t *testing.T) {
	ctx := log.Testing(t)
	got := Group{
		"root", Range{0, 22}, Spans{
			Range{0, 3},
			Range{3, 4},
			Range{4, 7},
			Range{7, 9},
			Range{9, 11},
			Range{11, 14},
			Range{14, 15},
			Group{"Child 1", Range{15, 16}, Spans{Range{15, 16}}},
			Group{"Child 2", Range{16, 17}, Spans{Range{16, 17}}},
			Group{"Child 3", Range{17, 18}, Spans{Range{17, 18}}},
			Group{"Child 4", Range{18, 19}, Spans{Range{18, 19}}},
			Group{"Child 5", Range{19, 20}, Spans{Range{19, 20}}},
			Group{"Child 6", Range{20, 21}, Spans{Range{20, 21}}},
			Group{"Child 7", Range{21, 22}, Spans{Range{21, 22}}},
		},
	}

	got.Spans = got.Spans.split(3)

	expected := Group{
		"root", Range{0, 22}, Spans{
			Group{"Sub Group 1", Range{0, 3}, Spans{Range{0, 3}}},
			Group{"Sub Group 2", Range{3, 6}, Spans{Range{3, 4}, Range{4, 6}}},
			Group{"Sub Group 3", Range{6, 9}, Spans{Range{6, 7}, Range{7, 9}}},
			Group{"Sub Group 4", Range{9, 12}, Spans{Range{9, 11}, Range{11, 12}}},
			Group{"Sub Group 5", Range{12, 15}, Spans{Range{12, 14}, Range{14, 15}}},
			Group{"Sub Group 6", Range{15, 18}, Spans{
				Group{"Child 1", Range{15, 16}, Spans{Range{15, 16}}},
				Group{"Child 2", Range{16, 17}, Spans{Range{16, 17}}},
				Group{"Child 3", Range{17, 18}, Spans{Range{17, 18}}},
			}},
			Group{"Sub Group 7", Range{18, 21}, Spans{
				Group{"Child 4", Range{18, 19}, Spans{Range{18, 19}}},
				Group{"Child 5", Range{19, 20}, Spans{Range{19, 20}}},
				Group{"Child 6", Range{20, 21}, Spans{Range{20, 21}}},
			}},
			Group{"Sub Group 8", Range{21, 22}, Spans{
				Group{"Child 7", Range{21, 22}, Spans{Range{21, 22}}},
			}},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

type idxAndGroupOrID struct {
	idx  uint64
	item GroupOrID
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
			{0, ID(0)},
			{1, ID(1)},
			{2, ID(2)},
		}},
		{98, 5, []idxAndGroupOrID{
			{98, ID(98)},
			{99, ID(99)},
			{100, root.Spans[1].(Group)},
			{101, ID(200)},
			{102, ID(201)},
		}},
		{199, 5, []idxAndGroupOrID{
			{199, ID(298)},
			{200, ID(299)},
			{201, root.Spans[3].(Group)},
			{202, ID(400)},
			{203, ID(401)},
		}},
		{300, 5, []idxAndGroupOrID{
			{300, ID(498)},
			{301, ID(499)},
			{302, root.Spans[5].(Group)},
			{303, ID(600)},
			{304, ID(601)},
		}},
		{700, 3, []idxAndGroupOrID{
			{700, ID(997)},
			{701, ID(998)},
			{702, ID(999)},
			{0xdead, nil}, // Not reached
		}},
	} {
		i := 0
		err := root.IterateForwards(test.from, func(childIdx uint64, item GroupOrID) error {
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
			{2, ID(2)},
			{1, ID(1)},
			{0, ID(0)},
			{0xdead, nil}, // Not reached
		}},
		{102, 5, []idxAndGroupOrID{
			{102, ID(201)},
			{101, ID(200)},
			{100, root.Spans[1].(Group)},
			{99, ID(99)},
			{98, ID(98)},
		}},
		{203, 5, []idxAndGroupOrID{
			{203, ID(401)},
			{202, ID(400)},
			{201, root.Spans[3].(Group)},
			{200, ID(299)},
			{199, ID(298)},
		}},
		{304, 5, []idxAndGroupOrID{
			{304, ID(601)},
			{303, ID(600)},
			{302, root.Spans[5].(Group)},
			{301, ID(499)},
			{300, ID(498)},
		}},
		{702, 3, []idxAndGroupOrID{
			{702, ID(999)},
			{701, ID(998)},
			{700, ID(997)},
		}},
	} {
		i := 0
		err := root.IterateBackwards(test.from, func(childIdx uint64, item GroupOrID) error {
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
	Item    GroupOrID
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
			{I(0), ID(0)},
			{I(1), ID(1)},
			{I(2), ID(2)},
		}},
		{I(98), []indicesAndGroupOrID{
			{I(98), ID(98)},
			{I(99), ID(99)},
			{I(100), root.Spans[1].(Group)},
			{I(100, 0), ID(100)},
			{I(100, 1), ID(101)},
			{I(100, 2), ID(102)},
		}},
		{I(199), []indicesAndGroupOrID{
			{I(199), ID(298)},
			{I(200), ID(299)},
			{I(201), root.Spans[3].(Group)},
			{I(201, 0), ID(310)},
			{I(201, 1), ID(311)},
		}},
		{I(201, 8), []indicesAndGroupOrID{
			{I(201, 8), ID(318)},
			{I(201, 9), ID(319)},
			{I(201, 10), root.Spans[3].(Group).Spans[1].(Group)},
			{I(201, 10, 0), ID(350)},
			{I(201, 11), ID(370)},
			{I(201, 12), ID(371)},
		}},
		{I(300), []indicesAndGroupOrID{
			{I(300), ID(498)},
			{I(301), ID(499)},
			{I(302), root.Spans[5].(Group)},
			{I(302, 0), ID(500)},
			{I(302, 1), ID(501)},
			{I(302, 2), ID(502)},
		}},
		{I(700), []indicesAndGroupOrID{
			{I(700), ID(997)},
			{I(701), ID(998)},
			{I(702), ID(999)},
		}},
	} {
		i := 0
		err := root.Traverse(false, test.from, func(indices []uint64, item GroupOrID) error {
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
	for ti, test := range []struct {
		from     []uint64
		expected []indicesAndGroupOrID
	}{
		{I(), []indicesAndGroupOrID{
			{I(702), ID(999)},
			{I(701), ID(998)},
			{I(700), ID(997)},
		}},
		{I(100, 2), []indicesAndGroupOrID{
			{I(100, 2), ID(102)},
			{I(100, 1), ID(101)},
			{I(100, 0), ID(100)},
			{I(100), root.Spans[1].(Group)},
			{I(99), ID(99)},
			{I(98), ID(98)},
		}},
		{I(201, 1), []indicesAndGroupOrID{
			{I(201, 1), ID(311)},
			{I(201, 0), ID(310)},
			{I(201), root.Spans[3].(Group)},
			{I(200), ID(299)},
			{I(199), ID(298)},
		}},
		{I(201, 12), []indicesAndGroupOrID{
			{I(201, 12), ID(371)},
			{I(201, 11), ID(370)},
			{I(201, 10, 0), ID(350)},
			{I(201, 10), root.Spans[3].(Group).Spans[1].(Group)},
			{I(201, 9), ID(319)},
			{I(201, 8), ID(318)},
		}},
		{I(302, 2), []indicesAndGroupOrID{
			{I(302, 2), ID(502)},
			{I(302, 1), ID(501)},
			{I(302, 0), ID(500)},
			{I(302), root.Spans[5].(Group)},
			{I(301), ID(499)},
			{I(300), ID(498)},
		}},
		{I(702), []indicesAndGroupOrID{
			{I(702), ID(999)},
			{I(701), ID(998)},
			{I(700), ID(997)},
		}},
	} {
		i := 0
		err := root.Traverse(true, test.from, func(indices []uint64, item GroupOrID) error {
			got, expected := indicesAndGroupOrID{indices, item}, test.expected[i]
			assert.For(ctx, "root.Traverse(true, %v) callback %v", test.from, i).That(got).DeepEquals(expected)
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
