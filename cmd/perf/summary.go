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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
)

var (
	flagLevel int64
)

func init() {
	verb := &app.Verb{
		Name:       "summary",
		ShortHelp:  "Summarizes metrics from a perfz file or compares two perfz files",
		Run:        summaryVerb,
		ShortUsage: "<perfz> [perfz]",
	}
	verb.Flags.Raw.Int64Var(&flagLevel, "level", 1, "amount of information to show [1..3]")
	app.AddVerb(verb)
}

func shouldIncludeInDiff(diffLevel int64, field reflect.StructField) bool {
	diffLevelStr := field.Tag.Get("diff")
	structDiffLevel := int64(1)
	if diffLevelStr == "ignore" {
		structDiffLevel = math.MaxInt32
	} else if diffLevelStr != "" {
		var err error
		structDiffLevel, err = strconv.ParseInt(diffLevelStr, 10, 64)
		if err != nil {
			return false
		}
	}
	return diffLevel >= structDiffLevel
}

func diff(diffLevel int64, a reflect.Value, b reflect.Value, differs []interface{}) (interface{}, error) {
	if a.IsValid() && !b.IsValid() {
		return "(only left value present)", nil
	} else if b.IsValid() && !a.IsValid() {
		return "(only right value present)", nil
	}

	if a.CanInterface() && b.CanInterface() {
		t1 := a.Type()
		for _, differ := range differs {
			differVal := reflect.ValueOf(differ)
			if t1 == differVal.Type().In(0) {
				vs := differVal.Call([]reflect.Value{a, b})
				return vs[0].Interface(), nil
			}
		}
	}

	if a.Kind() != b.Kind() {
		return nil, fmt.Errorf("Kind mismatch %v %v", a.Kind(), b.Kind())
	}

	switch a.Kind() {
	case reflect.Slice:
		// TODO: this will work rather poorly if there are missing or extra items.
		res := make([]interface{}, a.Len())
		for i := 0; i < a.Len(); i++ {
			val, err := diff(diffLevel, a.Index(i), b.Index(i), differs)
			if err != nil {
				return nil, err
			}
			res[i] = val
		}
		return res, nil
	case reflect.Map:
		res := make(map[string]interface{})
		for _, key := range a.MapKeys() {
			val, err := diff(diffLevel, a.MapIndex(key), b.MapIndex(key), differs)
			if err != nil {
				return nil, err
			}
			res[fmt.Sprintf("%v", key.Interface())] = val
		}
		return res, nil
	case reflect.Struct:
		res := make(map[string]interface{})
		for i := 0; i < a.NumField(); i++ {
			field := a.Type().Field(i)
			fieldName := field.Name

			if !shouldIncludeInDiff(diffLevel, field) {
				continue
			}

			val, err := diff(diffLevel, a.Field(i), b.Field(i), differs)
			if err != nil {
				return nil, err
			}
			res[fieldName] = val
		}
		return res, nil
	case reflect.Ptr:
		return diff(diffLevel, a.Elem(), b.Elem(), differs)
	default:
		if !a.CanInterface() || !b.CanInterface() {
			return "(cannot access)", nil
		}

		ai := a.Interface()
		bi := b.Interface()
		if reflect.DeepEqual(ai, bi) {
			return ai, nil
		} else {
			return map[string]interface{}{
				"left":  ai,
				"right": bi,
			}, nil
		}
	}
}

func diffLinks(a *Link, b *Link) interface{} {
	if a.Key == b.Key {
		return a
	} else {
		return fmt.Sprintf("'%v' != '%v'", a.Key, b.Key)
	}
}

func diffTimes(a time.Time, b time.Time) interface{} {
	if a == b {
		return a
	} else {
		return fmt.Sprintf("%v . . . %v (%v difference)", a, b, b.Sub(a))
	}
}

func diffAnnotatedSamples(a Sample, b Sample) interface{} {
	return diffDurations(a.Duration(), b.Duration())
}

func diffDurations(a time.Duration, b time.Duration) interface{} {
	if a == b {
		return fmt.Sprintf("%v", a)
	}
	return fmt.Sprintf("%v . . .  %v (%v difference, %.3f%%)", a, b, b-a, 100*(float64(b)-float64(a))/float64(a))
}

func diffCounters(a benchmark.Counters, b benchmark.Counters) interface{} {
	res, err := diff(2, reflect.ValueOf(a.AllCounters()), reflect.ValueOf(b.AllCounters()), []interface{}{})
	if err != nil {
		panic(err)
	}
	return res
}

func diffSamplers(a Multisample, b Multisample) interface{} {
	return map[string]interface{}{
		"average": diffDurations(a.Average(), b.Average()),
		"median":  diffDurations(a.Median(), b.Median()),
		"min":     diffDurations(a.Min(), b.Min()),
		"max":     diffDurations(a.Max(), b.Max()),
	}
}

func summaryVerb(ctx log.Context, flags flag.FlagSet) error {
	if flags.NArg() < 1 {
		app.Usage(ctx, "At least one argument expected.")
		return nil
	}

	perfz1, err := LoadPerfz(ctx, flags.Arg(0), flagVerifyHashes)
	if err != nil {
		return err
	}

	var perfz2 *Perfz

	if flags.NArg() >= 2 {
		perfz2, err = LoadPerfz(ctx, flags.Arg(1), flagVerifyHashes)
		if err != nil {
			return err
		}
	} else {
		perfz2 = perfz1
	}

	diffResult, err := diff(
		flagLevel,
		reflect.ValueOf(perfz1),
		reflect.ValueOf(perfz2),
		[]interface{}{diffLinks, diffTimes, diffSamplers, diffAnnotatedSamples, diffCounters},
	)
	if err != nil {
		return err
	}

	j, err := json.MarshalIndent(diffResult, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(j))

	return nil
}
