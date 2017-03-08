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

package manifest

import (
	"context"
	"encoding/xml"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

const (
	// ActionMain is the action treated as the main entry point, which does not
	// expect to receive data.
	ActionMain = "android.intent.action.MAIN"

	// CategoryInfo provides information about the package it is in; typically
	// used if a package does not contain a CATEGORY_LAUNCHER to provide a
	// front-door to the user without having to be shown in the all apps list.
	CategoryInfo = "android.intent.category.INFO"

	// CategoryLauncher means the action should be displayed in the top-level
	// launcher.
	CategoryLauncher = "android.intent.category.LAUNCHER"

	ErrNoActivityFound = fault.Const("No suitable activity found")
)

// Manifest represents an APK's AndroidManifest.xml file.
type Manifest struct {
	Package     string       `xml:"package,attr"`
	VersionCode int          `xml:"versionCode,attr"`
	VersionName string       `xml:"versionName,attr"`
	Application Application  `xml:"application"`
	Features    []Feature    `xml:"uses-feature"`
	Permissions []Permission `xml:"uses-permission"`
}

// Parse parses the AndroidManifest XML string.
func Parse(ctx context.Context, s string) (Manifest, error) {
	m := Manifest{}
	if err := xml.Unmarshal([]byte(s), &m); err != nil {
		return Manifest{}, log.Err(ctx, err, "Parsing manifest")
	}
	return m, nil
}

// Application represents an application declared in an APK.
type Application struct {
	Activities []Activity `xml:"activity"`
	Debuggable bool       `xml:"debuggable,attr"`
}

// Activity represents an activity declared in an Application.
type Activity struct {
	Name          string         `xml:"name,attr"`
	IntentFilters []IntentFilter `xml:"intent-filter"`
}

// IntentFilter represents an intent filter declared in an Activity.
type IntentFilter struct {
	Action     Action     `xml:"action"`
	Categories []Category `xml:"category"`
}

// Action represents an action of an IntentFilter.
type Action struct {
	Name string `xml:"name,attr"`
}

// Category represents the category of an IntentFilter.
type Category struct {
	Name string `xml:"name,attr"`
}

// Feature represents a feature used by an APK.
type Feature struct {
	Name        string `xml:"name,attr"`
	Required    bool   `xml:"required,attr"`
	GlEsVersion string `xml:"glEsVersion,attr"`
}

// Permission represents a permission used by an APK.
type Permission struct {
	Name string `xml:"name,attr"`
}

func (m Manifest) MainActivity(ctx context.Context) (activity, action string, err error) {
	search := func(category string) (activity, action string, ok bool) {
		for _, a := range m.Application.Activities {
			for _, i := range a.IntentFilters {
				if i.Action.Name == ActionMain {
					for _, c := range i.Categories {
						if c.Name == category {
							return a.Name, i.Action.Name, true
						}
					}
				}
			}
		}
		return "", "", false
	}

	var ok bool
	// Try searching for a launcher category intent
	if activity, action, ok = search(CategoryInfo); ok {
		return
	}
	// Try searching for a launcher category intent
	if activity, action, ok = search(CategoryLauncher); ok {
		return
	}
	// Not found.
	return "", "", log.Err(ctx, ErrNoActivityFound, "")
}
