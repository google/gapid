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
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/stream/fmts"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/client"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
	"github.com/google/gapid/gapis/vertex"
)

var (
	actionKey               = struct{}{}
	thumbSize        uint32 = 192
	fbRenderSettings        = &path.RenderSettings{
		MaxWidth:  0xffff,
		MaxHeight: 0xffff,
		DrawMode:  path.DrawMode_NORMAL,
	}
	fbHints    = &path.UsageHints{Primary: true}
	meshFormat = &vertex.BufferFormat{
		Streams: []*vertex.StreamFormat{
			&vertex.StreamFormat{
				Semantic: &vertex.Semantic{
					Type:  vertex.Semantic_Position,
					Index: 0,
				},
				Format: fmts.XYZ_F32,
			},
			&vertex.StreamFormat{
				Semantic: &vertex.Semantic{
					Type:  vertex.Semantic_Normal,
					Index: 0,
				},
				Format: fmts.XYZ_F32,
			},
		},
	}
	defaultNumDraws = 2
)

type benchmarkVerb struct{ BenchmarkFlags }

func init() {
	verb := &benchmarkVerb{}

	app.AddVerb(&app.Verb{
		Name:      "benchmark",
		ShortHelp: "Runs a set of benchmarking tests on a trace",
		Action:    verb,
	})
}

func printIndices(index []uint64) string {
	parts := make([]string, len(index))
	for i, v := range index {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, ".")
}

func ignoreDataUnavailable(val interface{}, err error) (interface{}, error) {
	if _, ok := err.(*service.ErrDataUnavailable); ok {
		return val, nil
	}
	return val, err
}

func newRandom(seed string, offset int64) *rand.Rand {
	h := fnv.New64()
	h.Write([]byte(seed))
	return rand.New(rand.NewSource(int64(h.Sum64()) + offset))
}

type cmdTree struct {
	path     *path.CommandTreeNode
	node     *service.CommandTreeNode
	cmd      *api.Command
	children []cmdTree
}

type chooser func(n int) (int, error)

