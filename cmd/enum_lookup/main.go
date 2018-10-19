// Copyright (C) 2018 Google Inc.
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

// The enum_lookup command parses its parameters as decimal and/or hex ints
// and then prints out all the API enums with that value.
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/google/gapid/core/app"
)

var (
	filterAPI     = flag.String("api", "", "Only show enums of the given API")
	filterType    = flag.String("type", "", "Only show enums of the given type")
	showEnums     = flag.Bool("e", true, "Lookup the value as an enum")
	showBitfields = flag.Bool("b", false, "Attempt to expand the value as or'ed bitfield")
)

type found struct {
	val int64
	enum
}

func main() {
	app.ShortHelp = "enum_lookup looks up API enums by value"
	app.Name = "enum_lookup"
	app.Run(run)
}

func run(ctx context.Context) error {
	todo := map[int64]bool{}

	for _, arg := range os.Args {
		arg = strings.ToLower(arg)
		if strings.HasPrefix(arg, "0x") {
			parse(todo, arg[2:], 16)
		} else {
			parse(todo, arg, 10)
			parse(todo, arg, 16)
		}
	}

	if *showEnums {
		lookup(todo, LookupEnum)
	}
	if *showBitfields {
		if *showEnums {
			fmt.Println("\n============= Bitfields =============\n")
		}
		lookup(todo, LookupBitfields)
	}

	return nil
}

func parse(m map[int64]bool, s string, base int) {
	if strings.HasPrefix(s, "-") {
		if val, err := strconv.ParseInt(s, base, 64); err == nil {
			m[val] = true
		}
	} else {
		if val, err := strconv.ParseUint(s, base, 64); err == nil {
			m[int64(val)] = true
		}
	}
}

func lookup(todo map[int64]bool, look func(v int64) []enum) {
	results := []found{}
	maxAPILength, maxTypeLength, maxVal := 0, 0, uint64(0)
	for v := range todo {
		for _, r := range look(v) {
			if filter(r) {
				continue
			}

			results = append(results, found{v, r})
			if l := len(r.API); l > maxAPILength {
				maxAPILength = l
			}
			if l := len(r.Type); l > maxTypeLength {
				maxTypeLength = l
			}
			if l := uint64(v); l > maxVal {
				maxVal = l
			}
		}
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.API == b.API {
			if a.Type == b.Type {
				return a.val < b.val
			} else {
				return a.Type < b.Type
			}
		} else {
			return a.API < b.API
		}
	})

	f := fmt.Sprintf("%%%dd 0x%%0%dx: %%%ds %%%ds  %%s\n",
		int(math.Ceil(math.Log10(float64(maxVal+1)))),
		int(math.Ceil(math.Log2(float64(maxVal+1))/4)),
		maxAPILength, maxTypeLength)
	for _, r := range results {
		fmt.Printf(f, r.val, r.val, r.API, r.Type, r.Name)
	}
}

func filter(e enum) bool {
	return (*filterAPI != "" && !strings.EqualFold(*filterAPI, e.API)) ||
		(*filterType != "" && !strings.EqualFold(*filterType, e.Type))
}
