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

package gles

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/gapid/core/data/binary"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
)

// findIssues is an atom transform that detects issues when replaying the
// stream of atoms. Any issues that are found are written to all the chans in
// the slice out. Once the last issue is sent (if any) all the chans in out are
// closed.
type findIssues struct {
	state       *gfxapi.State
	device      *device.Instance
	issues      []replay.Issue
	res         []replay.Result
	lastGlError GLenum
}

func newFindIssues(ctx context.Context, c *capture.Capture, device *device.Instance) *findIssues {
	transform := &findIssues{
		state:  c.NewState(),
		device: device,
	}
	transform.state.OnError = func(err interface{}) {
		transform.lastGlError = err.(GLenum)
	}
	return transform
}

// reportTo adds the chan c to the list of issue listeners.
func (t *findIssues) reportTo(r replay.Result) { t.res = append(t.res, r) }

func (t *findIssues) onIssue(a atom.Atom, i atom.ID, s service.Severity, e error) {
	if s == service.Severity_FatalLevel && isIssueWhitelisted(a, e) {
		s = service.Severity_ErrorLevel
	}
	t.issues = append(t.issues, replay.Issue{Atom: i, Severity: s, Error: e})
}

// The value 0 is used for many enums - prefer GL_NO_ERROR in this case.
func (e GLenum) ErrorString() string {
	if e == GLenum_GL_NO_ERROR {
		return "GL_NO_ERROR"
	}
	return e.String()
}

type ErrUnexpectedDriverTraceError struct {
	DriverError   GLenum
	ExpectedError GLenum
}

func (e ErrUnexpectedDriverTraceError) Error() string {
	return fmt.Sprintf("%s in trace driver, but we expected %s",
		e.DriverError.ErrorString(), e.ExpectedError.ErrorString())
}

