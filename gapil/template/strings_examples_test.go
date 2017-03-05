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

package template

import "fmt"

func ExamplestringList_String() {
	fmt.Printf("%#v", stringify("kitty", "caught", "the", "kitten", "in", "the", "kitchen").String())

	// Output:
	// "kittycaughtthekitteninthekitchen"
}

func ExamplestringList_JoinWith() {
	fmt.Printf("%#v", Functions{}.JoinWith("😸", "kitty", "caught", "the", "kitten", "in", "the", "kitchen"))

	// Output:
	// "kitty😸caught😸the😸kitten😸in😸the😸kitchen"
}

func ExamplestringList_SplitOn() {
	fmt.Printf("%#v", Functions{}.SplitOn(" a ", "once a fellow met a fellow in a field of beans"))

	// Output:
	// template.stringList{"once", "fellow met", "fellow in", "field of beans"}
}

func ExamplestringList_SplitUpperCase() {
	fmt.Printf("%#v", Functions{}.SplitUpperCase("friendlyFLEASand fireflies."))

	// Output:
	// template.stringList{"friendly", "F", "L", "E", "A", "S", "and fireflies."}
}

func ExamplestringList_SplitPascalCase() {
	fmt.Printf("%#v", Functions{}.SplitPascalCase("CleanClamsCrammedInCleanCans"))

	// Output:
	// template.stringList{"Clean", "Clams", "Crammed", "In", "Clean", "Cans"}
}

func ExamplestringList_Title() {
	fmt.Printf("%#v", Functions{}.Title("She", "sells", "sea", "shells", "by", "the", "sea", "shore"))

	// Output:
	// template.stringList{"She", "Sells", "Sea", "Shells", "By", "The", "Sea", "Shore"}
}

func ExamplestringList_Untitle() {
	fmt.Printf("%#v", Functions{}.Untitle("Three", "FREE", "Throws."))

	// Output:
	// template.stringList{"three", "fREE", "throws."}
}

func ExamplestringList_Upper() {
	fmt.Printf("%#v", Functions{}.Upper("Zebras", "Zig", "and", "Zebras", "Zag."))

	// Output:
	// template.stringList{"ZEBRAS", "ZIG", "AND", "ZEBRAS", "ZAG."}
}

func ExamplestringList_Lower() {
	fmt.Printf("%#v", Functions{}.Lower("Fresh", "FRIED", "Fish"))

	// Output:
	// template.stringList{"fresh", "fried", "fish"}
}

func ExamplestringList_Contains() {
	fmt.Printf("%#v\n", Functions{}.Contains("lemon", "Lovely", "lemon", "liniment."))
	fmt.Printf("%#v", Functions{}.Contains("lylem", "Lovely", "lemon", "liniment."))

	// Output:
	// true
	// false
}

func ExamplestringList_Replace() {
	fmt.Printf("%#v", Functions{}.Replace("lorry", "leather", "red", "lorry", ",", "yellow", "lorry"))

	// Output:
	// template.stringList{"red", "leather", ",", "yellow", "leather"}
}
