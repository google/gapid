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
	"context"
	"fmt"
	"time"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/text/note"
)

type (
	TestOrderBy    int
	TestKeyToValue string
	TestStringer   int
	TestPage       struct {
		name     string
		page     note.Page
		raw      string
		brief    string
		normal   string
		detailed string
		unsorted string
		sorted   string
	}
)

func (TestOrderBy) OrderBy() string       { return "Number" }
func (TestOrderBy) String() string        { return "_OrderBy_" }
func (TestKeyToValue) KeyToValue() string { return "⇒" }
func (v TestStringer) String() string {
	switch v {
	case 0:
		return "Stringer"
	default:
		return fmt.Sprintf("stringed(%d)", v)
	}
}

var (
	severity  = note.SectionInfo{Key: "Severity", Relevance: note.Important, Order: 3}
	tag       = note.SectionInfo{Key: "Tag", Relevance: note.Critical, Order: 1}
	text      = note.SectionInfo{Key: "Text", Relevance: note.Critical, Order: 2}
	detail    = note.SectionInfo{Key: "Detail", Relevance: note.Relevant, Order: 4, Multiline: true}
	testPages = []TestPage{{
		name:   "basic message",
		raw:    "Tagged:A Message",
		brief:  "VerySevere:Tagged:A Message",
		normal: "VerySevere:Tagged:A Message:SomeKey=A Value,AnotherKey=42",
		detailed: `VerySevere:Tagged:A Message
    SomeKey    = A Value
    AnotherKey = 42`,
		unsorted: `Severity{"VerySevere"},Tag{"Tagged"},Text{"A Message"},Detail{SomeKey="A Value",AnotherKey=42}` + "",
		sorted:   `Tag{"Tagged"},Text{"A Message"},Severity{"VerySevere"},Detail{AnotherKey=42,SomeKey="A Value"}` + "",
	}, {
		name:   "nested message",
		raw:    "Tagged:A Message",
		brief:  "VerySevere:Tagged:A Message",
		normal: "VerySevere:Tagged:A Message:SomeKey=A Value,AnotherKey=42,Cause⇒{Knowledge:Tasty=Dec  4 14:19:11,More=true}",
		detailed: `VerySevere:Tagged:A Message
    SomeKey    = A Value
    AnotherKey = 42
    Cause      ⇒ Knowledge
        Tasty = Dec  4 14:19:11
        More  = true`,
		unsorted: `Severity{"VerySevere"},Tag{"Tagged"},Text{"A Message"},Detail{SomeKey="A Value",AnotherKey=42,Cause=[Text{"Knowledge"},Detail{Tasty="0000-12-04 14:19:11.691 +0000 UTC",More=true}]}`,
		sorted:   `Tag{"Tagged"},Text{"A Message"},Severity{"VerySevere"},Detail{AnotherKey=42,Cause=[Text{"Knowledge"},Detail{More=true,Tasty="0000-12-04 14:19:11.691 +0000 UTC"}],SomeKey="A Value"}`,
	}, {
		name:     "empty page",
		page:     note.Page{},
		raw:      "",
		brief:    "",
		normal:   "",
		detailed: "",
		unsorted: "",
		sorted:   "",
	}, {
		name:     "empty section",
		page:     note.Page{{Content: []note.Item{}}},
		raw:      "",
		brief:    "",
		normal:   "",
		detailed: "",
		unsorted: "",
		sorted:   "",
	}, {
		name: "value types",
		page: note.Page{{Content: []note.Item{
			{Key: TestStringer(0), Value: TestStringer(4)},
			{Key: TestOrderBy(0), Value: 2},
			{Key: "Err", Value: fault.Const("AnError")},
			{Key: "Z", Value: 3},
			{Key: "A", Value: 4},
		}}},
		raw:      "stringed(4),2,⦕AnError⦖,3,4",
		brief:    "Stringer=stringed(4),_OrderBy_=2,Err=⦕AnError⦖,Z=3,A=4",
		normal:   "Stringer=stringed(4),_OrderBy_=2,Err=⦕AnError⦖,Z=3,A=4",
		detailed: "Stringer=stringed(4),_OrderBy_=2,Err=⦕AnError⦖,Z=3,A=4",
		unsorted: `{Stringer="stringed(4)",_OrderBy_=2,Err="AnError",Z=3,A=4}`,
		sorted:   `{A=4,Err="AnError",_OrderBy_=2,Stringer="stringed(4)",Z=3}`,
	}, {
		name: "bad sorting",
		page: note.Page{{Content: []note.Item{
			{Key: "B", Value: 2},
			{Key: "A", Value: 1},
			{Key: "D", Value: 4},
			{Key: "C", Value: 3},
		}}},
		raw:      "2,1,4,3",
		brief:    "B=2,A=1,D=4,C=3",
		normal:   "B=2,A=1,D=4,C=3",
		detailed: "B=2,A=1,D=4,C=3",
		unsorted: "{B=2,A=1,D=4,C=3}",
		sorted:   "{A=1,B=2,C=3,D=4}",
	}, {
		name: "relevance",
		page: note.Page{{Content: []note.Item{
			{Key: note.Relevant, Value: 1},
			{Key: note.Unknown, Value: 2},
			{Key: note.Irrelevant, Value: 3},
			{Key: note.Important, Value: 4},
			{Key: note.Critical, Value: 5},
			{Key: note.Relevance(-1), Value: 6},
		}}},
		raw:      "1,2,3,4,5,6",
		brief:    "Relevant=1,Unknown=2,Irrelevant=3,Important=4,Critical=5,-1=6",
		normal:   "Relevant=1,Unknown=2,Irrelevant=3,Important=4,Critical=5,-1=6",
		detailed: "Relevant=1,Unknown=2,Irrelevant=3,Important=4,Critical=5,-1=6",
		unsorted: "{Relevant=1,Unknown=2,Irrelevant=3,Important=4,Critical=5,-1=6}",
		sorted:   "{-1=6,Unknown=2,Critical=5,Important=4,Relevant=1,Irrelevant=3}",
	}, {
		name: "simple nested",
		page: note.Page{{Content: []note.Item{
			{Key: "B", Value: 2},
			{Key: "A", Value: 1},
			{Key: 4, Value: note.Page{{Content: []note.Item{
				{Key: 2.1, Value: 21},
				{Key: 1.1, Value: 11},
			}}}},
			{Key: 3, Value: "three"},
		}}},
		raw:      "2,1,{21,11},three",
		brief:    "B=2,A=1,4={2.1=21,1.1=11},3=three",
		normal:   "B=2,A=1,4={2.1=21,1.1=11},3=three",
		detailed: "B=2,A=1,4={2.1=21,1.1=11},3=three",
		unsorted: `{B=2,A=1,4=[{2.1=21,1.1=11}],3="three"}`,
		sorted:   `{3="three",4=[{1.1=11,2.1=21}],A=1,B=2}`,
	}, {
		name: "unindent",
		page: note.Page{{Content: []note.Item{
			{Key: "A", Value: 1},
		}}, {SectionInfo: note.SectionInfo{Relevance: note.Relevant, Multiline: true}, Content: []note.Item{
			{Key: "B", Value: note.Page{{Content: []note.Item{
				{Key: "C", Value: 2},
			}}}},
		}}, {Content: []note.Item{
			{Key: "D", Value: 3},
		}}},
		raw:      "1:3",
		brief:    "A=1:D=3",
		normal:   "A=1:B={C=2}:D=3",
		detailed: "A=1\n    B = C=2\nD=3",
		unsorted: `{A=1},{B=[{C=2}]},{D=3}`,
		sorted:   `{A=1},{B=[{C=2}]},{D=3}`,
	}}
)

func init() {
	ctx := context.Background()
	testTime, _ := time.Parse(time.Stamp, "Dec  4 14:19:11.691")
	page := note.Page{}
	severity.Transcribe(ctx, &page, "VerySevere")
	tag.Transcribe(ctx, &page, "Tagged")
	text.Transcribe(ctx, &page, "A Message")
	page.Append(detail, "SomeKey", "A Value")
	page.Append(detail, "AnotherKey", 42)
	testPages[0].page = page.Clone()
	childPage := note.Page{}
	text.Transcribe(ctx, &childPage, "Knowledge")
	childPage.Append(text, "Empty", nil)
	childPage.Append(detail, "Tasty", testTime)
	childPage.Append(detail, "More", true)
	page.Append(detail, TestKeyToValue("Cause"), childPage)
	testPages[1].page = page
}
