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

package android

import (
	"fmt"
	"strings"
)

// ActivityAction represents an Android action that can be sent as an intent to
// an activity.
type ActivityAction struct {
	// The action name.
	// Example: android.intent.action.MAIN
	Name string

	// The package that owns this activity action.
	Package *InstalledPackage

	// The activity that performs the action.
	// Example:  FooBarAction
	Activity string
}

// ActivityActions is a list of activity actions.
type ActivityActions []*ActivityAction

// FindByName returns the service action with the specified names, or nil if no
// action with the matching names is found.
func (l ActivityActions) FindByName(action, activity string) *ActivityAction {
	for _, a := range l {
		if a.Name == action && a.Activity == activity {
			return a
		}
	}
	return nil
}

// ServiceAction represents an Android action that can be sent as an intent to
// a service.
type ServiceAction struct {
	// The action name.
	// Example: android.intent.action.MAIN
	Name string

	// The package that owns this service action.
	Package *InstalledPackage

	// The service that performs the action.
	// Example:  FooBarService
	Service string
}

// ServiceActions is a list of service actions.
type ServiceActions []*ServiceAction

// FindByName returns the service action with the specified names, or nil if no
// action with the matching names is found.
func (l ServiceActions) FindByName(action, service string) *ServiceAction {
	for _, a := range l {
		if a.Name == action && a.Service == service {
			return a
		}
	}
	return nil
}

// Component returns the component name with package name prefix. For example:
// "com.example.app/.ExampleActivity" or "com.example.app/com.foo.ExampleActivity"
func (a *ActivityAction) Component() string {
	return actionComponent(a.Package, a.Activity)
}

func (a *ActivityAction) String() string {
	return a.Name + ":" + a.Component()
}

// Component returns the component name with package name prefix. For example:
// "com.example.app/.ExampleService" or "com.example.app/com.foo.ExampleService"
func (a *ServiceAction) Component() string {
	return actionComponent(a.Package, a.Service)
}

func (a *ServiceAction) String() string {
	return a.Name + ":" + a.Component()
}

func actionComponent(pkg *InstalledPackage, owner string) string {
	if strings.ContainsRune(owner, '.') {
		return fmt.Sprintf("%s/%s", pkg.Name, owner)
	}
	return fmt.Sprintf("%s/.%s", pkg.Name, owner)
}

// ActionExtra is the interface implemented by intent extras.
type ActionExtra interface {
	// Flags returns the formatted flags to pass to the Android am command.
	Flags() []string
}

// StringExtra represents an extra with a string value.
type StringExtra struct {
	Key   string
	Value string
}

// BoolExtra represents an extra with a bool value.
type BoolExtra struct {
	Key   string
	Value bool
}

// IntExtra represents an extra with an int value.
type IntExtra struct {
	Key   string
	Value int
}

// LongExtra represents an extra with a long value.
type LongExtra struct {
	Key   string
	Value int
}

// FloatExtra represents an extra with a float value.
type FloatExtra struct {
	Key   string
	Value float32
}

// URIExtra represents an extra with a URI value.
type URIExtra struct {
	Key   string
	Value string
}

// CustomExtras is a list of custom intent extras
type CustomExtras []string

// Flags returns the formatted flags to pass to the Android am command.
func (e StringExtra) Flags() []string { return []string{"--es", e.Key, fmt.Sprintf(`"%v"`, e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e BoolExtra) Flags() []string { return []string{"--ez", e.Key, fmt.Sprintf("%v", e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e IntExtra) Flags() []string { return []string{"--ei", e.Key, fmt.Sprintf("%v", e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e LongExtra) Flags() []string { return []string{"--el", e.Key, fmt.Sprintf("%v", e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e FloatExtra) Flags() []string { return []string{"--ef", e.Key, fmt.Sprintf("%v", e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e URIExtra) Flags() []string { return []string{"--eu", e.Key, fmt.Sprintf("%v", e.Value)} }

// Flags returns the formatted flags to pass to the Android am command.
func (e CustomExtras) Flags() []string { return []string(e) }