func (t *cmdTree) choose(choose chooser, pred func(t *cmdTree) bool) (*cmdTree, error) {
	var candidates []*cmdTree
	for i := range t.children {
		if pred(&t.children[i]) {
			candidates = append(candidates, &t.children[i])
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	idx, err := choose(len(candidates))
	if err != nil {
		return nil, err
	}

	return candidates[idx], nil
}

func (t *cmdTree) chooseQueueSubmit(choose chooser) (*cmdTree, error) {
	submit, err := t.choose(choose, func(t *cmdTree) bool {
		return t.cmd.GetName() == "vkQueueSubmit"
	})

	if err != nil {
		return nil, err
	} else if submit == nil {
		return nil, fmt.Errorf("No submits found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return submit, nil
}

func (t *cmdTree) chooseSubmitInfo(choose chooser) (*cmdTree, error) {
	info, err := t.choose(choose, func(t *cmdTree) bool {
		return strings.HasPrefix(t.node.GetGroup(), "pSubmits[")
	})

	if err != nil {
		return nil, err
	} else if info == nil {
		return nil, fmt.Errorf("No submit infos found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return info, nil
}

func (t *cmdTree) chooseCommandBuffer(choose chooser) (*cmdTree, error) {
	cb, err := t.choose(choose, func(t *cmdTree) bool {
		return strings.HasPrefix(t.node.GetGroup(), "Command Buffer: ")
	})

	if err != nil {
		return nil, err
	} else if cb == nil {
		return nil, fmt.Errorf("No command buffers found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return cb, nil
}

func (t *cmdTree) chooseRenderPass(choose chooser) (*cmdTree, error) {
	rp, err := t.choose(choose, func(t *cmdTree) bool {
		return strings.HasPrefix(t.node.GetGroup(), "RenderPass: ")
	})

	if err != nil {
		return nil, err
	} else if rp == nil {
		return nil, fmt.Errorf("No renderpass found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return rp, nil
}

func (t *cmdTree) chooseExecute(choose chooser) (*cmdTree, error) {
	e, err := t.choose(choose, func(t *cmdTree) bool {
		return t.cmd.GetName() == "vkCmdExecuteCommands"
	})

	if err != nil {
		return nil, err
	} else if e == nil {
		return nil, fmt.Errorf("No vkCmdExecuteCommands found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return e, nil
}

func (t *cmdTree) chooseDrawCall(choose chooser) (*cmdTree, error) {
	dc, err := t.choose(choose, func(t *cmdTree) bool {
		return strings.HasPrefix(t.node.GetGroup(), "Draw")
	})

	if err != nil {
		return nil, err
	} else if dc == nil {
		return nil, fmt.Errorf("No draw calls found at node<%s>", printIndices(t.path.GetIndices()))
	}

	return dc, nil
}

type stateTree struct {
	path     *path.StateTreeNode
	node     *service.StateTreeNode
	children []stateTree
}

type measurement struct {
	name     string
	duration time.Duration
	children []*measurement
}

type drawSummary struct {
	find        time.Duration
	selection   time.Duration
	framebuffer time.Duration
}

type summary struct {
	init            time.Duration
	initStart       time.Duration
	initLoad        time.Duration
	initLoadProfile time.Duration
	draws           []drawSummary
}

func (s *summary) print(w io.Writer) error {
	if _, err := fmt.Fprint(w, "Init,Init.Start,Init.Load,Init.Load.Profile"); err != nil {
		return err
	}
	for i := range s.draws {
		if _, err := fmt.Fprintf(w, ",Draw%d.Find,Draw%d.Select,Draw%d.Select.Framebuffer", i, i, i); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "%0.3f,%0.3f,%0.3f,%0.3f", s.init.Seconds(), s.initStart.Seconds(), s.initLoad.Seconds(), s.initLoadProfile.Seconds()); err != nil {
		return err
	}
	for _, draw := range s.draws {
		if _, err := fmt.Fprintf(w, ",%0.3f,%0.3f,%0.3f", draw.find.Seconds(), draw.selection.Seconds(), draw.framebuffer.Seconds()); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return nil
}

func (m *measurement) getDuration() time.Duration {
	if m == nil {
		return time.Duration(0)
	}
	return m.duration
}

func (m *measurement) computeSummary() summary {
	r := summary{
		init:            m.find("init").getDuration(),
		initStart:       m.find("init", "start_gapis").getDuration(),
		initLoad:        m.find("init", "load").getDuration(),
		initLoadProfile: m.find("init", "load", "profile").getDuration(),
	}

	for _, child := range m.children {
		if child.name == "find_and_select_command" {
			r.draws = append(r.draws, drawSummary{
				find:        child.find("find_draw").getDuration(),
				selection:   child.find("select_command").getDuration(),
				framebuffer: child.find("select_command", "framebuffer").getDuration(),
			})
		}
	}

	return r
}

func (m *measurement) find(path ...string) *measurement {
	if len(path) == 0 {
		return m
	}

	for _, child := range m.children {
		if child.name == path[0] || strings.HasPrefix(child.name, path[0]+"<") {
			if r := child.find(path[1:]...); r != nil {
				return r
			}
		}
	}
	return nil
}

func (m *measurement) writeCsv(w io.Writer) error {
	for _, child := range m.children {
		if err := child.writeCsvNode(w, ""); err != nil {
			return err
		}
	}
	return nil
}

func (m *measurement) writeCsvNode(w io.Writer, parent string) error {
	name := parent + m.name
	if _, err := fmt.Fprintf(w, "\"%s\",%v\n", name, m.duration); err != nil {
		return err
	}
	for _, child := range m.children {
		if err := child.writeCsvNode(w, name+"."); err != nil {
			return err
		}
	}
	return nil
}

func (m *measurement) writeGraph(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "digraph Profile {"); err != nil {
		return err
	}

	id := 1
	var err error
	for _, child := range m.children {
		id, err = child.writeGraphNode(w, id)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w, "}")
	return err
}

func (m *measurement) writeGraphNode(w io.Writer, id int) (int, error) {
	if _, err := fmt.Fprintf(w, "%d [label=\"%s|%v\"];\n", id, m.name, m.duration); err != nil {
		return 0, err
	}

	cur := id + 1
	for _, child := range m.children {
		next, err := child.writeGraphNode(w, cur)
		if err != nil {
			return 0, err
		}
		if _, err := fmt.Fprintf(w, "%d -> %d;\n", id, cur); err != nil {
			return 0, err
		}
		cur = next
	}
	return cur, nil
}

type benchmark struct {
	rnd *rand.Rand

	client    client.Client
	capture   *path.Capture
	device    *path.Device
	cmdTree   *cmdTree
	resources *service.Resources

	measurement measurement
	mutex       sync.Mutex

	gapitTrace     bytes.Buffer
	gapisTrace     bytes.Buffer
	stopGapitTrace status.Unregister
	stopGapisTrace func() error
}

type action interface {
	name() string
	exec(ctx context.Context, b *benchmark) (interface{}, error)
	cleanup(ctx context.Context, b *benchmark, err error)
}

func (b *benchmark) measure(ctx context.Context, a action) (interface{}, error) {
	return b.measureFun(ctx, a.name(), func(ctx context.Context) (interface{}, error) {
		return a.exec(ctx, b)
	})
}

func (b *benchmark) measureFun(ctx context.Context, name string, f func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	me := &measurement{name: name}

	var parent *measurement
	parentValue := ctx.Value(actionKey)
	if parentValue == nil {
		parent = &b.measurement
	} else {
		parent = parentValue.(*measurement)
	}

	b.mutex.Lock()
	parent.children = append(parent.children, me)
	b.mutex.Unlock()

	ctx = context.WithValue(ctx, actionKey, me)

	ctx = status.Start(ctx, name)
	defer status.Finish(ctx)

	start := time.Now()
	defer func() {
		me.duration = time.Since(start)
	}()

	return f(ctx)
}

func (b *benchmark) resolveConfig() *path.ResolveConfig {
	return &path.ResolveConfig{
		ReplayDevice: b.device,
	}
}

func (b *benchmark) resourcesByType(after *path.Command, ty path.ResourceType) []*service.Resource {
	var res []*service.Resource
	for _, byType := range b.resources.GetTypes() {
		if byType.GetType() == ty {
			for _, resource := range byType.GetResources() {
				count := len(resource.Accesses)
				if count == 0 || !resource.Accesses[0].IsAfter(after) {
					res = append(res, resource)
				}
			}
		}
	}
	return res
}

func (b *benchmark) textures(after *path.Command) []*service.Resource {
	return b.resourcesByType(after, path.ResourceType_TextureResource)
}

func (b *benchmark) shaders(after *path.Command) []*service.Resource {
	return b.resourcesByType(after, path.ResourceType_ShaderResource)
}

type parallel struct {
	label   string
	actions []action
}

func (p *parallel) name() string {
	return p.label
}

func (p *parallel) exec(ctx context.Context, b *benchmark) (interface{}, error) {
	var wg sync.WaitGroup
	errors := make([]error, len(p.actions))

	for i := range p.actions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := status.PutTask(ctx, nil) // Clear the parent, since we're parallel
			_, err := b.measure(ctx, p.actions[i])
			errors[i] = err
		}(i)
	}

	wg.Wait()

	// TODO: combine errors, rather than returning the first non-nil error.
	for _, err := range errors {
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (p *parallel) cleanup(ctx context.Context, b *benchmark, err error) {
	for _, a := range p.actions {
		a.cleanup(ctx, b, err)
	}
}

type sequential struct {
	label   string
	actions []action
}

func (s *sequential) name() string {
	return s.label
}

func (s *sequential) exec(ctx context.Context, b *benchmark) (interface{}, error) {
	for _, a := range s.actions {
		if _, err := b.measure(ctx, a); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (s *sequential) cleanup(ctx context.Context, b *benchmark, err error) {
	for i := len(s.actions) - 1; i >= 0; i-- {
		s.actions[i].cleanup(ctx, b, err)
	}
}

type actionFun func(ctx context.Context, b *benchmark) (interface{}, error)

type simple struct {
	label string
	fun   actionFun
}

func (s *simple) name() string {
	return s.label
}

func (s *simple) exec(ctx context.Context, b *benchmark) (interface{}, error) {
	return s.fun(ctx, b)
}

func (s *simple) cleanup(ctx context.Context, b *benchmark, err error) {
}

type startGapis struct {
	gapisFlags *GapisFlags
	gapirFlags *GapirFlags
	dumpTrace  bool
}

func (a *startGapis) name() string {
	return "start_gapis"
}

func (a *startGapis) exec(ctx context.Context, b *benchmark) (interface{}, error) {
	var err error
	if b.client, err = getGapis(ctx, *a.gapisFlags, *a.gapirFlags); err != nil {
		return nil, err
	}

	if a.dumpTrace {
		b.stopGapisTrace, err = b.client.Profile(ctx, nil, &b.gapisTrace, 1)
		if err != nil {
			return nil, err
		}
	}

	_, err = b.measureFun(ctx, "server_info", func(ctx context.Context) (interface{}, error) {
		return b.client.GetServerInfo(ctx)
	})
	if err != nil {
		return nil, err
	}

	_, err = b.measureFun(ctx, "string_table", func(ctx context.Context) (interface{}, error) {
		tables, err := b.client.GetAvailableStringTables(ctx)
		if len(tables) > 0 {
			_, err = b.client.GetStringTable(ctx, tables[0])
		}
		return nil, err
	})

	return nil, err
}

func (a *startGapis) cleanup(ctx context.Context, b *benchmark, err error) {
	if b.stopGapisTrace != nil {
		b.stopGapisTrace()
	}
	if b.client != nil {
		b.client.Close()
	}
}

func loadCapture(file string) action {
	return &simple{
		"load_capture",
		func(ctx context.Context, b *benchmark) (ignored interface{}, err error) {
			b.capture, err = b.client.LoadCapture(ctx, file)
			if err == nil {
				var c interface{}
				c, err = b.client.Get(ctx, b.capture.Path(), &path.ResolveConfig{})
				if err == nil && c.(*service.Capture).GetType() != service.TraceType_Graphics {
					err = errors.New("Not a graphics capture")
				}
			}
			return
		},
	}
}

func loadReplaydevice() action {
	return &simple{
		"load_replay_device",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			devices, compat, _, err := b.client.GetDevicesForReplay(ctx, b.capture)
			if err != nil {
				return nil, err
			}

			if len(devices) == 0 || !compat[0] {
				return nil, errors.New("No compatible replay device attached")
			}
			b.device = devices[0]
			return devices[0], nil
		},
	}
}

func loadResources() action {
	return &simple{
		"resources",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			res, err := b.client.Get(ctx, b.capture.Resources().Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}
			b.resources = res.(*service.Resources)
			return res, err
		},
	}
}

func loadCommandTree() action {
	return &simple{
		"cmd_tree",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			tree, err := b.client.Get(ctx, (&path.CommandTree{
				Capture:                  b.capture,
				GroupByFrame:             true,
				GroupByDrawCall:          true,
				GroupByTransformFeedback: true,
				GroupByUserMarkers:       true,
				GroupBySubmission:        true,
				AllowIncompleteFrame:     true,
				MaxChildren:              2000,
				MaxNeighbours:            20,
			}).Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}
			b.cmdTree = &cmdTree{
				path: tree.(*service.CommandTree).GetRoot(),
			}

			if _, err := b.measure(ctx, loadCommandTreeNode(b.cmdTree)); err != nil {
				return nil, err
			}

			return b.measure(ctx, expandCommandTreeNode(b.cmdTree))
		},
	}
}

func loadCommandTreeThumbnail(node *path.CommandTreeNode) action {
	return &simple{
		"cmd_tree_thumbnail",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			info, err := ignoreDataUnavailable(b.client.Get(ctx, (&path.Thumbnail{
				Object:           &path.Thumbnail_CommandTreeNode{CommandTreeNode: node},
				DesiredFormat:    image.RGBA_U8_NORM,
				DesiredMaxWidth:  thumbSize,
				DesiredMaxHeight: thumbSize,
			}).Path(), b.resolveConfig()))
			if err != nil || info == nil {
				return nil, err
			}

			return ignoreDataUnavailable(
				b.client.Get(ctx, path.NewBlob(info.(*image.Info).GetBytes().ID()).Path(), b.resolveConfig()))
		},
	}
}

func loadCommand(cmd *path.Command) action {
	return &simple{
		"command<" + printIndices(cmd.GetIndices()) + ">",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, cmd.Path(), b.resolveConfig())
		},
	}
}

func loadCommandTreeNode(tree *cmdTree) action {
	return &simple{
		"cmd_tree_node<" + printIndices(tree.path.GetIndices()) + ">",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			node, err := b.client.Get(ctx, tree.path.Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}
			tree.node = node.(*service.CommandTreeNode)

			if tree.node.GetGroup() != "" {
				_, err = b.measure(ctx, loadCommandTreeThumbnail(tree.path))
			} else if tree.node.GetCommands() != nil { // group is empty
				cmd, err := b.measure(ctx, loadCommand(tree.node.GetCommands().Last()))
				if err != nil {
					return nil, err
				}
				tree.cmd = cmd.(*api.Command)
			}

			return node, err
		},
	}
}

func expandCommandTreeNode(tree *cmdTree) action {
	actions := make([]action, tree.node.GetNumChildren())
	tree.children = make([]cmdTree, tree.node.GetNumChildren())
	for i := range actions {
		tree.children[i].path = tree.path.Child(uint64(i))
		actions[i] = loadCommandTreeNode(&tree.children[i])
	}
	return &parallel{
		"cmd_tree_expand<" + printIndices(tree.path.GetIndices()) + ">",
		actions,
	}
}

func loadProfilingData() action {
	return &simple{
		"profile",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.GpuProfile(ctx, &service.GpuProfileRequest{
				Capture: b.capture,
				Device:  b.device,
			})
		},
	}
}

func findDrawCall(path []uint64, secondary bool) action {
	return &simple{
		"find_draw",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			choose := func(name string, idx int) chooser {
				if idx < len(path) {
					return func(count int) (int, error) {
						v := int(path[idx])
						if v >= count {
							return 0, fmt.Errorf("Invalid %s index provided in path %v: %d >= %d", name, path, v, count)
						}
						return v, nil
					}
				} else {
					return func(count int) (int, error) {
						return b.rnd.Intn(count), nil
					}
				}
			}

			submit, err := b.cmdTree.chooseQueueSubmit(choose("queue submit", 0))
			if err != nil {
				return nil, err
			}
			if submit.children == nil {
				if _, err := b.measure(ctx, expandCommandTreeNode(submit)); err != nil {
					return nil, err
				}
			}

			info, err := submit.chooseSubmitInfo(choose("submit info", 1))
			if err != nil {
				return nil, err
			}
			if info.children == nil {
				if _, err := b.measure(ctx, expandCommandTreeNode(info)); err != nil {
					return nil, err
				}
			}

			cb, err := info.chooseCommandBuffer(choose("command buffer", 2))
			if err != nil {
				return nil, err
			}
			if cb.children == nil {
				if _, err := b.measure(ctx, expandCommandTreeNode(cb)); err != nil {
					return nil, err
				}
			}

			rp, err := cb.chooseRenderPass(choose("renderpass", 3))
			if err != nil {
				return nil, err
			}
			if rp.children == nil {
				if _, err := b.measure(ctx, expandCommandTreeNode(rp)); err != nil {
					return nil, err
				}
			}

			level := 4
			parent := rp
			if secondary {
				level = 6
				exec, err := rp.chooseExecute(choose("execute", 4))
				if err != nil {
					return nil, err
				}
				if exec.children == nil {
					if _, err := b.measure(ctx, expandCommandTreeNode(exec)); err != nil {
						return nil, err
					}
				}

				parent, err = exec.chooseCommandBuffer(choose("secondary command buffer", 5))
				if err != nil {
					return nil, err
				}
				if parent.children == nil {
					if _, err := b.measure(ctx, expandCommandTreeNode(parent)); err != nil {
						return nil, err
					}
				}
			}

			return parent.chooseDrawCall(choose("draw call", level))
		},
	}
}

func loadAttachments(after *path.Command) action {
	return &simple{
		"attachments",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, (&path.FramebufferAttachments{
				After: after,
			}).Path(), b.resolveConfig())
		},
	}
}

func loadAttachment(after *path.Command, index uint32) action {
	return &simple{
		fmt.Sprintf("attachment<%d>", index),
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, (&path.FramebufferAttachment{
				After:          after,
				Index:          index,
				RenderSettings: fbRenderSettings,
				Hints:          fbHints,
			}).Path(), b.resolveConfig())
		},
	}
}

func loadFrameBuffer(after *path.Command) action {
	return &simple{
		"framebuffer",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			attsBoxed, err := b.measure(ctx, loadAttachments(after))
			if err != nil {
				return nil, err
			}

			atts := attsBoxed.(*service.FramebufferAttachments).GetAttachments()
			if len(atts) == 0 {
				return nil, nil
			}

			attBoxed, err := b.measure(ctx, loadAttachment(after, atts[0].GetIndex()))
			if err != nil {
				return nil, err
			}
			infoPath := attBoxed.(*service.FramebufferAttachment).GetImageInfo()

			infoBoxed, err := b.client.Get(ctx, infoPath.Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}

			return b.client.Get(ctx, path.NewBlob(infoBoxed.(*image.Info).GetBytes().ID()).Path(), b.resolveConfig())
		},
	}
}

