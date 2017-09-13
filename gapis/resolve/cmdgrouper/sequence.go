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

package cmdgrouper

import (
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

// Rule is a single rule in a sequence grouper.
type Rule struct {
	// Pred returns true if the rule passes.
	Pred func(cmd, prev api.Cmd) bool
	// Repeats is true if the rule should repeat until it no longer passes.
	// The rule has to pass at least once for the sequence to complete.
	Repeats bool
	// Optional is true if the rule can be skipped and still have the sequence
	// complete.
	Optional bool
}

// Sequence returns a Grouper that groups commands that match a sequence of
// rules.
func Sequence(name string, rules ...Rule) Grouper {
	return &sequence{name: name, rules: rules}
}

var _ Grouper = &sequence{}

type sequence struct {
	name   string
	rules  []Rule
	rule   int
	passed bool
	prev   api.Cmd
	start  api.CmdID
	groups []Group
}

func (g *sequence) flush(id api.CmdID) {
	if g.rule == len(g.rules) {
		g.groups = append(g.groups, Group{
			Start: g.start,
			End:   id,
			Name:  g.name,
		})
		g.rule, g.passed, g.start = 0, false, id
	}
}

// Process considers the command for inclusion in the group.
func (g *sequence) Process(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.GlobalState) {
	const debug = false

	prev := g.prev
	g.prev = cmd
	if g.start == api.CmdNoID {
		g.start = id
	}

	for {
		g.flush(id)
		rule := g.rules[g.rule]
		passed := rule.Pred(cmd, prev)
		optional := rule.Optional
		repeats := rule.Repeats

		switch {
		case passed && repeats:
			g.passed = true
			return
		case passed && !repeats:
			g.rule++
			g.passed = false
			return
		// --- below failed ---
		case optional, g.passed:
			g.rule++
			g.passed = false
			continue
		default: // failed sequence
			if debug && g.rule > 1 {
				log.W(ctx, "%v: %v failed at rule %v. cmd: %v", id, g.name, g.rule, cmd)
			}
			g.rule, g.passed, g.start = 0, false, api.CmdNoID
			return
		}
	}
}

// Build returns the groups built and resets the state of the grouper.
func (g *sequence) Build(end api.CmdID) []Group {
	g.flush(end)
	out := g.groups
	g.groups = nil
	return out
}
