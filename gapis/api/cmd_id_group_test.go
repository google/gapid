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

package api_test

import (
	"fmt"
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/data/slice"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

func check(t *testing.T, name string, expected, got uint64) {
	if expected != got {
		t.Errorf("%s was not as expected.\nExpected: %d\nGot:      %d", name, expected, got)
	}
}

var tree = `Group 'root' [0..1099]
 ├─ [0..99] ───── Commands [0..99]
 ├─ [100] ─────── Group 'Sub-group 0' [100..199]
 │                └─ [0..99] ───── Commands [100..199]
 ├─ [101..200] ── Commands [200..299]
 ├─ [201] ─────── Group 'Sub-group 1' [300..399]
 │                ├─ [0..9] ────── Commands [310..319]
 │                ├─ [10] ──────── Group 'Sub-group 1.0' [340..359]
 │                │                └─ [0] ───────── Commands [350..350]
 │                ├─ [11] ──────── Group 'Sub-group 1.1' [360..369]
 │                │                ├─ [0] ───────── Group 'Sub-group 1.1.0' [360..361]
 │                │                │                └─ [0..1] ────── Commands [360..361]
 │                │                └─ [1] ───────── Group 'Sub-group 1.1.1' [362..364]
 │                │                                 └─ [0..2] ────── Commands [362..364]
 │                └─ [12..21] ──── Commands [370..379]
 ├─ [202..301] ── Commands [400..499]
 ├─ [302] ─────── Group 'Sub-group 2' [500..599]
 │                └─ [0..99] ───── Commands [500..599]
 └─ [303..702] ── Commands [600..999]`

func buildTestGroup(end uint64) api.CmdIDGroup {
	return api.CmdIDGroup{
		"root", api.CmdIDRange{0, api.CmdID(end)}, api.Spans{
			&api.CmdIDRange{0, 100},
			&api.CmdIDGroup{"Sub-group 0", api.CmdIDRange{100, 200}, api.Spans{
				&api.CmdIDRange{100, 200},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{200, 300},
			&api.CmdIDGroup{"Sub-group 1", api.CmdIDRange{300, 400}, api.Spans{
				&api.CmdIDRange{310, 320},
				&api.CmdIDGroup{"Sub-group 1.0", api.CmdIDRange{340, 360}, api.Spans{
					&api.CmdIDRange{350, 351},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Sub-group 1.1", api.CmdIDRange{360, 370}, api.Spans{
					&api.CmdIDGroup{"Sub-group 1.1.0", api.CmdIDRange{360, 362}, api.Spans{
						&api.CmdIDRange{360, 362},
					}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub-group 1.1.1", api.CmdIDRange{362, 365}, api.Spans{
						&api.CmdIDRange{362, 365},
					}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{370, 380},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{400, 500},
			&api.CmdIDGroup{"Sub-group 2", api.CmdIDRange{500, 600}, api.Spans{
				&api.CmdIDRange{500, 600},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{600, api.CmdID(end - 100)},
		},
		[]api.SubCmdIdx{}, nil}
}

func TestGroupFormat(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	assert.For(ctx, "string").ThatString("\n" + fmt.Sprintf("%+v", root)).Equals("\n" + tree)
}

func TestGroupCount(t *testing.T) {
	root := buildTestGroup(1100)

	check(t, "root count", 703, root.Count())
	check(t, "sub group 0 count", 100, root.Spans[1].(*api.CmdIDGroup).Count())
	check(t, "sub group 1 count", 22, root.Spans[3].(*api.CmdIDGroup).Count())
	check(t, "sub group 1.0 count", 1, root.Spans[3].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup).Count())
	check(t, "sub group 2 count", 100, root.Spans[5].(*api.CmdIDGroup).Count())
}

func TestGroupIndex(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		index    uint64
		expected api.SpanItem
	}{
		{0, api.SubCmdIdx{0}},
		{1, api.SubCmdIdx{1}},
		{50, api.SubCmdIdx{50}},
		{100, *root.Spans[1].(*api.CmdIDGroup)},
		{101, api.SubCmdIdx{200}},
		{102, api.SubCmdIdx{201}},
		{151, api.SubCmdIdx{250}},
		{200, api.SubCmdIdx{299}},
		{201, *root.Spans[3].(*api.CmdIDGroup)},
		{202, api.SubCmdIdx{400}},
		{203, api.SubCmdIdx{401}},
		{252, api.SubCmdIdx{450}},
		{301, api.SubCmdIdx{499}},
		{302, *root.Spans[5].(*api.CmdIDGroup)},
		{303, api.SubCmdIdx{600}},
		{304, api.SubCmdIdx{601}},
		{353, api.SubCmdIdx{650}},
		{402, api.SubCmdIdx{699}},
	} {
		got := root.Index(test.index)
		assert.For(ctx, "root.Index(%v)", test.index).That(got).DeepEquals(test.expected)
	}
}

func TestGroupIndexOf(t *testing.T) {
	ctx := log.Testing(t)
	root := buildTestGroup(1100)
	for _, test := range []struct {
		id       api.CmdID
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
	got := api.CmdIDGroup{ExperimentableCmds: []api.SubCmdIdx{}, UserData: nil}
	got.Range = api.CmdIDRange{0, 1000}

	got.AddGroup(0, 1000, "R", []api.SubCmdIdx{})

	got.AddGroup(100, 200, "A0", []api.SubCmdIdx{})
	got.AddGroup(300, 400, "B0", []api.SubCmdIdx{})
	got.AddGroup(500, 600, "C0", []api.SubCmdIdx{})

	got.AddGroup(120, 180, "A1", []api.SubCmdIdx{})
	got.AddGroup(310, 390, "B1", []api.SubCmdIdx{})
	got.AddGroup(500, 600, "C1", []api.SubCmdIdx{})

	got.AddGroup(140, 160, "A2", []api.SubCmdIdx{{150, 0, 5}})
	got.AddGroup(320, 380, "B2", []api.SubCmdIdx{{360, 0, 6}})
	got.AddGroup(500, 600, "C2", []api.SubCmdIdx{{550, 0, 7}})

	expected := api.CmdIDGroup{
		Range: api.CmdIDRange{0, 1000},
		Spans: api.Spans{
			&api.CmdIDGroup{
				Range: api.CmdIDRange{0, 1000},
				Name:  "R",
				Spans: api.Spans{
					&api.CmdIDGroup{
						Range: api.CmdIDRange{100, 200},
						Name:  "A0",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{120, 180},
								Name:  "A1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{140, 160},
										Name:               "A2",
										ExperimentableCmds: []api.SubCmdIdx{{150, 0, 5}},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
					&api.CmdIDGroup{
						Range: api.CmdIDRange{300, 400},
						Name:  "B0",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{310, 390},
								Name:  "B1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{320, 380},
										Name:               "B2",
										ExperimentableCmds: []api.SubCmdIdx{{360, 0, 6}},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
					&api.CmdIDGroup{
						Range: api.CmdIDRange{500, 600},
						Name:  "C2",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{500, 600},
								Name:  "C1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{500, 600},
										Name:               "C0",
										ExperimentableCmds: []api.SubCmdIdx{},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{{550, 0, 7}},
						UserData:           nil,
					},
				},
				ExperimentableCmds: []api.SubCmdIdx{},
				UserData:           nil,
			},
		},
		ExperimentableCmds: []api.SubCmdIdx{},
		UserData:           nil,
	}

	assert.For(ctx, "got").That(got).DeepEquals(expected)
}

func TestAddGroupBottomUp(t *testing.T) {
	ctx := log.Testing(t)
	got := api.CmdIDGroup{ExperimentableCmds: []api.SubCmdIdx{}, UserData: nil}
	got.Range = api.CmdIDRange{0, 1000}

	got.AddGroup(140, 160, "A2", []api.SubCmdIdx{{150, 0, 5}})
	got.AddGroup(320, 380, "B2", []api.SubCmdIdx{{360, 0, 6}})
	got.AddGroup(500, 600, "C2", []api.SubCmdIdx{{550, 0, 7}})

	got.AddGroup(120, 180, "A1", []api.SubCmdIdx{})
	got.AddGroup(310, 390, "B1", []api.SubCmdIdx{})
	got.AddGroup(500, 600, "C1", []api.SubCmdIdx{})

	got.AddGroup(100, 200, "A0", []api.SubCmdIdx{})
	got.AddGroup(300, 400, "B0", []api.SubCmdIdx{})
	got.AddGroup(500, 600, "C0", []api.SubCmdIdx{})

	got.AddGroup(0, 1000, "R", []api.SubCmdIdx{})

	expected := api.CmdIDGroup{
		Range: api.CmdIDRange{0, 1000},
		Spans: api.Spans{
			&api.CmdIDGroup{
				Range: api.CmdIDRange{0, 1000},
				Name:  "R",
				Spans: api.Spans{
					&api.CmdIDGroup{
						Range: api.CmdIDRange{100, 200},
						Name:  "A0",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{120, 180},
								Name:  "A1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{140, 160},
										Name:               "A2",
										ExperimentableCmds: []api.SubCmdIdx{{150, 0, 5}},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
					&api.CmdIDGroup{
						Range: api.CmdIDRange{300, 400},
						Name:  "B0",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{310, 390},
								Name:  "B1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{320, 380},
										Name:               "B2",
										ExperimentableCmds: []api.SubCmdIdx{{360, 0, 6}},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
					&api.CmdIDGroup{
						Range: api.CmdIDRange{500, 600},
						Name:  "C0",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range: api.CmdIDRange{500, 600},
								Name:  "C1",
								Spans: api.Spans{
									&api.CmdIDGroup{
										Range:              api.CmdIDRange{500, 600},
										Name:               "C2",
										ExperimentableCmds: []api.SubCmdIdx{{550, 0, 7}},
										UserData:           nil,
									},
								},
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
				},
				ExperimentableCmds: []api.SubCmdIdx{},
				UserData:           nil,
			},
		},
		ExperimentableCmds: []api.SubCmdIdx{},
		UserData:           nil,
	}

	assert.For(ctx, "got").That(got).DeepEquals(expected)
}

func TestAddGroupMixed(t *testing.T) {
	ctx := log.Testing(t)
	got := api.CmdIDGroup{ExperimentableCmds: []api.SubCmdIdx{}, UserData: nil}
	got.Range = api.CmdIDRange{0, 1000}

	got.AddGroup(100, 500, "A", []api.SubCmdIdx{})
	got.AddGroup(400, 500, "C", []api.SubCmdIdx{})
	got.AddGroup(200, 500, "B", []api.SubCmdIdx{})

	expected := api.CmdIDGroup{
		Range: api.CmdIDRange{0, 1000},
		Spans: api.Spans{
			&api.CmdIDGroup{
				Range: api.CmdIDRange{100, 500},
				Name:  "A",
				Spans: api.Spans{
					&api.CmdIDGroup{
						Range: api.CmdIDRange{200, 500},
						Name:  "B",
						Spans: api.Spans{
							&api.CmdIDGroup{
								Range:              api.CmdIDRange{400, 500},
								Name:               "C",
								ExperimentableCmds: []api.SubCmdIdx{},
								UserData:           nil,
							},
						},
						ExperimentableCmds: []api.SubCmdIdx{},
						UserData:           nil,
					},
				},
				ExperimentableCmds: []api.SubCmdIdx{},
				UserData:           nil,
			},
		},
		ExperimentableCmds: []api.SubCmdIdx{},
		UserData:           nil,
	}

	assert.For(ctx, "got").That(got).DeepEquals(expected)
}

func shuffle(r api.CmdIDRange) []api.CmdID {
	const magic = 5329857283
	n := r.End - r.Start
	remaining := make([]api.CmdID, 0, n)
	for i := r.Start; i < r.End; i++ {
		remaining = append(remaining, i)
	}
	out := make([]api.CmdID, 0, n)
	for len(remaining) > 0 {
		i := magic % len(remaining)
		out = append(out, remaining[i])
		slice.RemoveAt(&remaining, i, 1)
	}
	return out
}

func addCommandsAndCluster(g *api.CmdIDGroup, maxChildren, maxNeighbours uint64, pred func(id api.CmdID) bool) {
	for _, id := range shuffle(g.Bounds()) {
		if pred(id) {
			g.AddCommand(id)
		}
	}
	for _, s := range g.Spans {
		if s, ok := s.(*api.CmdIDGroup); ok {
			addCommandsAndCluster(s, maxChildren, maxNeighbours, pred)
		}
	}
	g.Cluster(maxChildren, maxNeighbours)
}

func TestAddCommandsFill(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(1100)
	addCommandsAndCluster(&got, 0, 0, func(api.CmdID) bool { return true })

	expected := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 1100}, api.Spans{
			&api.CmdIDRange{0, 100},
			&api.CmdIDGroup{"Sub-group 0", api.CmdIDRange{100, 200}, api.Spans{
				&api.CmdIDRange{100, 200},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{200, 300},
			&api.CmdIDGroup{"Sub-group 1", api.CmdIDRange{300, 400}, api.Spans{
				&api.CmdIDRange{300, 340},
				&api.CmdIDGroup{"Sub-group 1.0", api.CmdIDRange{340, 360}, api.Spans{
					&api.CmdIDRange{340, 360},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Sub-group 1.1", api.CmdIDRange{360, 370}, api.Spans{
					&api.CmdIDGroup{"Sub-group 1.1.0", api.CmdIDRange{360, 362}, api.Spans{
						&api.CmdIDRange{360, 362},
					}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub-group 1.1.1", api.CmdIDRange{362, 365}, api.Spans{
						&api.CmdIDRange{362, 365},
					}, []api.SubCmdIdx{}, nil},
					&api.CmdIDRange{365, 370},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{370, 400},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{400, 500},
			&api.CmdIDGroup{"Sub-group 2", api.CmdIDRange{500, 600}, api.Spans{
				&api.CmdIDRange{500, 600},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{600, 1100},
		},
		[]api.SubCmdIdx{}, nil}

	if !assert.For(ctx, "got").That(got).DeepEquals(expected) {
		fmt.Printf("Got: %+v\n", got)
		fmt.Println()
		fmt.Printf("Want: %+v\n", expected)
	}
}

func TestAddCommandsSparse(t *testing.T) {
	ctx := log.Testing(t)
	got := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 1100}, api.Spans{
			&api.CmdIDGroup{"Sub-group 0", api.CmdIDRange{100, 200}, api.Spans{}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub-group 1", api.CmdIDRange{300, 400}, api.Spans{
				&api.CmdIDGroup{"Sub-group 1.0", api.CmdIDRange{340, 360}, api.Spans{}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Sub-group 1.1", api.CmdIDRange{360, 370}, api.Spans{
					&api.CmdIDGroup{"Sub-group 1.1.0", api.CmdIDRange{360, 362}, api.Spans{}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub-group 1.1.1", api.CmdIDRange{362, 365}, api.Spans{}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub-group 2", api.CmdIDRange{500, 600}, api.Spans{}, []api.SubCmdIdx{}, nil},
		}, []api.SubCmdIdx{}, nil,
	}

	addCommandsAndCluster(&got, 0, 0, func(id api.CmdID) bool { return (id/50)&1 == 0 })

	expected := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 1100}, api.Spans{
			&api.CmdIDRange{0, 50},
			&api.CmdIDGroup{"Sub-group 0", api.CmdIDRange{100, 200}, api.Spans{
				&api.CmdIDRange{100, 150},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{200, 250},
			&api.CmdIDGroup{"Sub-group 1", api.CmdIDRange{300, 400}, api.Spans{
				&api.CmdIDRange{300, 340},
				&api.CmdIDGroup{"Sub-group 1.0", api.CmdIDRange{340, 360}, api.Spans{
					&api.CmdIDRange{340, 350},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Sub-group 1.1", api.CmdIDRange{360, 370}, api.Spans{
					&api.CmdIDGroup{"Sub-group 1.1.0", api.CmdIDRange{360, 362}, api.Spans{}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub-group 1.1.1", api.CmdIDRange{362, 365}, api.Spans{}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{400, 450},
			&api.CmdIDGroup{"Sub-group 2", api.CmdIDRange{500, 600}, api.Spans{
				&api.CmdIDRange{500, 550},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDRange{600, 650},
			&api.CmdIDRange{700, 750},
			&api.CmdIDRange{800, 850},
			&api.CmdIDRange{900, 950},
			&api.CmdIDRange{1000, 1050},
		},
		[]api.SubCmdIdx{}, nil}

	if !assert.For(ctx, "got").That(got).DeepEquals(expected) {
		fmt.Printf("Got: %+v\n", got)
		fmt.Println()
		fmt.Printf("Want: %+v\n", expected)
	}
}

func TestAddCommandsWithSplitting(t *testing.T) {
	ctx := log.Testing(t)
	got := buildTestGroup(700)

	addCommandsAndCluster(&got, 45, 0, func(api.CmdID) bool { return true })

	expected := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 700}, api.Spans{
			&api.CmdIDGroup{"Sub Group 1", api.CmdIDRange{0, 45}, api.Spans{&api.CmdIDRange{0, 45}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 2", api.CmdIDRange{45, 90}, api.Spans{&api.CmdIDRange{45, 90}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 3", api.CmdIDRange{90, 234}, api.Spans{
				&api.CmdIDRange{90, 100},
				&api.CmdIDGroup{"Sub-group 0", api.CmdIDRange{100, 200}, api.Spans{
					&api.CmdIDGroup{"Sub Group 1", api.CmdIDRange{100, 145}, api.Spans{&api.CmdIDRange{100, 145}}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub Group 2", api.CmdIDRange{145, 190}, api.Spans{&api.CmdIDRange{145, 190}}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub Group 3", api.CmdIDRange{190, 200}, api.Spans{&api.CmdIDRange{190, 200}}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{200, 234},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 4", api.CmdIDRange{234, 279}, api.Spans{&api.CmdIDRange{234, 279}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 5", api.CmdIDRange{279, 423}, api.Spans{
				&api.CmdIDRange{279, 300},
				&api.CmdIDGroup{"Sub-group 1", api.CmdIDRange{300, 400}, api.Spans{
					&api.CmdIDGroup{"Sub Group 1", api.CmdIDRange{300, 373}, api.Spans{
						&api.CmdIDRange{300, 340},
						&api.CmdIDGroup{"Sub-group 1.0", api.CmdIDRange{340, 360}, api.Spans{
							&api.CmdIDRange{340, 360},
						}, []api.SubCmdIdx{}, nil},
						&api.CmdIDGroup{"Sub-group 1.1", api.CmdIDRange{360, 370}, api.Spans{
							&api.CmdIDGroup{"Sub-group 1.1.0", api.CmdIDRange{360, 362}, api.Spans{
								&api.CmdIDRange{360, 362},
							}, []api.SubCmdIdx{}, nil},
							&api.CmdIDGroup{"Sub-group 1.1.1", api.CmdIDRange{362, 365}, api.Spans{
								&api.CmdIDRange{362, 365},
							}, []api.SubCmdIdx{}, nil},
							&api.CmdIDRange{365, 370},
						}, []api.SubCmdIdx{}, nil},
						&api.CmdIDRange{370, 373},
					}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub Group 2", api.CmdIDRange{373, 400}, api.Spans{&api.CmdIDRange{373, 400}}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{400, 423},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 6", api.CmdIDRange{423, 468}, api.Spans{&api.CmdIDRange{423, 468}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 7", api.CmdIDRange{468, 612}, api.Spans{
				&api.CmdIDRange{468, 500},
				&api.CmdIDGroup{"Sub-group 2", api.CmdIDRange{500, 600}, api.Spans{
					&api.CmdIDGroup{"Sub Group 1", api.CmdIDRange{500, 545}, api.Spans{&api.CmdIDRange{500, 545}}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub Group 2", api.CmdIDRange{545, 590}, api.Spans{&api.CmdIDRange{545, 590}}, []api.SubCmdIdx{}, nil},
					&api.CmdIDGroup{"Sub Group 3", api.CmdIDRange{590, 600}, api.Spans{&api.CmdIDRange{590, 600}}, []api.SubCmdIdx{}, nil},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{600, 612},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 8", api.CmdIDRange{612, 657}, api.Spans{&api.CmdIDRange{612, 657}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 9", api.CmdIDRange{657, 700}, api.Spans{&api.CmdIDRange{657, 700}}, []api.SubCmdIdx{}, nil},
		},
		[]api.SubCmdIdx{}, nil}

	assert.For(ctx, "got").That(got).DeepEquals(expected)
}

func TestAddCommandsWithNeighbours(t *testing.T) {
	ctx := log.Testing(t)
	{
		got := api.CmdIDGroup{
			"root", api.CmdIDRange{0, 52}, api.Spans{
				&api.CmdIDGroup{"Child 1", api.CmdIDRange{10, 20}, api.Spans{&api.CmdIDRange{10, 20}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 2", api.CmdIDRange{31, 50}, api.Spans{&api.CmdIDRange{31, 50}}, []api.SubCmdIdx{}, nil},
			},
			[]api.SubCmdIdx{},
			nil,
		}

		addCommandsAndCluster(&got, 0, 10, func(api.CmdID) bool { return true })

		expected := api.CmdIDGroup{
			"root", api.CmdIDRange{0, 52}, api.Spans{
				&api.CmdIDRange{0, 10},
				&api.CmdIDGroup{"Child 1", api.CmdIDRange{10, 20}, api.Spans{&api.CmdIDRange{10, 20}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Sub Group", api.CmdIDRange{20, 31}, api.Spans{&api.CmdIDRange{20, 31}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 2", api.CmdIDRange{31, 50}, api.Spans{&api.CmdIDRange{31, 50}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{50, 52},
			},
			[]api.SubCmdIdx{},
			nil,
		}

		assert.For(ctx, "got").That(got).DeepEquals(expected)
	}
}

func TestSpansSplit(t *testing.T) {
	ctx := log.Testing(t)
	got := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 22}, api.Spans{
			&api.CmdIDRange{0, 3},
			&api.CmdIDRange{3, 4},
			&api.CmdIDRange{4, 7},
			&api.CmdIDRange{7, 9},
			&api.CmdIDRange{9, 11},
			&api.CmdIDRange{11, 14},
			&api.CmdIDRange{14, 15},
			&api.CmdIDGroup{"Child 1", api.CmdIDRange{15, 16}, api.Spans{&api.CmdIDRange{15, 16}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 2", api.CmdIDRange{16, 17}, api.Spans{&api.CmdIDRange{16, 17}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 3", api.CmdIDRange{17, 18}, api.Spans{&api.CmdIDRange{17, 18}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 4", api.CmdIDRange{18, 19}, api.Spans{&api.CmdIDRange{18, 19}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 5", api.CmdIDRange{19, 20}, api.Spans{&api.CmdIDRange{19, 20}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 6", api.CmdIDRange{20, 21}, api.Spans{&api.CmdIDRange{20, 21}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Child 7", api.CmdIDRange{21, 22}, api.Spans{&api.CmdIDRange{21, 22}}, []api.SubCmdIdx{}, nil},
		},
		[]api.SubCmdIdx{},
		nil}

	got.Spans = got.Spans.Split(3)

	expected := api.CmdIDGroup{
		"root", api.CmdIDRange{0, 22}, api.Spans{
			&api.CmdIDGroup{"Sub Group 1", api.CmdIDRange{0, 3}, api.Spans{&api.CmdIDRange{0, 3}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 2", api.CmdIDRange{3, 6}, api.Spans{&api.CmdIDRange{3, 4}, &api.CmdIDRange{4, 6}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 3", api.CmdIDRange{6, 9}, api.Spans{&api.CmdIDRange{6, 7}, &api.CmdIDRange{7, 9}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 4", api.CmdIDRange{9, 12}, api.Spans{&api.CmdIDRange{9, 11}, &api.CmdIDRange{11, 12}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 5", api.CmdIDRange{12, 15}, api.Spans{&api.CmdIDRange{12, 14}, &api.CmdIDRange{14, 15}}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 6", api.CmdIDRange{15, 18}, api.Spans{
				&api.CmdIDGroup{"Child 1", api.CmdIDRange{15, 16}, api.Spans{&api.CmdIDRange{15, 16}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 2", api.CmdIDRange{16, 17}, api.Spans{&api.CmdIDRange{16, 17}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 3", api.CmdIDRange{17, 18}, api.Spans{&api.CmdIDRange{17, 18}}, []api.SubCmdIdx{}, nil},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 7", api.CmdIDRange{18, 21}, api.Spans{
				&api.CmdIDGroup{"Child 4", api.CmdIDRange{18, 19}, api.Spans{&api.CmdIDRange{18, 19}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 5", api.CmdIDRange{19, 20}, api.Spans{&api.CmdIDRange{19, 20}}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Child 6", api.CmdIDRange{20, 21}, api.Spans{&api.CmdIDRange{20, 21}}, []api.SubCmdIdx{}, nil},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Sub Group 8", api.CmdIDRange{21, 22}, api.Spans{
				&api.CmdIDGroup{"Child 7", api.CmdIDRange{21, 22}, api.Spans{&api.CmdIDRange{21, 22}}, []api.SubCmdIdx{}, nil},
			}, []api.SubCmdIdx{}, nil},
		},
		[]api.SubCmdIdx{}, nil}

	assert.For(ctx, "got").That(got).DeepEquals(expected)
}

type idxAndGroupOrID struct {
	idx  uint64
	item api.SpanItem
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
			{0, api.SubCmdIdx{0}},
			{1, api.SubCmdIdx{1}},
			{2, api.SubCmdIdx{2}},
		}},
		{98, 5, []idxAndGroupOrID{
			{98, api.SubCmdIdx{98}},
			{99, api.SubCmdIdx{99}},
			{100, *root.Spans[1].(*api.CmdIDGroup)},
			{101, api.SubCmdIdx{200}},
			{102, api.SubCmdIdx{201}},
		}},
		{199, 5, []idxAndGroupOrID{
			{199, api.SubCmdIdx{298}},
			{200, api.SubCmdIdx{299}},
			{201, *root.Spans[3].(*api.CmdIDGroup)},
			{202, api.SubCmdIdx{400}},
			{203, api.SubCmdIdx{401}},
		}},
		{300, 5, []idxAndGroupOrID{
			{300, api.SubCmdIdx{498}},
			{301, api.SubCmdIdx{499}},
			{302, *root.Spans[5].(*api.CmdIDGroup)},
			{303, api.SubCmdIdx{600}},
			{304, api.SubCmdIdx{601}},
		}},
		{700, 3, []idxAndGroupOrID{
			{700, api.SubCmdIdx{997}},
			{701, api.SubCmdIdx{998}},
			{702, api.SubCmdIdx{999}},
			{0xdead, nil}, // Not reached
		}},
	} {
		i := 0
		err := root.IterateForwards(test.from, func(childIdx uint64, item api.SpanItem) error {
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
			{2, api.SubCmdIdx{2}},
			{1, api.SubCmdIdx{1}},
			{0, api.SubCmdIdx{0}},
			{0xdead, nil}, // Not reached
		}},
		{102, 5, []idxAndGroupOrID{
			{102, api.SubCmdIdx{201}},
			{101, api.SubCmdIdx{200}},
			{100, *root.Spans[1].(*api.CmdIDGroup)},
			{99, api.SubCmdIdx{99}},
			{98, api.SubCmdIdx{98}},
		}},
		{203, 5, []idxAndGroupOrID{
			{203, api.SubCmdIdx{401}},
			{202, api.SubCmdIdx{400}},
			{201, *root.Spans[3].(*api.CmdIDGroup)},
			{200, api.SubCmdIdx{299}},
			{199, api.SubCmdIdx{298}},
		}},
		{304, 5, []idxAndGroupOrID{
			{304, api.SubCmdIdx{601}},
			{303, api.SubCmdIdx{600}},
			{302, *root.Spans[5].(*api.CmdIDGroup)},
			{301, api.SubCmdIdx{499}},
			{300, api.SubCmdIdx{498}},
		}},
		{702, 3, []idxAndGroupOrID{
			{702, api.SubCmdIdx{999}},
			{701, api.SubCmdIdx{998}},
			{700, api.SubCmdIdx{997}},
		}},
	} {
		i := 0
		err := root.IterateBackwards(test.from, func(childIdx uint64, item api.SpanItem) error {
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
	Item    api.SpanItem
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
			{I(0), api.SubCmdIdx{0}},
			{I(1), api.SubCmdIdx{1}},
			{I(2), api.SubCmdIdx{2}},
		}},
		{I(98), []indicesAndGroupOrID{
			{I(98), api.SubCmdIdx{98}},
			{I(99), api.SubCmdIdx{99}},
			{I(100), *root.Spans[1].(*api.CmdIDGroup)},
			{I(100, 0), api.SubCmdIdx{100}},
			{I(100, 1), api.SubCmdIdx{101}},
			{I(100, 2), api.SubCmdIdx{102}},
		}},
		{I(199), []indicesAndGroupOrID{
			{I(199), api.SubCmdIdx{298}},
			{I(200), api.SubCmdIdx{299}},
			{I(201), *root.Spans[3].(*api.CmdIDGroup)},
			{I(201, 0), api.SubCmdIdx{310}},
			{I(201, 1), api.SubCmdIdx{311}},
		}},
		{I(201, 8), []indicesAndGroupOrID{
			{I(201, 8), api.SubCmdIdx{318}},
			{I(201, 9), api.SubCmdIdx{319}},
			{I(201, 10), *root.Spans[3].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 10, 0), api.SubCmdIdx{350}},
			{I(201, 11), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup)},
			{I(201, 11, 0), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[0].(*api.CmdIDGroup)},
			{I(201, 11, 0, 0), api.SubCmdIdx{360}},
			{I(201, 11, 0, 1), api.SubCmdIdx{361}},
			{I(201, 11, 1), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 11, 1, 0), api.SubCmdIdx{362}},
			{I(201, 11, 1, 1), api.SubCmdIdx{363}},
			{I(201, 11, 1, 2), api.SubCmdIdx{364}},
			{I(201, 12), api.SubCmdIdx{370}},
		}},
		{I(300), []indicesAndGroupOrID{
			{I(300), api.SubCmdIdx{498}},
			{I(301), api.SubCmdIdx{499}},
			{I(302), *root.Spans[5].(*api.CmdIDGroup)},
			{I(302, 0), api.SubCmdIdx{500}},
			{I(302, 1), api.SubCmdIdx{501}},
			{I(302, 2), api.SubCmdIdx{502}},
		}},
		{I(700), []indicesAndGroupOrID{
			{I(700), api.SubCmdIdx{997}},
			{I(701), api.SubCmdIdx{998}},
			{I(702), api.SubCmdIdx{999}},
		}},
	} {
		i := 0
		err := root.Traverse(false, test.from, func(indices []uint64, item api.SpanItem) error {
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
	overflowTest := api.CmdIDGroup{
		"overflowTest", api.CmdIDRange{0, 10}, api.Spans{
			&api.CmdIDGroup{"Frame 1", api.CmdIDRange{0, 5}, api.Spans{
				&api.CmdIDGroup{"Draw 1", api.CmdIDRange{0, 2}, api.Spans{
					&api.CmdIDRange{0, 2},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Draw 2", api.CmdIDRange{2, 4}, api.Spans{
					&api.CmdIDRange{2, 4},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{4, 5},
			}, []api.SubCmdIdx{}, nil},
			&api.CmdIDGroup{"Frame 2", api.CmdIDRange{5, 10}, api.Spans{
				&api.CmdIDGroup{"Draw 1", api.CmdIDRange{5, 7}, api.Spans{
					&api.CmdIDRange{5, 7},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDGroup{"Draw 2", api.CmdIDRange{7, 9}, api.Spans{
					&api.CmdIDRange{7, 9},
				}, []api.SubCmdIdx{}, nil},
				&api.CmdIDRange{9, 10},
			}, []api.SubCmdIdx{}, nil},
		},
		[]api.SubCmdIdx{}, nil}

	for ti, test := range []struct {
		root     api.CmdIDGroup
		from     []uint64
		expected []indicesAndGroupOrID
	}{
		{root, I(), []indicesAndGroupOrID{
			{I(702), api.SubCmdIdx{999}},
			{I(701), api.SubCmdIdx{998}},
			{I(700), api.SubCmdIdx{997}},
		}},
		{root, I(100, 2), []indicesAndGroupOrID{
			{I(100, 2), api.SubCmdIdx{102}},
			{I(100, 1), api.SubCmdIdx{101}},
			{I(100, 0), api.SubCmdIdx{100}},
			{I(100), *root.Spans[1].(*api.CmdIDGroup)},
			{I(99), api.SubCmdIdx{99}},
			{I(98), api.SubCmdIdx{98}},
		}},
		{root, I(201, 1), []indicesAndGroupOrID{
			{I(201, 1), api.SubCmdIdx{311}},
			{I(201, 0), api.SubCmdIdx{310}},
			{I(201), *root.Spans[3].(*api.CmdIDGroup)},
			{I(200), api.SubCmdIdx{299}},
			{I(199), api.SubCmdIdx{298}},
		}},
		{root, I(201, 13), []indicesAndGroupOrID{
			{I(201, 13), api.SubCmdIdx{371}},
			{I(201, 12), api.SubCmdIdx{370}},
			{I(201, 11, 1, 2), api.SubCmdIdx{364}},
			{I(201, 11, 1, 1), api.SubCmdIdx{363}},
			{I(201, 11, 1, 0), api.SubCmdIdx{362}},
			{I(201, 11, 1), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 11, 0, 1), api.SubCmdIdx{361}},
			{I(201, 11, 0, 0), api.SubCmdIdx{360}},
			{I(201, 11, 0), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[0].(*api.CmdIDGroup)},
			{I(201, 11), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup)},
			{I(201, 10, 0), api.SubCmdIdx{350}},
			{I(201, 10), *root.Spans[3].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 9), api.SubCmdIdx{319}},
			{I(201, 8), api.SubCmdIdx{318}},
		}},
		{root, I(201, 11, 1, 1), []indicesAndGroupOrID{
			{I(201, 11, 1, 1), api.SubCmdIdx{363}},
			{I(201, 11, 1, 0), api.SubCmdIdx{362}},
			{I(201, 11, 1), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 11, 0, 1), api.SubCmdIdx{361}},
			{I(201, 11, 0, 0), api.SubCmdIdx{360}},
			{I(201, 11, 0), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup).Spans[0].(*api.CmdIDGroup)},
			{I(201, 11), *root.Spans[3].(*api.CmdIDGroup).Spans[2].(*api.CmdIDGroup)},
			{I(201, 10, 0), api.SubCmdIdx{350}},
			{I(201, 10), *root.Spans[3].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(201, 9), api.SubCmdIdx{319}},
		}},
		{root, I(302, 2), []indicesAndGroupOrID{
			{I(302, 2), api.SubCmdIdx{502}},
			{I(302, 1), api.SubCmdIdx{501}},
			{I(302, 0), api.SubCmdIdx{500}},
			{I(302), *root.Spans[5].(*api.CmdIDGroup)},
			{I(301), api.SubCmdIdx{499}},
			{I(300), api.SubCmdIdx{498}},
		}},
		{root, I(702), []indicesAndGroupOrID{
			{I(702), api.SubCmdIdx{999}},
			{I(701), api.SubCmdIdx{998}},
			{I(700), api.SubCmdIdx{997}},
		}},
		{overflowTest, I(1, 1, 1), []indicesAndGroupOrID{
			{I(1, 1, 1), api.SubCmdIdx{8}},
			{I(1, 1, 0), api.SubCmdIdx{7}},
			{I(1, 1), *overflowTest.Spans[1].(*api.CmdIDGroup).Spans[1].(*api.CmdIDGroup)},
			{I(1, 0, 1), api.SubCmdIdx{6}},
			{I(1, 0, 0), api.SubCmdIdx{5}},
			{I(1, 0), *overflowTest.Spans[1].(*api.CmdIDGroup).Spans[0].(*api.CmdIDGroup)},
			{I(1), *overflowTest.Spans[1].(*api.CmdIDGroup)},
			{I(0, 2), api.SubCmdIdx{4}},
			{I(0, 1, 1), api.SubCmdIdx{3}},
			{I(0, 1, 0), api.SubCmdIdx{2}},
		}},
		// This test should pass, given the previous test (it's a subrange), but
		// it used to cause an unsinged int overflow and thus fail (see 3c90b4c).
		{overflowTest, I(1, 0, 1), []indicesAndGroupOrID{
			{I(1, 0, 1), api.SubCmdIdx{6}},
			{I(1, 0, 0), api.SubCmdIdx{5}},
			{I(1, 0), *overflowTest.Spans[1].(*api.CmdIDGroup).Spans[0].(*api.CmdIDGroup)},
			{I(1), *overflowTest.Spans[1].(*api.CmdIDGroup)},
			{I(0, 2), api.SubCmdIdx{4}},
			{I(0, 1, 1), api.SubCmdIdx{3}},
			{I(0, 1, 0), api.SubCmdIdx{2}},
		}},
	} {
		i := 0
		err := test.root.Traverse(true, test.from, func(indices []uint64, item api.SpanItem) error {
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