func loadTextureThumbnail(data *path.ResourceData) action {
	return &simple{
		"texture_thumbnail",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			info, err := ignoreDataUnavailable(b.client.Get(ctx, (&path.Thumbnail{
				Object:           &path.Thumbnail_Resource{Resource: data},
				DesiredFormat:    image.RGBA_U8_NORM,
				DesiredMaxWidth:  thumbSize,
				DesiredMaxHeight: thumbSize,
			}).Path(), b.resolveConfig()))
			if err != nil || info == nil {
				return nil, err
			}

			return ignoreDataUnavailable(
				b.client.Get(ctx, path.NewBlob(info.(*image.Info).GetBytes().ID()).Path(), b.resolveConfig()))
		},
	}
}

func loadTexture(after *path.Command, texture *service.Resource) action {
	data := after.ResourceAfter(texture.ID)
	return &simple{
		texture.Handle,
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			text, err := b.client.Get(ctx, data.Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}

			_, err = b.measure(ctx, loadTextureThumbnail(data))
			return text, err
		},
	}
}

func loadTextures(after *path.Command) action {
	return &simple{
		"textures",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			textures := b.textures(after)
			actions := make([]action, len(textures))

			for i := range textures {
				actions[i] = loadTexture(after, textures[i])
			}

			return b.measure(ctx, &parallel{
				label:   "load",
				actions: actions,
			})
		},
	}
}