func (t *findIssues) Transform(ctx context.Context, i atom.ID, a atom.Atom, out transform.Writer) {
	ctx = log.Enter(ctx, "findIssues")
	t.lastGlError = GLenum_GL_NO_ERROR
	if err := a.Mutate(ctx, t.state, nil /* no builder */); err != nil {
		if atom.IsAbortedError(err) && t.lastGlError != GLenum_GL_NO_ERROR {
			// GL errors have already been reported - so do not log it again.
		} else {
			t.onIssue(a, i, service.Severity_ErrorLevel, err)
		}
	}
	mutatorsGlError := t.lastGlError
	out.MutateAndWrite(ctx, i, a)

	dID := i.Derived()
	s := GetState(t.state)
	c := s.getContext()
	if c == nil {
		return
	}

	// Check the result of glGetError after every command.
	out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
		ptr := b.AllocateTemporaryMemory(4)
		b.Call(funcInfoGlGetError)
		b.Store(ptr)
		b.Post(ptr, 4, builder.Postback(func(r binary.Reader, err error) error {
			if err != nil {
				return err
			}
			v := r.Uint32()
			err = r.Error()
			if err != nil {
				t.onIssue(a, i, service.Severity_FatalLevel, fmt.Errorf("Failed to decode glGetError postback: %v", err))
				return err
			}
			replayDriversGlError := GLenum(v)
			if e := FindErrorState(a.Extras()); e != nil {
				// Check that our API file agrees with the driver which we used for tracking.
				if (e.TraceDriversGlError != GLenum_GL_NO_ERROR) != (mutatorsGlError != GLenum_GL_NO_ERROR) {
					err := ErrUnexpectedDriverTraceError{
						DriverError:   e.TraceDriversGlError,
						ExpectedError: mutatorsGlError,
					}
					t.onIssue(a, i, service.Severity_FatalLevel, err)
				}
				// Check that the C++ and Go versions of the generated code precisely agree.
				if e.InterceptorsGlError != mutatorsGlError {
					t.onIssue(a, i, service.Severity_FatalLevel, fmt.Errorf("%s in interceptor, but we expected %s",
						e.InterceptorsGlError.ErrorString(), mutatorsGlError.ErrorString()))
				}
			}
			// The compatibility layer should not produce errors but it can propagate or silence them.
			if (replayDriversGlError != GLenum_GL_NO_ERROR) && (mutatorsGlError == GLenum_GL_NO_ERROR) {
				t.onIssue(a, i, service.Severity_FatalLevel, fmt.Errorf("%s in replay driver, but we expected %s",
					replayDriversGlError.ErrorString(), mutatorsGlError.ErrorString()))
			}
			return nil
		}))
		return nil
	}))

	// null-terminated byte slice to string
	ntbs := func(b []byte) string {
		s := string(b)
		for i, r := range s {
			if r == 0 {
				return strings.TrimSpace(s[:i])
			}
		}
		return strings.TrimSpace(s)
	}

	// Do not do any further checks if we have already determined the command fails.
	// Furthermore, invalid commands run into risk of accessing invalid state.
	if mutatorsGlError != GLenum_GL_NO_ERROR {
		return
	}

	switch a := a.(type) {
	case *GlShaderSource:
		if !config.UseGlslang { // TODO: Verify shader with glslang
			shader := c.SharedObjects.Shaders[a.Shader]
			var errs []error
			var kind string
			switch shader.Type {
			case GLenum_GL_VERTEX_SHADER:
				_, _, _, errs = glsl.Parse(shader.Source, ast.LangVertexShader)
				kind = "vertex"
			case GLenum_GL_FRAGMENT_SHADER:
				_, _, _, errs = glsl.Parse(shader.Source, ast.LangFragmentShader)
				kind = "fragment"
			default:
				t.onIssue(a, i, service.Severity_ErrorLevel, fmt.Errorf("Unknown shader type %v", shader.Type))
			}
			if len(errs) > 0 {
				msgs := make([]string, len(errs))
				for i, err := range errs {
					msgs[i] = err.Error()
				}
				t.onIssue(a, i, service.Severity_ErrorLevel, fmt.Errorf("Failed to parse %s shader source. Errors:\n%s\nSource:\n%s",
					kind, strings.Join(msgs, "\n"), text.LineNumber(shader.Source)))
			}
		}

	case *GlCompileShader:
		const buflen = 8192
		tmp := atom.Must(atom.Alloc(ctx, t.state, buflen))

		infoLog := make([]byte, buflen)
		out.MutateAndWrite(ctx, dID, NewGlGetShaderInfoLog(a.Shader, buflen, memory.Nullptr, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
			b.ReserveMemory(tmp.Range())
			b.Post(value.ObservedPointer(tmp.Address()), buflen, func(r binary.Reader, err error) error {
				if err != nil {
					return err
				}
				r.Data(infoLog)
				return r.Error()
			})
			return nil
		}))

		source := make([]byte, buflen)
		out.MutateAndWrite(ctx, dID, NewGlGetShaderSource(a.Shader, buflen, memory.Nullptr, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
			b.ReserveMemory(tmp.Range())
			b.Post(value.ObservedPointer(tmp.Address()), buflen, func(r binary.Reader, err error) error {
				if err != nil {
					return err
				}
				r.Data(source)
				return r.Error()
			})
			return nil
		}))

		out.MutateAndWrite(ctx, dID, NewGlGetShaderiv(a.Shader, GLenum_GL_COMPILE_STATUS, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
			b.ReserveMemory(tmp.Range())
			b.Post(value.ObservedPointer(tmp.Address()), 4, func(r binary.Reader, err error) error {
				if err != nil {
					return err
				}
				if r.Uint32() != uint32(GLboolean_GL_TRUE) {
					originalSource := "<unknown>"
					if shader := c.SharedObjects.Shaders[a.Shader]; shader != nil {
						originalSource = shader.Source
					}
					t.onIssue(a, i, service.Severity_ErrorLevel, fmt.Errorf("Shader %d failed to compile. Error:\n%v\nOriginal source:\n%s\nTranslated source:\n%s\n",
						a.Shader, ntbs(infoLog), text.LineNumber(originalSource), text.LineNumber(ntbs(source))))
				}
				return r.Error()
			})
			return nil
		}))
		tmp.Free()

	case *GlLinkProgram:
		const buflen = 2048
		tmp := atom.Must(atom.Alloc(ctx, t.state, 4+buflen))
		out.MutateAndWrite(ctx, dID, NewGlGetProgramiv(a.Program, GLenum_GL_LINK_STATUS, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, NewGlGetProgramInfoLog(a.Program, buflen, memory.Nullptr, tmp.Offset(4)))
		out.MutateAndWrite(ctx, dID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
			b.ReserveMemory(tmp.Range())
			b.Post(value.ObservedPointer(tmp.Address()), 4+buflen, func(r binary.Reader, err error) error {
				if err != nil {
					return err
				}
				msg := make([]byte, buflen)
				res := r.Uint32()
				r.Data(msg)
				if res != uint32(GLboolean_GL_TRUE) {
					vss, fss := "<unknown>", "<unknown>"
					if program := c.SharedObjects.Programs[a.Program]; program != nil {
						if shader := program.Shaders[GLenum_GL_VERTEX_SHADER]; shader != nil {
							vss = shader.Source
						}
						if shader := program.Shaders[GLenum_GL_FRAGMENT_SHADER]; shader != nil {
							fss = shader.Source
						}
					}
					logLevel := service.Severity_ErrorLevel
					if pi := FindProgramInfo(a.Extras()); pi != nil && pi.LinkStatus == GLboolean_GL_TRUE {
						// Increase severity if the program linked successfully during trace.
						logLevel = service.Severity_FatalLevel
					}
					t.onIssue(a, i, logLevel, fmt.Errorf("Program %d failed to link. Error:\n%v\n"+
						"Vertex shader source:\n%sFragment shader source:\n%s", a.Program, ntbs(msg),
						text.LineNumber(vss), text.LineNumber(fss)))
				}
				return r.Error()
			})
			return nil
		}))
		tmp.Free()

	case *GlProgramBinary, *GlProgramBinaryOES, *GlShaderBinary:
		glDev := t.device.Configuration.Drivers.OpenGL
		if !canUsePrecompiledShader(c, glDev) {
			t.onIssue(a, i, service.Severity_WarningLevel, fmt.Errorf("Pre-compiled binaries cannot be used across on different devices. Capture: %s-%s, Replay: %s-%s",
				c.Constants.Vendor, c.Constants.Version, glDev.Vendor, glDev.Version))
		}
	}
}

func (t *findIssues) Flush(ctx context.Context, out transform.Writer) {
	out.MutateAndWrite(ctx, atom.NoID, replay.Custom(func(ctx context.Context, s *gfxapi.State, b *builder.Builder) error {
		// Since the PostBack function is called before the replay target has actually arrived at the post command,
		// we need to actually write some data here. r.Uint32() is what actually waits for the replay target to have
		// posted the data in question. If we did not do this, we would shut-down the replay as soon as the second-to-last
		// Post had occurred, which may not be anywhere near the end of the stream.
		code := uint32(0xbeefcace)
		b.Push(value.U32(code))
		b.Post(b.Buffer(1), 4, func(r binary.Reader, err error) error {
			if err != nil {
				t.res = nil
				return err
			}
			if r.Uint32() != code {
				return fmt.Errorf("Flush did not get expected EOS code")
			}
			for _, res := range t.res {
				res(t.issues, nil)
			}
			return err
		})
		return nil
	}))
}
