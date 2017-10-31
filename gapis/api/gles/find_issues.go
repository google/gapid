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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay"
	"github.com/google/gapid/gapis/replay/builder"
	"github.com/google/gapid/gapis/replay/value"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/shadertools"
)

// findIssues is a command transform that detects issues when replaying the
// stream of commands. Any issues that are found are written to all the chans in
// the slice out. Once the last issue is sent (if any) all the chans in out are
// closed.
type findIssues struct {
	state       *api.GlobalState
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

func (t *findIssues) onIssue(cmd api.Cmd, id api.CmdID, s service.Severity, e error) {
	if s == service.Severity_FatalLevel && isIssueWhitelisted(cmd, e) {
		s = service.Severity_ErrorLevel
	}
	t.issues = append(t.issues, replay.Issue{Command: id, Severity: s, Error: e})
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

func (t *findIssues) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	ctx = log.Enter(ctx, "findIssues")
	cb := CommandBuilder{Thread: cmd.Thread()}
	t.lastGlError = GLenum_GL_NO_ERROR
	mutateErr := cmd.Mutate(ctx, id, t.state, nil /* no builder */)

	mutatorsGlError := t.lastGlError
	if e := FindErrorState(cmd.Extras()); e != nil {
		// Check that our API file agrees with the driver which we used for tracking.
		if (e.TraceDriversGlError != GLenum_GL_NO_ERROR) != (mutatorsGlError != GLenum_GL_NO_ERROR) {
			errorMsg := ErrUnexpectedDriverTraceError{
				DriverError:   e.TraceDriversGlError,
				ExpectedError: mutatorsGlError,
			}
			t.onIssue(cmd, id, service.Severity_FatalLevel, errorMsg)
		}
		// Check that the C++ and Go versions of the generated code precisely agree.
		if e.InterceptorsGlError != mutatorsGlError {
			t.onIssue(cmd, id, service.Severity_FatalLevel, fmt.Errorf("%s in interceptor, but we expected %s",
				e.InterceptorsGlError.ErrorString(), mutatorsGlError.ErrorString()))
		}
	}

	if mutateErr != nil {
		// Ignore since downstream transform layers can only consume valid commands
		return
	}

	out.MutateAndWrite(ctx, id, cmd)

	dID := id.Derived()
	s := GetState(t.state)
	c := s.GetContext(cmd.Thread())
	if c == nil {
		return
	}

	// Check the result of glGetError after every command.
	out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
		ptr := b.AllocateTemporaryMemory(4)
		b.Call(funcInfoGlGetError)
		b.Store(ptr)
		b.Post(ptr, 4, builder.Postback(func(r binary.Reader, err error) error {
			if err != nil {
				return err
			}
			v := GLenum(r.Uint32())
			err = r.Error()
			if err != nil {
				t.onIssue(cmd, id, service.Severity_FatalLevel, fmt.Errorf("Failed to decode glGetError postback: %v", err))
				return err
			}
			if v != GLenum_GL_NO_ERROR {
				t.onIssue(cmd, id, service.Severity_FatalLevel, fmt.Errorf("%v in replay driver", v))
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

	switch cmd := cmd.(type) {
	case *GlCompileShader:
		shader := c.Objects.Shaders.Get(cmd.Shader)
		st, err := shader.Type.ShaderType()
		if err != nil {
			t.onIssue(cmd, id, service.Severity_ErrorLevel, err)
			return
		}
		opts := shadertools.Options{
			ShaderType:        st,
			CheckAfterChanges: true,
			Disassemble:       true,
		}

		if _, err := shadertools.ConvertGlsl(shader.Source, &opts); err != nil {
			t.onIssue(cmd, id, service.Severity_ErrorLevel, err)
		}

		const buflen = 8192
		tmp := t.state.AllocOrPanic(ctx, buflen)

		infoLog := make([]byte, buflen)
		out.MutateAndWrite(ctx, dID, cb.GlGetShaderInfoLog(cmd.Shader, buflen, memory.Nullptr, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
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
		out.MutateAndWrite(ctx, dID, cb.GlGetShaderSource(cmd.Shader, buflen, memory.Nullptr, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
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

		out.MutateAndWrite(ctx, dID, cb.GlGetShaderiv(cmd.Shader, GLenum_GL_COMPILE_STATUS, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
			b.ReserveMemory(tmp.Range())
			b.Post(value.ObservedPointer(tmp.Address()), 4, func(r binary.Reader, err error) error {
				if err != nil {
					return err
				}
				if r.Uint32() != uint32(GLboolean_GL_TRUE) {
					originalSource := "<unknown>"
					if shader := c.Objects.Shaders.Get(cmd.Shader); shader != nil {
						originalSource = shader.Source
					}
					t.onIssue(cmd, id, service.Severity_ErrorLevel, fmt.Errorf("Shader %d failed to compile. Error:\n%v\nOriginal source:\n%s\nTranslated source:\n%s\n",
						cmd.Shader, ntbs(infoLog), text.LineNumber(originalSource), text.LineNumber(ntbs(source))))
				}
				return r.Error()
			})
			return nil
		}))
		tmp.Free()

	case *GlLinkProgram:
		const buflen = 2048
		tmp := t.state.AllocOrPanic(ctx, 4+buflen)
		out.MutateAndWrite(ctx, dID, cb.GlGetProgramiv(cmd.Program, GLenum_GL_LINK_STATUS, tmp.Ptr()))
		out.MutateAndWrite(ctx, dID, cb.GlGetProgramInfoLog(cmd.Program, buflen, memory.Nullptr, tmp.Offset(4)))
		out.MutateAndWrite(ctx, dID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
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
					if program := c.Objects.Programs.Get(cmd.Program); program != nil {
						if shader := program.Shaders.Get(GLenum_GL_VERTEX_SHADER); shader != nil {
							vss = shader.Source
						}
						if shader := program.Shaders.Get(GLenum_GL_FRAGMENT_SHADER); shader != nil {
							fss = shader.Source
						}
					}
					logLevel := service.Severity_ErrorLevel
					if pi := FindProgramInfo(cmd.Extras()); pi != nil && pi.LinkStatus == GLboolean_GL_TRUE {
						// Increase severity if the program linked successfully during trace.
						logLevel = service.Severity_FatalLevel
					}
					t.onIssue(cmd, id, logLevel, fmt.Errorf("Program %d failed to link. Error:\n%v\n"+
						"Vertex shader source:\n%sFragment shader source:\n%s", cmd.Program, ntbs(msg),
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
			t.onIssue(cmd, id, service.Severity_WarningLevel, fmt.Errorf("Pre-compiled binaries cannot be used across on different devices. Capture: %s-%s, Replay: %s-%s",
				c.Constants.Vendor, c.Constants.Version, glDev.Vendor, glDev.Version))
		}
	}
}

func (t *findIssues) Flush(ctx context.Context, out transform.Writer) {
	cb := CommandBuilder{Thread: 0}
	out.MutateAndWrite(ctx, api.CmdNoID, cb.Custom(func(ctx context.Context, s *api.GlobalState, b *builder.Builder) error {
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