func loadShader(after *path.Command, shader *service.Resource) action {
	return &simple{
		shader.Handle,
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, after.ResourceAfter(shader.ID).Path(), b.resolveConfig())
		},
	}
}

func loadShaders(after *path.Command) action {
	return &simple{
		"shaders",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			shaders := b.shaders(after)
			actions := make([]action, len(shaders))

			for i := range shaders {
				actions[i] = loadShader(after, shaders[i])
			}

			return b.measure(ctx, &parallel{
				label:   "load",
				actions: actions,
			})
		},
	}
}

func loadStateTree(after *path.Command) action {
	return &simple{
		"state_tree",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			treeBoxed, err := b.client.Get(ctx, after.StateTreeAfter(2000).Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}
			tree := &stateTree{
				path: treeBoxed.(*service.StateTree).GetRoot(),
			}
			if _, err := b.measure(ctx, loadStateTreeNode(tree)); err != nil {
				return nil, err
			}

			if _, err := b.measure(ctx, expandStateTreeNode(tree)); err != nil {
				return nil, err
			}

			return tree, nil
		},
	}
}

func loadStateTreeNode(tree *stateTree) action {
	return &simple{
		"state_tree_node<" + printIndices(tree.path.GetIndices()) + ">",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			node, err := b.client.Get(ctx, tree.path.Path(), b.resolveConfig())
			if err != nil {
				return nil, err
			}
			tree.node = node.(*service.StateTreeNode)
			return node, nil
		},
	}
}

