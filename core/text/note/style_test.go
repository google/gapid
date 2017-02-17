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

package note_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/text/note"
)

func TestFlag(t *testing.T) {
	assert := assert.To(t)
	chooser := note.DefaultStyle.Chooser()
	for _, test := range []struct {
		style note.Style
		name  string
	}{
		{note.Raw, "Raw"},
		{note.Brief, "Brief"},
		{note.Normal, "Normal"},
		{note.Detailed, "Detailed"},
		{note.JSON, "JSON"},
		{note.CompactJSON, "CompactJSON"},
	} {
		matched := false
		assert.For(test.name).That(test.style.String()).Equals(test.name)
		for _, c := range chooser.Choices {
			if c.String() == test.name {
				matched = true
				var s note.Style
				s.Choose(c)
				assert.For(test.name).That(s.Name).Equals(test.style.Name)
			}
		}
		assert.For("Style %s in choice list", test.name).That(matched).Equals(true)
	}
}

func testStyle(assert assert.Manager, name string, style note.Style, page note.Page, expect string) {
	assert.For(name).Add("Style", style.Name).That(style.Print(page)).Equals(expect)
}

func TestPrint(t *testing.T) {
	assert := assert.To(t)
	for _, test := range testPages {
		testStyle(assert, test.name, note.Raw, test.page, test.raw)
		testStyle(assert, test.name, note.Brief, test.page, test.brief)
		testStyle(assert, test.name, note.Normal, test.page, test.normal)
		testStyle(assert, test.name, note.Detailed, test.page, test.detailed)
	}
}

func TestCanonical(t *testing.T) {
	assert := assert.To(t)
	for _, test := range testPages {
		testStyle(assert, test.name, note.Canonical, test.page, test.unsorted)
		page := test.page.Clone()
		page.Sort()
		testStyle(assert, test.name+" sorted", note.Canonical, page, test.sorted)
	}
}

func TestJSON(t *testing.T) {
	assert := assert.To(t)
	const expect = `[
	{
		"Key": "Severity",
		"Order": 3,
		"Relevance": "Important",
		"Multiline": false,
		"Content": [
			{
				"Key": null,
				"Value": "VerySevere"
			}
		]
	},
	{
		"Key": "Tag",
		"Order": 1,
		"Relevance": "Critical",
		"Multiline": false,
		"Content": [
			{
				"Key": null,
				"Value": "Tagged"
			}
		]
	},
	{
		"Key": "Text",
		"Order": 2,
		"Relevance": "Critical",
		"Multiline": false,
		"Content": [
			{
				"Key": null,
				"Value": "A Message"
			}
		]
	},
	{
		"Key": "Detail",
		"Order": 4,
		"Relevance": "Relevant",
		"Multiline": true,
		"Content": [
			{
				"Key": "SomeKey",
				"Value": "A Value"
			},
			{
				"Key": "AnotherKey",
				"Value": 42
			}
		]
	}
]`
	testStyle(assert, "json", note.JSON, testPages[0].page, expect)
}

func TestCompactJSON(t *testing.T) {
	assert := assert.To(t)
	const expect = `[{"Key":"Severity","Order":3,"Relevance":"Important","Multiline":false,"Content":[{"Key":null,"Value":"VerySevere"}]},{"Key":"Tag","Order":1,"Relevance":"Critical","Multiline":false,"Content":[{"Key":null,"Value":"Tagged"}]},{"Key":"Text","Order":2,"Relevance":"Critical","Multiline":false,"Content":[{"Key":null,"Value":"A Message"}]},{"Key":"Detail","Order":4,"Relevance":"Relevant","Multiline":true,"Content":[{"Key":"SomeKey","Value":"A Value"},{"Key":"AnotherKey","Value":42}]}]`
	testStyle(assert, "json", note.CompactJSON, testPages[0].page, expect)
}

func TestLongValue(t *testing.T) {
	assert := assert.To(t)
	p := note.Page{{Content: []note.Item{{Value: `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`}}}}
	const expect = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed ...`
	testStyle(assert, "json", note.Brief, p, expect)
}

func TestDefault(t *testing.T) {
	assert := assert.To(t)
	note.DefaultStyle = note.Brief
	got := note.Print(testPages[0].page)
	assert.For("before").That(got).Equals(testPages[0].brief)
	note.DefaultStyle = note.Normal
	got = note.Print(testPages[0].page)
	assert.For("after").That(got).Equals(testPages[0].normal)
}
