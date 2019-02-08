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

package file

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

// ZIP zips the given input file/directory to the given writer.
func ZIP(out io.Writer, in Path) error {
	w := zip.NewWriter(out)

	err := filepath.Walk(in.System(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		r, err := filepath.Rel(in.System(), path)
		if err != nil {
			return err
		}

		// We are zipping a single file
		if r == "." {
			r = in.Basename()
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = r
		fw, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		fr, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fr.Close()

		_, err = io.Copy(fw, fr)
		return err
	})
	if err != nil {
		return err
	}

	return w.Close()
}
