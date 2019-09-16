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

package dependencygraph2

import (
	"context"

	"github.com/google/gapid/core/log"
)

type ForwardAccessMode uint

const (
	FORWARD_OPEN ForwardAccessMode = iota + 1
	FORWARD_CLOSE
	FORWARD_DROP
)

type ForwardNodes struct {
	Open  NodeID
	Close NodeID
	Drop  NodeID
}

type ForwardAccess struct {
	Nodes        *ForwardNodes
	DependencyID interface{}
	Mode         ForwardAccessMode
}

type Accesses struct {
	nodeAccesses map[NodeID][]ForwardAccess
	isUnopened   bool
}

type ForwardWatcher interface {
	OpenForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{})
	CloseForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{})
	DropForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{})
	OnBeginCmd(ctx context.Context, cmdCtx CmdContext)
	OnEndCmd(ctx context.Context, cmdCtx CmdContext) Accesses
	OnBeginSubCmd(ctx context.Context, cmdCtx CmdContext, subCmdCtx CmdContext)
	OnEndSubCmd(ctx context.Context, cmdCtx CmdContext)
}

type forwardWatcher struct {
	openForwardDependencies map[interface{}]*ForwardNodes
	forwardAccesses         []ForwardAccess
	nodeAccesses            map[NodeID][]ForwardAccess
	isUnopened              bool
}

func NewForwardWatcher() *forwardWatcher {
	return &forwardWatcher{
		openForwardDependencies: make(map[interface{}]*ForwardNodes),
		nodeAccesses:            make(map[NodeID][]ForwardAccess),
	}
}

func (b *forwardWatcher) OpenForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{}) {
	if len(b.forwardAccesses) == 0 {
		if forwardAccesses, ok := b.nodeAccesses[cmdCtx.nodeID]; ok {
			b.forwardAccesses = forwardAccesses
		}
	}

	nodes := ForwardNodes{
		Open:  cmdCtx.nodeID,
		Close: NodeNoID,
		Drop:  NodeNoID,
	}
	acc := ForwardAccess{
		Nodes:        &nodes,
		DependencyID: dependencyID,
		Mode:         FORWARD_OPEN,
	}

	if _, ok := b.openForwardDependencies[dependencyID]; ok {
		log.D(ctx, "OpenForwardDependency: Forward dependency opened multiple times before being closed. DependencyID: %v, close node: %v", dependencyID, cmdCtx.nodeID)
	} else {
		b.openForwardDependencies[dependencyID] = acc.Nodes
	}
	b.forwardAccesses = append(b.forwardAccesses, acc)
}

func (b *forwardWatcher) CloseForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{}) {
	if len(b.forwardAccesses) == 0 {
		if forwardAccesses, ok := b.nodeAccesses[cmdCtx.nodeID]; ok {
			b.forwardAccesses = forwardAccesses
		}
	}

	if open, ok := b.openForwardDependencies[dependencyID]; ok {
		delete(b.openForwardDependencies, dependencyID)
		open.Close = cmdCtx.nodeID
		b.forwardAccesses = append(b.forwardAccesses, ForwardAccess{
			Nodes:        open,
			DependencyID: dependencyID,
			Mode:         FORWARD_CLOSE,
		})
	} else {
		b.isUnopened = true
		log.D(ctx, "CloseForwardDependency: Forward dependency closed before being opened. DependencyID: %v, close node: %v", dependencyID, cmdCtx.nodeID)
	}
}

func (b *forwardWatcher) DropForwardDependency(ctx context.Context, cmdCtx CmdContext, dependencyID interface{}) {
	if open, ok := b.openForwardDependencies[dependencyID]; ok {
		delete(b.openForwardDependencies, dependencyID)
		open.Drop = cmdCtx.nodeID
		b.forwardAccesses = append(b.forwardAccesses, ForwardAccess{
			Nodes:        open,
			DependencyID: dependencyID,
			Mode:         FORWARD_DROP,
		})
	} else {
		log.D(ctx, "DropForwardDependency: Forward dependency dropped before being opened. DependencyID: %v, close node: %v", dependencyID, cmdCtx.nodeID)
	}
}

func (b *forwardWatcher) Flush(ctx context.Context, cmdCtx CmdContext) {
	b.nodeAccesses[cmdCtx.nodeID] = b.forwardAccesses
	b.forwardAccesses = []ForwardAccess{}
}
func (b *forwardWatcher) OnBeginCmd(ctx context.Context, cmdCtx CmdContext) {}
func (b *forwardWatcher) OnEndCmd(ctx context.Context, cmdCtx CmdContext) Accesses {
	b.Flush(ctx, cmdCtx)
	acc := Accesses{
		nodeAccesses: b.nodeAccesses,
		isUnopened:   b.isUnopened,
	}
	b.nodeAccesses = make(map[NodeID][]ForwardAccess)
	b.forwardAccesses = []ForwardAccess{}
	b.isUnopened = false
	return acc
}
func (b *forwardWatcher) OnBeginSubCmd(ctx context.Context, cmdCtx CmdContext, subCmdCtx CmdContext) {
	b.Flush(ctx, cmdCtx)
}
func (b *forwardWatcher) OnEndSubCmd(ctx context.Context, cmdCtx CmdContext) {
	b.Flush(ctx, cmdCtx)
}
