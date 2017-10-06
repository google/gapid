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

package dom

import (
	"path/filepath"
)

type Source struct {
	*Element
	Src string `js:"src"`
	Typ string `js:"type"`
}

// Video represents an HTML <video> element
type Video struct {
	*Element

	Autoplay bool `js:"autoplay"`
	Controls bool `js:"controls"`
}

func isSupportedSourceType(typ string) bool {
	return map[string]bool{"mp4": true, "webm": true, "ogg": true}[typ]
}

func (v *Video) Resize(width, height int) {
	drp := Win.DevicePixelRatio
	style := v.Style
	v.Width, v.Height = int(float64(width)*drp), int(float64(height)*drp)
	style.Width, style.Height = width, height
}

func (s *Source) SetSource(source, typ string) {
	s.Src = source
	if isSupportedSourceType(typ) {
		s.Typ = "video/" + typ
	} else if isSupportedSourceType(filepath.Ext(source)) {
		s.Typ = "video/" + filepath.Ext(source)
	}
}

func NewVideo(width, height int, source, typ string) *Video {
	v := &Video{Element: newEl("video")}
	v.Autoplay = true
	v.Controls = true
	s := &Source{Element: newEl("source")}
	s.SetSource(source, typ)
	v.Append(s)
	v.Resize(width, height)
	return v
}
