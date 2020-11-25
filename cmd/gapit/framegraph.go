// Copyright (C) 2020 Google Inc.
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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
)

type framegraphVerb struct{ FramegraphFlags }

func init() {
	verb := &framegraphVerb{}
	app.AddVerb(&app.Verb{
		Name:      "framegraph",
		ShortHelp: "Get the frame graph of a capture",
		Action:    verb,
	})
}

// Run is the main logic for the 'gapit framegraph' command.
func (verb *framegraphVerb) Run(ctx context.Context, flags flag.FlagSet) error {
	if flags.NArg() != 1 {
		app.Usage(ctx, "Exactly one gfx trace file expected, got %d", flags.NArg())
		return nil
	}

	if verb.Dot == "" && verb.Json == "" {
		app.Usage(ctx, "At least one of -dot or -json flag is expected")
		return nil
	}

	captureFilename := flags.Arg(0)

	client, capture, err := getGapisAndLoadCapture(ctx, verb.Gapis, GapirFlags{}, captureFilename, CaptureFileFlags{})
	if err != nil {
		return err
	}
	defer client.Close()

	boxedFramegraph, err := client.Get(ctx, capture.Framegraph().Path(), nil)
	if err != nil {
		return err
	}
	framegraph := boxedFramegraph.(*api.Framegraph)

	if verb.Json != "" {
		err = exportJSON(ctx, framegraph, verb.Json)
	}
	if err != nil {
		return err
	}

	if verb.Dot != "" {
		err = exportDot(ctx, framegraph, captureFilename, verb.Dot)
	}

	return err
}

func exportJSON(ctx context.Context, framegraph *api.Framegraph, outFile string) error {
	file, err := os.Create(outFile)
	if err != nil {
		return log.Errf(ctx, err, "Creating file (%v)", outFile)
	}
	defer file.Close()

	m := &jsonpb.Marshaler{
		EmitDefaults: true,
		Indent:       " ",
	}
	return m.Marshal(file, framegraph)
}

// exportDot exports the framegraph in the Graphviz DOT format.
// https://graphviz.org/doc/info/lang.html
// In node labels we use "\l" as a newline to obtain left-aligned text.
func exportDot(ctx context.Context, framegraph *api.Framegraph, captureFilename string, outFile string) error {
	file, err := os.Create(outFile)
	if err != nil {
		return log.Errf(ctx, err, "Creating file (%v)", outFile)
	}
	defer file.Close()

	fmt.Fprintf(file, "digraph agiFramegraph {\n")

	// Graph title: use capture filename, on top
	fmt.Fprintf(file, "label = \"%s\";\n", captureFilename)
	fmt.Fprintf(file, "labelloc = \"t\";\n")
	// Node style
	fmt.Fprintf(file, "node [fontname=Monospace shape=rectangle];\n")
	fmt.Fprintf(file, "\n")
	// Node IDs cannot start with a digit, so use "n<node.Id>", e.g. n0 n1 n2
	for _, node := range framegraph.Nodes {
		fmt.Fprintf(file, fmt.Sprintf("n%v [label=\"%s\"];\n", node.Id, node2dot(node)))
	}
	fmt.Fprintf(file, "\n")
	for _, edge := range framegraph.Edges {
		fmt.Fprintf(file, fmt.Sprintf("n%v -> n%v;\n", edge.Origin, edge.Destination))
	}
	fmt.Fprintf(file, "}\n")
	return nil
}

func image2dot(img *api.FramegraphImage) string {
	usage := ""

	if img.TransferSrc {
		usage += " TransferSrc"
	}
	if img.TransferDst {
		usage += " TransferDst"
	}
	if img.Sampled {
		usage += " Sampled"
	}
	if img.Storage {
		usage += " Storage"
	}
	if img.ColorAttachment {
		usage += " ColorAttachment"
	}
	if img.DepthStencilAttachment {
		usage += " DepthStencilAttachment"
	}
	if img.TransientAttachment {
		usage += " TransientAttachment"
	}
	if img.InputAttachment {
		usage += " InputAttachment"
	}
	if img.Swapchain {
		usage += " Swapchain"
	}

	imgType := strings.TrimPrefix(fmt.Sprintf("%v", img.ImageType), "VK_IMAGE_TYPE_")
	imgFormat := strings.TrimPrefix(fmt.Sprintf("%v", img.Info.Format.Name), "VK_FORMAT_")
	return fmt.Sprintf("[Img:%v %s %s %vx%vx%v usage:%v%s]", img.Handle, imgType, imgFormat, img.Info.Width, img.Info.Height, img.Info.Depth, img.Usage, usage)
}

