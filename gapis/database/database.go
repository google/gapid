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
)

// Database is the interface to a resource store.
type Database interface {
	// Store adds a key-value pair to the database.
	// It is an error if the id is already mapped to an object.
	Store(context.Context, interface{}) (id.ID, error)
	// Resolve attempts to resolve the final value associated with an id.
	// It will traverse all Resolvable objects, blocking until they are ready.
	Resolve(context.Context, id.ID) (interface{}, error)
	// Contains returns true if the database has an entry for the specified id.
	Contains(context.Context, id.ID) bool
}

// Store stores v to the database held by the context.
func Store(ctx context.Context, v interface{}) (id.ID, error) {
	return Get(ctx).Store(ctx, v)
}

// Resolve resolves id with the database held by the context.
func Resolve(ctx context.Context, id id.ID) (interface{}, error) {
	return Get(ctx).Resolve(ctx, id)
}

// Build stores resolvable into d, and then resolves and returns the resolved
// object.
func Build(ctx context.Context, r Resolvable) (interface{}, error) {
	id, err := Store(ctx, r)
	if err != nil {
		return nil, err
	}
	return Get(ctx).Resolve(ctx, id)
}

type databaseKeyTy string

const databaseKey = databaseKeyTy("database")

// Get returns the Database attached to the given context.
func Get(ctx context.Context) Database {
	if val := ctx.Value(databaseKey); val != nil {
		return val.(Database)
	}
	panic("database missing from context")
}

// Put amends a Context by attaching a Database reference to it.
func Put(ctx context.Context, d Database) context.Context {
	if val := ctx.Value(databaseKey); val != nil {
		panic("Context already holds database")
	}
	return keys.WithValue(ctx, databaseKey, d)
}
