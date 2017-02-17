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

// Package database implements the persistence layer for the gpu debugger tools.
package database

import (
	"context"

	"github.com/google/gapid/core/context/keys"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/note"
)

// Database is the interface to a resource store.
type Database interface {
	// Store adds a key-value pair to the database.
	// It is an error if the id is already mapped to an object.
	Store(log.Context, id.ID, interface{}) error
	// Resolve attempts to resolve the final value associated with an id.
	// It will traverse all Resolvable objects, blocking until they are ready.
	Resolve(log.Context, id.ID) (interface{}, error)
	// Containts returns true if the database has an entry for the specified id.
	Contains(log.Context, id.ID) bool
}

// Store is a helper that stores v to the database with the id calculated by
// the Hash function.
func Store(ctx log.Context, v interface{}) (id.ID, error) {
	id, err := Hash(v)
	if err != nil {
		return id, err
	}
	return id, Get(ctx).Store(ctx, id, v)
}

// Resolve is a helper that resolves id with the database held by the context.
func Resolve(ctx log.Context, id id.ID) (interface{}, error) {
	return Get(ctx).Resolve(ctx, id)
}

// Build stores resolvable into d, and then resolves and returns the resolved
// object.
func Build(ctx log.Context, r Resolvable) (interface{}, error) {
	id, err := Store(ctx, r)
	if err != nil {
		return nil, err
	}
	return Get(ctx).Resolve(ctx, id)
}

type databaseKeyTy string

const databaseKey = databaseKeyTy("database")

func (databaseKeyTy) Transcribe(context.Context, *note.Page, interface{}) {}

// Get returns the Database attached to the given context.
func Get(ctx log.Context) Database {
	if val := ctx.Value(databaseKey); val != nil {
		return val.(Database)
	}
	panic("database missing from context")
}

// Put amends a Context by attaching a Database reference to it.
func Put(ctx log.Context, d Database) log.Context {
	if val := ctx.Value(databaseKey); val != nil {
		panic("Context already holds database")
	}
	return log.Wrap(keys.WithValue(ctx.Unwrap(), databaseKey, d))
}