func attachment2dot(att *api.FramegraphAttachment) string {
	if att == nil {
		return "unused"
	}
	return fmt.Sprintf("load:%v store:%v %s", att.LoadOp, att.StoreOp, image2dot(att.Image))
}

func imageAccess2dot(acc *api.FramegraphImageAccess) string {
	r := "-"
	if acc.Read {
		r = "r"
	}
	w := "-"
	if acc.Write {
		w = "w"
	}
	return fmt.Sprintf("%s%s %s", r, w, image2dot(acc.Image))
}

func buffer2dot(buf *api.FramegraphBuffer) string {
	usage := ""

	if buf.TransferSrc {
		usage += " TransferSrc"
	}
	if buf.TransferDst {
		usage += " TransferDst"
	}
	if buf.UniformTexel {
		usage += " UniformTexel"
	}
	if buf.StorageTexel {
		usage += " StorageTexel"
	}
	if buf.Uniform {
		usage += " Uniform"
	}
	if buf.Storage {
		usage += " Storage"
	}
	if buf.Index {
		usage += " Index"
	}
	if buf.Vertex {
		usage += " Vertex"
	}
	if buf.Indirect {
		usage += " Indirect"
	}

	return fmt.Sprintf("[Buf:%v size:%v usage:%v%s]", buf.Handle, buf.Size, buf.Usage, usage)
}

func bufferAccess2dot(acc *api.FramegraphBufferAccess) string {
	r := "-"
	if acc.Read {
		r = "r"
	}
	w := "-"
	if acc.Write {
		w = "w"
	}
	return fmt.Sprintf("%s%s %s", r, w, buffer2dot(acc.Buffer))
}

func renderpass2dot(rp *api.FramegraphRenderpass) string {
	s := fmt.Sprintf("Renderpass %v\\lbegin:%v\\lend:  %v\\lFramebuffer: %vx%vx%v\\l", rp.Handle, rp.BeginSubCmdIdx, rp.EndSubCmdIdx, rp.FramebufferWidth, rp.FramebufferHeight, rp.FramebufferLayers)
	for i, subpass := range rp.Subpass {
		s += fmt.Sprintf("\\lSubpass %v\\l", i)
		for j, a := range subpass.Input {
			s += fmt.Sprintf("input(%v): %v\\l", j, attachment2dot(a))
		}
		for j, a := range subpass.Color {
			s += fmt.Sprintf("color(%v): %v\\l", j, attachment2dot(a))
		}
		for j, a := range subpass.Resolve {
			s += fmt.Sprintf("resolve(%v): %v\\l", j, attachment2dot(a))
		}
		s += fmt.Sprintf("depth/stencil: %v\\l", attachment2dot(subpass.DepthStencil))
	}

	if len(rp.ImageAccess) > 0 {
		s += "\\lImage accesses:\\l"
		for _, acc := range rp.ImageAccess {
			s += fmt.Sprintf("%s\\l", imageAccess2dot(acc))
		}
	}

	if len(rp.BufferAccess) > 0 {
		s += "\\lBuffer accesses:\\l"
		for _, acc := range rp.BufferAccess {
			s += fmt.Sprintf("%s\\l", bufferAccess2dot(acc))
		}
	}

	return s
}

func compute2dot(compute *api.FramegraphCompute) string {
	s := fmt.Sprintf("Compute\\lcmd:%v\\l", compute.SubCmdIdx)
	if compute.Indirect {
		s += "Indirect\\l"
	} else {
		s += fmt.Sprintf("BaseGroupX:%v\\lBaseGroupY:%v\\lBaseGroupZ:%v\\lGroupCountX:%v\\lGroupCountY:%v\\lGroupCountZ:%v\\l", compute.BaseGroupX, compute.BaseGroupY, compute.BaseGroupZ, compute.GroupCountX, compute.GroupCountY, compute.GroupCountZ)
	}

	if len(compute.ImageAccess) > 0 {
		s += "\\lImage accesses:\\l"
		for _, acc := range compute.ImageAccess {
			s += fmt.Sprintf("%s\\l", imageAccess2dot(acc))
		}
	}

	if len(compute.BufferAccess) > 0 {
		s += "\\lBuffer accesses:\\l"
		for _, acc := range compute.BufferAccess {
			s += fmt.Sprintf("%s\\l", bufferAccess2dot(acc))
		}
	}

	return s
}

func node2dot(node *api.FramegraphNode) string {
	if renderpass := node.GetRenderpass(); renderpass != nil {
		return renderpass2dot(renderpass)
	}
	if compute := node.GetCompute(); compute != nil {
		return compute2dot(compute)
	}
	return "INVALID NODE: neither renderpass nor compute"
}
