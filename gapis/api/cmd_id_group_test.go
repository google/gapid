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
			CmdIDRange{0, 100},
			CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				CmdIDRange{100, 200},
			}},
			CmdIDRange{200, 300},
			CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				CmdIDRange{310, 320},
				CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					CmdIDRange{350, 351},
				}},
				CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
						CmdIDRange{360, 362},
					}},
					CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
						CmdIDRange{362, 365},
					}},
				}},
				CmdIDRange{370, 380},
			}},
			CmdIDRange{400, 500},
			CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				CmdIDRange{500, 600},
			}},
			CmdIDRange{600, CmdID(end - 100)},
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
	check(t, "sub group 0 count", 100, root.Spans[1].(CmdIDGroup).Count())
	check(t, "sub group 1 count", 22, root.Spans[3].(CmdIDGroup).Count())
	check(t, "sub group 1.0 count", 1, root.Spans[3].(CmdIDGroup).Spans[1].(CmdIDGroup).Count())
	check(t, "sub group 2 count", 100, root.Spans[5].(CmdIDGroup).Count())
}

func TestGroupIndex(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		index    uint64
		expected CmdIDGroupOrID
	}{
		{0, CmdID(0)},
		{1, CmdID(1)},
		{50, CmdID(50)},
		{100, root.Spans[1].(CmdIDGroup)},
		{101, CmdID(200)},
		{102, CmdID(201)},
		{151, CmdID(250)},
		{200, CmdID(299)},
		{201, root.Spans[3].(CmdIDGroup)},
		{202, CmdID(400)},
		{203, CmdID(401)},
		{252, CmdID(450)},
		{301, CmdID(499)},
		{302, root.Spans[5].(CmdIDGroup)},
		{303, CmdID(600)},
		{304, CmdID(601)},
		{353, CmdID(650)},
		{402, CmdID(699)},
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
	got := CmdIDGroup{}
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
			CmdIDGroup{
				Range: CmdIDRange{0, 1000},
				Name:  "R",
				Spans: Spans{
					CmdIDGroup{
						Range: CmdIDRange{100, 200},
						Name:  "A0",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{120, 180},
								Name:  "A1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{140, 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					CmdIDGroup{
						Range: CmdIDRange{300, 400},
						Name:  "B0",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{310, 390},
								Name:  "B1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{320, 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					CmdIDGroup{
						Range: CmdIDRange{500, 600},
						Name:  "C0",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{500, 600},
								Name:  "C1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{500, 600},
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
	got := CmdIDGroup{}
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
			CmdIDGroup{
				Range: CmdIDRange{0, 1000},
				Name:  "R",
				Spans: Spans{
					CmdIDGroup{
						Range: CmdIDRange{100, 200},
						Name:  "A0",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{120, 180},
								Name:  "A1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{140, 160},
										Name:  "A2",
									},
								},
							},
						},
					},
					CmdIDGroup{
						Range: CmdIDRange{300, 400},
						Name:  "B0",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{310, 390},
								Name:  "B1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{320, 380},
										Name:  "B2",
									},
								},
							},
						},
					},
					CmdIDGroup{
						Range: CmdIDRange{500, 600},
						Name:  "C2",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{500, 600},
								Name:  "C1",
								Spans: Spans{
									CmdIDGroup{
										Range: CmdIDRange{500, 600},
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
	got := CmdIDGroup{}
	got.Range = CmdIDRange{0, 1000}

	got.AddGroup(100, 500, "A")
	got.AddGroup(400, 500, "C")
	got.AddGroup(200, 500, "B")

	expected := CmdIDGroup{
		Range: CmdIDRange{0, 1000},
		Spans: Spans{
			CmdIDGroup{
				Range: CmdIDRange{100, 500},
				Name:  "A",
				Spans: Spans{
					CmdIDGroup{
						Range: CmdIDRange{200, 500},
						Name:  "B",
						Spans: Spans{
							CmdIDGroup{
								Range: CmdIDRange{400, 500},
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

	got.AddAtoms(func(CmdID) bool { return true }, 0)

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 1100}, Spans{
			CmdIDRange{0, 100},
			CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				CmdIDRange{100, 200},
			}},
			CmdIDRange{200, 300},
			CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				CmdIDRange{300, 340},
				CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					CmdIDRange{340, 360},
				}},
				CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
						CmdIDRange{360, 362},
					}},
					CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
						CmdIDRange{362, 365},
					}},
					CmdIDRange{365, 370},
				}},
				CmdIDRange{370, 400},
			}},
			CmdIDRange{400, 500},
			CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				CmdIDRange{500, 600},
			}},
			CmdIDRange{600, 1100},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsSparse(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(1100)

	got.AddAtoms(func(id CmdID) bool { return (id/50)&1 == 0 }, 0)

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 1100}, Spans{
			CmdIDRange{0, 50},
			CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
				CmdIDRange{100, 150},
			}},
			CmdIDRange{200, 250},
			CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
				CmdIDRange{300, 340},
				CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
					CmdIDRange{340, 350},
				}},
				CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
					CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{}},
					CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{}},
				}},
			}},
			CmdIDRange{400, 450},
			CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
				CmdIDRange{500, 550},
			}},
			CmdIDRange{600, 650},
			CmdIDRange{700, 750},
			CmdIDRange{800, 850},
			CmdIDRange{900, 950},
			CmdIDRange{1000, 1050},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestAddAtomsWithSplitting(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(700)

	got.AddAtoms(func(CmdID) bool { return true }, 45)

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 700}, Spans{
			CmdIDGroup{"Sub Group 1", CmdIDRange{0, 45}, Spans{CmdIDRange{0, 45}}},
			CmdIDGroup{"Sub Group 2", CmdIDRange{45, 90}, Spans{CmdIDRange{45, 90}}},
			CmdIDGroup{"Sub Group 3", CmdIDRange{90, 234}, Spans{
				CmdIDRange{90, 100},
				CmdIDGroup{"Sub-group 0", CmdIDRange{100, 200}, Spans{
					CmdIDGroup{"Sub Group 1", CmdIDRange{100, 145}, Spans{CmdIDRange{100, 145}}},
					CmdIDGroup{"Sub Group 2", CmdIDRange{145, 190}, Spans{CmdIDRange{145, 190}}},
					CmdIDGroup{"Sub Group 3", CmdIDRange{190, 200}, Spans{CmdIDRange{190, 200}}},
				}},
				CmdIDRange{200, 234},
			}},
			CmdIDGroup{"Sub Group 4", CmdIDRange{234, 279}, Spans{CmdIDRange{234, 279}}},
			CmdIDGroup{"Sub Group 5", CmdIDRange{279, 423}, Spans{
				CmdIDRange{279, 300},
				CmdIDGroup{"Sub-group 1", CmdIDRange{300, 400}, Spans{
					CmdIDGroup{"Sub Group 1", CmdIDRange{300, 373}, Spans{
						CmdIDRange{300, 340},
						CmdIDGroup{"Sub-group 1.0", CmdIDRange{340, 360}, Spans{
							CmdIDRange{340, 360},
						}},
						CmdIDGroup{"Sub-group 1.1", CmdIDRange{360, 370}, Spans{
							CmdIDGroup{"Sub-group 1.1.0", CmdIDRange{360, 362}, Spans{
								CmdIDRange{360, 362},
							}},
							CmdIDGroup{"Sub-group 1.1.1", CmdIDRange{362, 365}, Spans{
								CmdIDRange{362, 365},
							}},
							CmdIDRange{365, 370},
						}},
						CmdIDRange{370, 373},
					}},
					CmdIDGroup{"Sub Group 2", CmdIDRange{373, 400}, Spans{CmdIDRange{373, 400}}},
				}},
				CmdIDRange{400, 423},
			}},
			CmdIDGroup{"Sub Group 6", CmdIDRange{423, 468}, Spans{CmdIDRange{423, 468}}},
			CmdIDGroup{"Sub Group 7", CmdIDRange{468, 612}, Spans{
				CmdIDRange{468, 500},
				CmdIDGroup{"Sub-group 2", CmdIDRange{500, 600}, Spans{
					CmdIDGroup{"Sub Group 1", CmdIDRange{500, 545}, Spans{CmdIDRange{500, 545}}},
					CmdIDGroup{"Sub Group 2", CmdIDRange{545, 590}, Spans{CmdIDRange{545, 590}}},
					CmdIDGroup{"Sub Group 3", CmdIDRange{590, 600}, Spans{CmdIDRange{590, 600}}},
				}},
				CmdIDRange{600, 612},
			}},
			CmdIDGroup{"Sub Group 8", CmdIDRange{612, 657}, Spans{CmdIDRange{612, 657}}},
			CmdIDGroup{"Sub Group 9", CmdIDRange{657, 700}, Spans{CmdIDRange{657, 700}}},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

func TestSpansSplit(t *testing.T) {
	ctx := log.Testing(t)
	got := CmdIDGroup{
		"root", CmdIDRange{0, 22}, Spans{
			CmdIDRange{0, 3},
			CmdIDRange{3, 4},
			CmdIDRange{4, 7},
			CmdIDRange{7, 9},
			CmdIDRange{9, 11},
			CmdIDRange{11, 14},
			CmdIDRange{14, 15},
			CmdIDGroup{"Child 1", CmdIDRange{15, 16}, Spans{CmdIDRange{15, 16}}},
			CmdIDGroup{"Child 2", CmdIDRange{16, 17}, Spans{CmdIDRange{16, 17}}},
			CmdIDGroup{"Child 3", CmdIDRange{17, 18}, Spans{CmdIDRange{17, 18}}},
			CmdIDGroup{"Child 4", CmdIDRange{18, 19}, Spans{CmdIDRange{18, 19}}},
			CmdIDGroup{"Child 5", CmdIDRange{19, 20}, Spans{CmdIDRange{19, 20}}},
			CmdIDGroup{"Child 6", CmdIDRange{20, 21}, Spans{CmdIDRange{20, 21}}},
			CmdIDGroup{"Child 7", CmdIDRange{21, 22}, Spans{CmdIDRange{21, 22}}},
		},
	}

	got.Spans = got.Spans.split(3)

	expected := CmdIDGroup{
		"root", CmdIDRange{0, 22}, Spans{
			CmdIDGroup{"Sub Group 1", CmdIDRange{0, 3}, Spans{CmdIDRange{0, 3}}},
			CmdIDGroup{"Sub Group 2", CmdIDRange{3, 6}, Spans{CmdIDRange{3, 4}, CmdIDRange{4, 6}}},
			CmdIDGroup{"Sub Group 3", CmdIDRange{6, 9}, Spans{CmdIDRange{6, 7}, CmdIDRange{7, 9}}},
			CmdIDGroup{"Sub Group 4", CmdIDRange{9, 12}, Spans{CmdIDRange{9, 11}, CmdIDRange{11, 12}}},
			CmdIDGroup{"Sub Group 5", CmdIDRange{12, 15}, Spans{CmdIDRange{12, 14}, CmdIDRange{14, 15}}},
			CmdIDGroup{"Sub Group 6", CmdIDRange{15, 18}, Spans{
				CmdIDGroup{"Child 1", CmdIDRange{15, 16}, Spans{CmdIDRange{15, 16}}},
				CmdIDGroup{"Child 2", CmdIDRange{16, 17}, Spans{CmdIDRange{16, 17}}},
				CmdIDGroup{"Child 3", CmdIDRange{17, 18}, Spans{CmdIDRange{17, 18}}},
			}},
			CmdIDGroup{"Sub Group 7", CmdIDRange{18, 21}, Spans{
				CmdIDGroup{"Child 4", CmdIDRange{18, 19}, Spans{CmdIDRange{18, 19}}},
				CmdIDGroup{"Child 5", CmdIDRange{19, 20}, Spans{CmdIDRange{19, 20}}},
				CmdIDGroup{"Child 6", CmdIDRange{20, 21}, Spans{CmdIDRange{20, 21}}},
			}},
			CmdIDGroup{"Sub Group 8", CmdIDRange{21, 22}, Spans{
				CmdIDGroup{"Child 7", CmdIDRange{21, 22}, Spans{CmdIDRange{21, 22}}},
			}},
		},
	}

	assert.With(ctx).That(got).DeepEquals(expected)
}

type idxAndGroupOrID struct {
	idx  uint64
	item CmdIDGroupOrID
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
			{0, CmdID(0)},
			{1, CmdID(1)},
			{2, CmdID(2)},
		}},
		{98, 5, []idxAndGroupOrID{
			{98, CmdID(98)},
			{99, CmdID(99)},
			{100, root.Spans[1].(CmdIDGroup)},
			{101, CmdID(200)},
			{102, CmdID(201)},
		}},
		{199, 5, []idxAndGroupOrID{
			{199, CmdID(298)},
			{200, CmdID(299)},
			{201, root.Spans[3].(CmdIDGroup)},
			{202, CmdID(400)},
			{203, CmdID(401)},
		}},
		{300, 5, []idxAndGroupOrID{
			{300, CmdID(498)},
			{301, CmdID(499)},
			{302, root.Spans[5].(CmdIDGroup)},
			{303, CmdID(600)},
			{304, CmdID(601)},
		}},
		{700, 3, []idxAndGroupOrID{
			{700, CmdID(997)},
			{701, CmdID(998)},
			{702, CmdID(999)},
			{0xdead, nil}, // Not reached
		}},
	} {
		i := 0
		err := root.IterateForwards(test.from, func(childIdx uint64, item CmdIDGroupOrID) error {
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
			{2, CmdID(2)},
			{1, CmdID(1)},
			{0, CmdID(0)},
			{0xdead, nil}, // Not reached
		}},
		{102, 5, []idxAndGroupOrID{
			{102, CmdID(201)},
			{101, CmdID(200)},
			{100, root.Spans[1].(CmdIDGroup)},
			{99, CmdID(99)},
			{98, CmdID(98)},
		}},
		{203, 5, []idxAndGroupOrID{
			{203, CmdID(401)},
			{202, CmdID(400)},
			{201, root.Spans[3].(CmdIDGroup)},
			{200, CmdID(299)},
			{199, CmdID(298)},
		}},
		{304, 5, []idxAndGroupOrID{
			{304, CmdID(601)},
			{303, CmdID(600)},
			{302, root.Spans[5].(CmdIDGroup)},
			{301, CmdID(499)},
			{300, CmdID(498)},
		}},
		{702, 3, []idxAndGroupOrID{
			{702, CmdID(999)},
			{701, CmdID(998)},
			{700, CmdID(997)},
		}},
	} {
		i := 0
		err := root.IterateBackwards(test.from, func(childIdx uint64, item CmdIDGroupOrID) error {
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
	Item    CmdIDGroupOrID
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
			{I(0), CmdID(0)},
			{I(1), CmdID(1)},
			{I(2), CmdID(2)},
		}},
		{I(98), []indicesAndGroupOrID{
			{I(98), CmdID(98)},
			{I(99), CmdID(99)},
			{I(100), root.Spans[1].(CmdIDGroup)},
			{I(100, 0), CmdID(100)},
			{I(100, 1), CmdID(101)},
			{I(100, 2), CmdID(102)},
		}},
		{I(199), []indicesAndGroupOrID{
			{I(199), CmdID(298)},
			{I(200), CmdID(299)},
			{I(201), root.Spans[3].(CmdIDGroup)},
			{I(201, 0), CmdID(310)},
			{I(201, 1), CmdID(311)},
		}},
		{I(201, 8), []indicesAndGroupOrID{
			{I(201, 8), CmdID(318)},
			{I(201, 9), CmdID(319)},
			{I(201, 10), root.Spans[3].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 10, 0), CmdID(350)},
			{I(201, 11), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup)},
			{I(201, 11, 0), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[0].(CmdIDGroup)},
			{I(201, 11, 0, 0), CmdID(360)},
			{I(201, 11, 0, 1), CmdID(361)},
			{I(201, 11, 1), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 11, 1, 0), CmdID(362)},
			{I(201, 11, 1, 1), CmdID(363)},
			{I(201, 11, 1, 2), CmdID(364)},
			{I(201, 12), CmdID(370)},
		}},
		{I(300), []indicesAndGroupOrID{
			{I(300), CmdID(498)},
			{I(301), CmdID(499)},
			{I(302), root.Spans[5].(CmdIDGroup)},
			{I(302, 0), CmdID(500)},
			{I(302, 1), CmdID(501)},
			{I(302, 2), CmdID(502)},
		}},
		{I(700), []indicesAndGroupOrID{
			{I(700), CmdID(997)},
			{I(701), CmdID(998)},
			{I(702), CmdID(999)},
		}},
	} {
		i := 0
		err := root.Traverse(false, test.from, func(indices []uint64, item CmdIDGroupOrID) error {
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
			CmdIDGroup{"Frame 1", CmdIDRange{0, 5}, Spans{
				CmdIDGroup{"Draw 1", CmdIDRange{0, 2}, Spans{
					CmdIDRange{0, 2},
				}},
				CmdIDGroup{"Draw 2", CmdIDRange{2, 4}, Spans{
					CmdIDRange{2, 4},
				}},
				CmdIDRange{4, 5},
			}},
			CmdIDGroup{"Frame 2", CmdIDRange{5, 10}, Spans{
				CmdIDGroup{"Draw 1", CmdIDRange{5, 7}, Spans{
					CmdIDRange{5, 7},
				}},
				CmdIDGroup{"Draw 2", CmdIDRange{7, 9}, Spans{
					CmdIDRange{7, 9},
				}},
				CmdIDRange{9, 10},
			}},
		},
	}

	for ti, test := range []struct {
		root     CmdIDGroup
		from     []uint64
		expected []indicesAndGroupOrID
	}{
		{root, I(), []indicesAndGroupOrID{
			{I(702), CmdID(999)},
			{I(701), CmdID(998)},
			{I(700), CmdID(997)},
		}},
		{root, I(100, 2), []indicesAndGroupOrID{
			{I(100, 2), CmdID(102)},
			{I(100, 1), CmdID(101)},
			{I(100, 0), CmdID(100)},
			{I(100), root.Spans[1].(CmdIDGroup)},
			{I(99), CmdID(99)},
			{I(98), CmdID(98)},
		}},
		{root, I(201, 1), []indicesAndGroupOrID{
			{I(201, 1), CmdID(311)},
			{I(201, 0), CmdID(310)},
			{I(201), root.Spans[3].(CmdIDGroup)},
			{I(200), CmdID(299)},
			{I(199), CmdID(298)},
		}},
		{root, I(201, 13), []indicesAndGroupOrID{
			{I(201, 13), CmdID(371)},
			{I(201, 12), CmdID(370)},
			{I(201, 11, 1, 2), CmdID(364)},
			{I(201, 11, 1, 1), CmdID(363)},
			{I(201, 11, 1, 0), CmdID(362)},
			{I(201, 11, 1), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 11, 0, 1), CmdID(361)},
			{I(201, 11, 0, 0), CmdID(360)},
			{I(201, 11, 0), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[0].(CmdIDGroup)},
			{I(201, 11), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup)},
			{I(201, 10, 0), CmdID(350)},
			{I(201, 10), root.Spans[3].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 9), CmdID(319)},
			{I(201, 8), CmdID(318)},
		}},
		{root, I(201, 11, 1, 1), []indicesAndGroupOrID{
			{I(201, 11, 1, 1), CmdID(363)},
			{I(201, 11, 1, 0), CmdID(362)},
			{I(201, 11, 1), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 11, 0, 1), CmdID(361)},
			{I(201, 11, 0, 0), CmdID(360)},
			{I(201, 11, 0), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup).Spans[0].(CmdIDGroup)},
			{I(201, 11), root.Spans[3].(CmdIDGroup).Spans[2].(CmdIDGroup)},
			{I(201, 10, 0), CmdID(350)},
			{I(201, 10), root.Spans[3].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(201, 9), CmdID(319)},
		}},
		{root, I(302, 2), []indicesAndGroupOrID{
			{I(302, 2), CmdID(502)},
			{I(302, 1), CmdID(501)},
			{I(302, 0), CmdID(500)},
			{I(302), root.Spans[5].(CmdIDGroup)},
			{I(301), CmdID(499)},
			{I(300), CmdID(498)},
		}},
		{root, I(702), []indicesAndGroupOrID{
			{I(702), CmdID(999)},
			{I(701), CmdID(998)},
			{I(700), CmdID(997)},
		}},
		{overflowTest, I(1, 1, 1), []indicesAndGroupOrID{
			{I(1, 1, 1), CmdID(8)},
			{I(1, 1, 0), CmdID(7)},
			{I(1, 1), overflowTest.Spans[1].(CmdIDGroup).Spans[1].(CmdIDGroup)},
			{I(1, 0, 1), CmdID(6)},
			{I(1, 0, 0), CmdID(5)},
			{I(1, 0), overflowTest.Spans[1].(CmdIDGroup).Spans[0].(CmdIDGroup)},
			{I(1), overflowTest.Spans[1].(CmdIDGroup)},
			{I(0, 2), CmdID(4)},
			{I(0, 1, 1), CmdID(3)},
			{I(0, 1, 0), CmdID(2)},
		}},
		// This test should pass, given the previous test (it's a subrange), but
		// it used to cause an unsinged int overflow and thus fail (see 3c90b4c).
		{overflowTest, I(1, 0, 1), []indicesAndGroupOrID{
			{I(1, 0, 1), CmdID(6)},
			{I(1, 0, 0), CmdID(5)},
			{I(1, 0), overflowTest.Spans[1].(CmdIDGroup).Spans[0].(CmdIDGroup)},
			{I(1), overflowTest.Spans[1].(CmdIDGroup)},
			{I(0, 2), CmdID(4)},
			{I(0, 1, 1), CmdID(3)},
			{I(0, 1, 0), CmdID(2)},
		}},
	} {
		i := 0
		err := test.root.Traverse(true, test.from, func(indices []uint64, item CmdIDGroupOrID) error {
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
