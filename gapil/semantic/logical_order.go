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

package semantic

import "fmt"

// LogicalOrder is a bitfield describing whether a statement, block or
// subroutine belongs before (pre) or after (post) the command's fence.
// Some statements and blocks straddle the fence, in which case both the pre and
// post bits will be set.
type LogicalOrder int

const (
	// Resolved indicates that the order has been set.
	Resolved = LogicalOrder(1)
	// Pre represents a statement before a fence.
	Pre = LogicalOrder(2)
	// Post represents a statement after a fence.
	Post = LogicalOrder(4)
)

// Resolved returns true if the logical order has the Resolved bit set.
func (o LogicalOrder) Resolved() bool { return (o & Resolved) != 0 }

// Pre returns true if the logical order has the Pre bit set.
func (o LogicalOrder) Pre() bool { return (o & Pre) != 0 }

// Post returns true if the logical order has the Post bit set.
func (o LogicalOrder) Post() bool { return (o & Post) != 0 }

func (o LogicalOrder) String() string {
	switch o {
	case 0:
		return "Unresolved"
	case Resolved:
		return "No-fence"
	case Resolved | Pre:
		return "Pre-fence"
	case Resolved | Post:
		return "Post-fence"
	case Resolved | Pre | Post:
		return "Contains-fence"
	default:
		return fmt.Sprintf("Unknown(%v)", int(o))
	}
}
