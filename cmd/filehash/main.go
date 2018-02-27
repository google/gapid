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
	"context"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/gapid/core/app"
)

var (
	hashlen = flag.Int("hashlen", 6, "length in characters of the hash output")
	in      = flag.String("in", "", "An input template file to replace the hash in")
	replace = flag.String("replace", "", "The token to replace with the hash")
	out     = flag.String("out", "", "An output file to write to")
)

func main() {
	app.ShortHelp = "filehash produces a SHA1 hash from the list of files"
	app.Name = "filehash"
	app.ShortUsage = "<files>"
	app.Run(run)
}

func run(ctx context.Context) error {
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
	}
	hasher := sha1.New()
	for _, path := range args {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}
	}
	hash := fmt.Sprintf("%x", hasher.Sum(nil))
	if l := *hashlen; len(hash) > l {
		hash = hash[:l]
	}
	if *in == "" {
		fmt.Print(hash)
		return nil
	}
	data, err := ioutil.ReadFile(*in)
	if err != nil {
		return err
	}
	result := strings.Replace(string(data), *replace, hash, -1)
	if *out == "" {
		fmt.Print(result)
	}
	if result == string(data) {
		return fmt.Errorf("'%v' was not found in file %v", *replace, *in)
	}
	return ioutil.WriteFile(*out, []byte(result), 0777)
}
