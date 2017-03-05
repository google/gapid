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

package analysis

// Possibility is an enumerator of possibilites.
type Possibility int

const (
	// True represents a logical certanty or true.
	True Possibility = iota

	// Maybe represents the possibility of true or false.
	Maybe

	// False represents a logical certanty of false.
	False

	// Impossible represents a contradiction of certanties (for example
	// True ∩ False)
	Impossible
)

// MaybeTrue returns true iff a is True or Maybe.
func (a Possibility) MaybeTrue() bool {
	return a == True || a == Maybe
}

// MaybeFalse returns true iff a is False or Maybe.
func (a Possibility) MaybeFalse() bool {
	return a == False || a == Maybe
}

// Not returns the logical negation of a.
func (a Possibility) Not() Possibility {
	switch a {
	case False:
		return True
	case True:
		return False
	case Impossible:
		return Impossible
	default:
		return Maybe
	}
}

// And returns the logical-and of a and b.
func (a Possibility) And(b Possibility) Possibility {
	switch {
	case a == Impossible || b == Impossible:
		return Impossible
	case a == False, b == False:
		return False
	case a == Maybe, b == Maybe:
		return Maybe
	default:
		return True
	}
}

// Or returns the logical-or of a and b.
func (a Possibility) Or(b Possibility) Possibility {
	switch {
	case a == Impossible || b == Impossible:
		return Impossible
	case a == True, b == True:
		return True
	case a == Maybe, b == Maybe:
		return Maybe
	default:
		return False
	}
}

// Equals returns the possibility of a equaling b.
func (a Possibility) Equals(b Possibility) Possibility {
	switch {
	case a == Impossible || b == Impossible:
		return Impossible
	case a == Maybe, b == Maybe:
		return Maybe
	default:
		if a == b {
			return True
		}
		return False
	}
}

// Union returns the union of possibile Possibilitys for a and b.
func (a Possibility) Union(b Possibility) Possibility {
	if a == Impossible || b == Impossible {
		return Impossible
	}
	if a.Equals(b) == True {
		return a
	}
	return Maybe
}

// Intersect returns the intersection of possibile Possibilitys for a and b.
func (a Possibility) Intersect(b Possibility) Possibility {
	if a == Impossible || b == Impossible {
		return Impossible
	}
	if a == Maybe {
		if b == Maybe {
			return Maybe
		}
		a, b = b, a
	}
	// a is True or False
	// b is True, False or Maybe
	if b == Maybe || a == b {
		return a
	}
	return Impossible
}

// Difference returns the possibile for v that are not found in o.
func (a Possibility) Difference(b Possibility) Possibility {
	if a == Impossible || b == Impossible {
		return Impossible
	}
	if a == Maybe {
		return b.Not()
	}
	if a == b {
		return Impossible
	}
	return a
}