func expandStateTreeNode(tree *stateTree) action {
	actions := make([]action, tree.node.GetNumChildren())
	tree.children = make([]stateTree, tree.node.GetNumChildren())
	for i := range actions {
		tree.children[i].path = tree.path.Index(uint64(i))
		actions[i] = loadStateTreeNode(&tree.children[i])
	}
	return &parallel{
		"state_tree_expand<" + printIndices(tree.path.GetIndices()) + ">",
		actions,
	}
}

func loadGeometryMetadata(tree *cmdTree) action {
	return &simple{
		"mesh_metadata",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, tree.path.Mesh(&path.MeshOptions{ExcludeData: true}).Path(), b.resolveConfig())
		},
	}
}

func loadGeometryMesh(tree *cmdTree, faceted bool) action {
	name := "mesh"
	if faceted {
		name = "mesh_faceted"
	}

	return &simple{
		name,
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return ignoreDataUnavailable(
				b.client.Get(ctx, tree.path.Mesh(path.NewMeshOptions(faceted)).As(meshFormat).Path(), b.resolveConfig()))
		},
	}
}

func loadGeometry(tree *cmdTree) action {
	return &sequential{
		label: "geometry",
		actions: []action{
			loadGeometryMetadata(tree),
			&parallel{
				label: "meshes",
				actions: []action{
					loadGeometryMesh(tree, false),
					loadGeometryMesh(tree, true),
				},
			},
		},
	}
}

