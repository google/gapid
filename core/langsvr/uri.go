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

package langsvr

import (
	"fmt"
	"net/url"
	"path/filepath"
)

// URItoPath returns the absolute filepath from the given URI.
func URItoPath(uri string) (string, error) {
	if url, _ := url.Parse(uri); url.Scheme == "file" {
		path, err := filepath.Abs(url.Path)
		if err != nil {
			return "", err
		}
		return path, nil
	}
	return "", fmt.Errorf("URI was not a file path")
}

// PathToURI returns the URI for the absolute filepath
func PathToURI(path string) string {
	return "file://" + filepath.ToSlash(path)
}
