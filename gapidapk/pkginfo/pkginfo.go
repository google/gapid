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

package pkginfo

import (
	"sort"
)

// Sort sorts the list of packages, activities and actions alphabetically.
func (l *PackageList) Sort() {
	sort.Sort(pkgSorter(l.Packages))
	for _, p := range l.Packages {
		sort.Sort(activitySorter(p.Activities))
		for _, a := range p.Activities {
			sort.Sort(actionSorter(a.Actions))
		}
	}
}

type pkgSorter []*Package

func (l pkgSorter) Len() int           { return len(l) }
func (l pkgSorter) Less(a, b int) bool { return l[a].Name < l[b].Name }
func (l pkgSorter) Swap(a, b int)      { l[a], l[b] = l[b], l[a] }

type activitySorter []*Activity

func (l activitySorter) Len() int           { return len(l) }
func (l activitySorter) Less(a, b int) bool { return l[a].Name < l[b].Name }
func (l activitySorter) Swap(a, b int)      { l[a], l[b] = l[b], l[a] }

type actionSorter []*Action

func (l actionSorter) Len() int           { return len(l) }
func (l actionSorter) Less(a, b int) bool { return l[a].Name < l[b].Name }
func (l actionSorter) Swap(a, b int)      { l[a], l[b] = l[b], l[a] }

// FindByName returns the package in this given the name
func (l *PackageList) FindByName(name string) *Package {
	for _, p := range l.Packages {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// GetIcon returns the package in this given the name
func (l *PackageList) GetIcon(p *Package) []byte {
	if p.Icon >= 0 && p.Icon < int32(len(l.Icons)) {
		return l.Icons[p.Icon]
	}
	return []byte{}
}