func loadPipeline(tree *cmdTree) action {
	return &simple{
		"pipeline",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			return b.client.Get(ctx, tree.path.Pipelines().Path(), b.resolveConfig())
		},
	}
}

func selectCommand(tree *cmdTree, isDraw bool) action {
	after := tree.node.GetCommands().Last()
	actions := []action{
		loadFrameBuffer(after),
		loadTextures(after),
		loadStateTree(after),
	}
	if isDraw {
		actions = append(actions,
			loadGeometry(tree),
			loadPipeline(tree),
		)
	}
	return &parallel{
		label:   "select_command<" + printIndices(after.GetIndices()) + ">",
		actions: actions,
	}
}

func selectCommandByAction(selCommand action, isDraw bool) action {
	return &simple{
		"find_and_select_command",
		func(ctx context.Context, b *benchmark) (interface{}, error) {
			node, err := b.measure(ctx, selCommand)
			if err != nil {
				return nil, err
			}

			return b.measure(ctx, selectCommand(node.(*cmdTree), isDraw))
		},
	}
}

func writeOutput(ctx context.Context, path string, f func(w io.Writer) error) error {
	out, err := os.Create(path)
	if err != nil {
		return log.Errf(ctx, err, "Failed to create output file: %s", path)
	}
	defer out.Close()

	if err := f(out); err != nil {
		return log.Errf(ctx, err, "Failed to write output file: %s", path)
	}
	return nil
}

func (verb *benchmarkVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	b := &benchmark{
		rnd: newRandom(flags.Arg(0), verb.Seed),
	}

	if verb.DumpTrace != "" {
		b.stopGapitTrace = status.RegisterTracer(&b.gapitTrace)

		defer func() {
			b.stopGapitTrace()

			f, err := os.Create(verb.DumpTrace)
			if err != nil {
				log.E(ctx, "Failed to create trace file: %v", err)
				return
			}
			defer f.Close()

			_, err = f.Write(b.gapitTrace.Bytes())
			if err != nil {
				log.E(ctx, "Failed to write gapit trace data: %v", err)
				return
			}
			if b.gapisTrace.Len() > 1 {
				// Skip the leading [
				_, err = f.Write(b.gapisTrace.Bytes()[1:])
				if err != nil {
					log.E(ctx, "Failed to write gapis trace data: %v", err)
					return
				}
			}
		}()
	}

	actions := []action{
		&sequential{
			label: "init",
			actions: []action{
				&startGapis{&verb.Gapis, &verb.Gapir, verb.DumpTrace != ""},
				loadCapture(flags.Arg(0)),
				loadReplaydevice(),
				&parallel{
					label: "load",
					actions: []action{
						loadResources(),
						loadCommandTree(),
						loadProfilingData(),
					},
				},
			},
		},
	}

	if len(verb.Paths) == 0 {
		draws := verb.NumDraws
		if draws == 0 {
			draws = defaultNumDraws
		}
		for i := 0; i < draws; i++ {
			actions = append(actions, selectCommandByAction(findDrawCall(nil, verb.Secondary), true))
		}
	} else {
		maxPathElements := 5
		if verb.Secondary {
			maxPathElements = 7
		}
		for _, path := range verb.Paths {
			draws := verb.NumDraws
			if len(path) > maxPathElements {
				return fmt.Errorf("Invalid path: %v - too long", path)
			} else if len(path) == maxPathElements {
				draws = 1
			} else if draws == 0 {
				draws = defaultNumDraws
			}

			for i := 0; i < draws; i++ {
				actions = append(actions, selectCommandByAction(findDrawCall(path, verb.Secondary), true))
			}
		}
	}

	group := &sequential{
		label:   "root",
		actions: actions,
	}

	_, err := group.exec(ctx, b)
	group.cleanup(ctx, b, err)
	if err != nil {
		return err
	}

	if verb.CsvOut != "" {
		if err := writeOutput(ctx, verb.CsvOut, b.measurement.writeCsv); err != nil {
			return err
		}
	}

	if verb.DotOut != "" {
		if err := writeOutput(ctx, verb.DotOut, b.measurement.writeGraph); err != nil {
			return err
		}
	}

	summary := b.measurement.computeSummary()
	if verb.SummaryOut != "" {
		if err := writeOutput(ctx, verb.SummaryOut, summary.print); err != nil {
			return err
		}
	} else {
		log.I(ctx, "-----------------------------------")
		if err := summary.print(log.From(ctx).Writer(log.Info)); err != nil {
			return err
		}
		log.I(ctx, "-----------------------------------")
	}

	return nil
}
