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
	"reflect"
	"strings"

	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/core/text"
	"github.com/google/gapid/gapis/api/gles/glsl"
	"github.com/google/gapid/gapis/api/gles/glsl/ast"
	"github.com/google/gapid/gapis/api/gles/glsl/parser"
	"github.com/google/gapid/gapis/api/gles/glsl/preprocessor"
	"github.com/google/gapid/gapis/database"
)

type glslTransform struct {
	stripPrecision           bool
	attributeToIn            bool
	varyingToIn              bool
	varyingToOut             bool
	depTextureFuncs          bool
	declareFragColor         bool
	declareFragData          bool
	renameSamplerExternalOES bool
	renameViewID             bool
	reservedNames            map[string]bool
}

const (
	compatFragColor = "FragColor"
	compatFragData  = "FragData"
)

type glslTransformState struct {
	glslTransform
	FragColor *ast.VariableSym
	ViewID    *ast.VariableSym
	FragData  map[int]*ast.VariableSym
}

func (t glslTransform) apply(n interface{}) {
	s := glslTransformState{
		glslTransform: t,
		FragData:      map[int]*ast.VariableSym{},
	}
	s.apply(n, nil)
}

var vec4 = &ast.BuiltinType{Type: ast.TVec4}

func (t *glslTransformState) apply(child, parent interface{}) interface{} {
	ast.TransformChildren(child, t.apply)

	addDecl := func(n *ast.Ast, v *ast.VariableSym, s ast.StorageQualifier) *ast.MultiVarDecl {
		decl := &ast.MultiVarDecl{
			Quals: &ast.TypeQualifiers{Storage: s},
			Type:  v.Type(),
			Vars:  []*ast.VariableSym{v},
		}
		decls := make([]interface{}, len(n.Decls)+1)
		copy(decls[1:], n.Decls)
		n.Decls = decls
		n.Decls[0] = decl
		return decl
	}
	fixReservedNames := func(name *string) {
		if t.depTextureFuncs {
			switch *name {
			case "texture", "textureProj", "textureLod", "textureProjLod",
				"shadow", "shadowProj", "shadowLod", "shadowProjLod":
				// These weren't keywords before the texture-lookup renames, but are now.
				// Rename these usages.
				*name += "_renamed_"
			}
		}
		if t.reservedNames[*name] {
			*name += "_renamed_"
		}
	}

	switch n := child.(type) {
	case *ast.Ast:
		if t.stripPrecision {
			n.Decls = removePrecisions(n.Decls)
		}
		if t.declareFragColor && t.FragColor != nil {
			addDecl(n, t.FragColor, ast.StorOut) // out vec4 FragColor;
		}
		if t.declareFragData {
			for idx, fragdata := range t.FragData {
				decl := addDecl(n, fragdata, ast.StorOut) // layout(location = N) out vec4 FragDataN;
				decl.Quals.Layout = &ast.LayoutQualifier{
					Ids: []ast.LayoutQualifierID{
						{Name: "location", Value: ast.IntValue(idx)},
					},
				}
			}
		}
		if t.ViewID != nil {
			addDecl(n, t.ViewID, ast.StorUniform)
		}
	case *ast.IfStmt:
		if t.stripPrecision {
			clearIfPrecision(&n.ThenStmt)
			clearIfPrecision(&n.ElseStmt)
		}
	case *ast.CompoundStmt:
		if t.stripPrecision {
			n.Stmts = removePrecisions(n.Stmts)
		}
	case *ast.WhileStmt:
		if t.stripPrecision {
			clearIfPrecision(&n.Stmt)
		}
	case *ast.DoStmt:
		if t.stripPrecision {
			clearIfPrecision(&n.Stmt)
		}
	case *ast.ForStmt:
		if t.stripPrecision {
			clearIfPrecision(&n.Init)
			clearIfPrecision(&n.Body)
		}
	case *ast.BuiltinType:
		if t.stripPrecision {
			n.Precision = ast.NoneP
		}
	case *ast.IndexExpr:
		if t.declareFragData {
			if vr, ok := n.Base.(*ast.VarRefExpr); ok {
				if v, ok := vr.Sym.(*ast.VariableSym); ok {
					if constExpr, ok := n.Index.(*ast.ConstantExpr); ok {
						if index, ok := constExpr.Value.(ast.IntValue); ok {
							idx := int(index)
							if v.Name() == "gl_FragData" {
								name := fmt.Sprintf("%s%d", compatFragData, idx)
								sym, found := t.FragData[idx]
								if !found {
									sym = &ast.VariableSym{SymType: vec4, SymName: name}
								}
								t.FragData[idx] = sym
								return &ast.VarRefExpr{Sym: sym}
							}
						}
					}
				}
			}
		}
	case *ast.FunctionDecl:
		if t.depTextureFuncs {
			// Texture-lookup functions got renamed.
			// Fix up old names to match the new names.
			var sym ast.Symbol
			switch n.Name() {
			case "texture1D", "texture1DProj", "texture1DLod", "texture1DProjLod":
				sym = parser.FindBuiltin(strings.Replace(n.Name(), "1D", "", -1)) // drop 1D

			case "texture2D", "texture2DProj", "texture2DLod", "texture2DProjLod":
				sym = parser.FindBuiltin(strings.Replace(n.Name(), "2D", "", -1)) // drop 2D

			case "texture3D", "texture3DProj", "texture3DLod", "texture3DProjLod":
				// TODO: For the proj versions, the texture coordinate is divided by coord.q.
				sym = parser.FindBuiltin(strings.Replace(n.Name(), "3D", "", -1)) // drop 3D

			case "textureCube", "textureCubeLod":
				sym = parser.FindBuiltin(strings.Replace(n.Name(), "Cube", "", -1)) // drop Cube

			case "shadow2DEXT":
				sym = parser.FindBuiltin("texture")

			case "shadow2DProjEXT":
				sym = parser.FindBuiltin("textureProj")
			}
			if v, ok := sym.(ast.ValueSymbol); ok {
				return v // replace usage
			}
		}
	case *ast.VariableSym:
		fixReservedNames(&n.SymName)
		if n.Quals != nil {
			switch {
			case n.Quals.Storage == ast.StorAttribute && t.attributeToIn:
				n.Quals.Storage = ast.StorIn
			case n.Quals.Storage == ast.StorVarying && t.varyingToOut:
				n.Quals.Storage = ast.StorOut
			case n.Quals.Storage == ast.StorVarying && t.varyingToIn:
				n.Quals.Storage = ast.StorIn
			}
		}
		if t.declareFragColor {
			if n.Name() == "gl_FragColor" {
				if t.FragColor == nil {
					t.FragColor = &ast.VariableSym{SymType: vec4, SymName: compatFragColor}
				}
				return t.FragColor // replace usage
			}
		}
		if t.renameSamplerExternalOES {
			if t, ok := n.Type().(*ast.BuiltinType); ok && t.Type == ast.TSamplerExternalOES {
				t.Type = ast.TSampler2D
			}
		}
		if t.renameViewID {
			if n.Name() == "gl_ViewID_OVR" {
				if t.ViewID == nil {
					t.ViewID = &ast.VariableSym{
						SymType: &ast.BuiltinType{Type: ast.TUint},
						SymName: "gapid_gl_ViewID_OVR",
					}
				}
				return t.ViewID // replace usage
			}
		}
	case *ast.LayoutDecl:
		if t.renameViewID {
			if n.Layout != nil {
				if ids := n.Layout.Ids; len(ids) == 1 && ids[0].Name == "num_views" {
					return ""
				}
			}
		}
	case *ast.FuncParameterSym:
		fixReservedNames(&n.SymName)
	case *ast.BinaryExpr:
		if left, ok := n.Left.(*ast.VarRefExpr); ok && left.Sym == t.FragColor {
			// Force cast RHS to vec4.
			// TODO: Only add cast if necessary.
			n.Right = &ast.CallExpr{
				Args:   []ast.Expression{n.Right},
				Callee: &ast.VarRefExpr{Sym: &ast.VariableSym{SymType: vec4, SymName: "vec4"}},
			}
		}
	}

	return child
}

type GLSLParseResult struct {
	Program interface{}
	Errors  []string
}

func (lt *GLSLParseResolvable) Resolve(ctx context.Context) (interface{}, error) {
	tree, _, _, errs := glsl.Parse(lt.ShaderSource, ast.Language(lt.Language))
	errors := make([]string, len(errs))
	for i, e := range errs {
		errors[i] = fmt.Sprintf("%v", e)
	}
	return &GLSLParseResult{Program: tree, Errors: errors}, nil
}

func glslCompat(
	ctx context.Context,
	src string,
	lang ast.Language,
	exts []preprocessor.Extension,
	device *device.Instance) (string, error) {

	robj, err := database.Build(
		ctx,
		&GLSLParseResolvable{
			ShaderSource: src,
			Language:     uint32(lang),
		},
	)
	r := robj.(*GLSLParseResult)
	if err != nil {
		return "", err
	}
	if len(r.Errors) > 0 {
		return "", fmt.Errorf("Failed to parse shader source:\n%s\n%s", text.LineNumber(src), r.Errors)
	}

	devGL := device.Configuration.Drivers.OpenGL

	deviceGLVersion, err := ParseVersion(devGL.Version)
	if err != nil {
		return "", fmt.Errorf("Could not parse GL version from '%s'. Error: %v",
			devGL.Version, err)
	}

	deviceGLSLVersion, err := GLSLVersion(devGL.Version)
	if err != nil {
		return "", fmt.Errorf("Could not get GLSL version from GL version: '%s'. Error: %v",
			devGL.Version, err)
	}

	// TODO: set targetGLSLVersion to a version closest to the original source
	// version, while still being maintained by the target device.
	targetGLSLVersion := deviceGLSLVersion

	transform := glslTransform{}
	if !deviceGLVersion.IsES {
		// Strip any precision specifiers
		transform.stripPrecision = true
		// ES-specific sampler - replace it with sampler2D
		transform.renameSamplerExternalOES = true
		transform.renameViewID = true
		// Rename identifiers which are keywords.
		// TODO: Handle all keyword differences.
		transform.reservedNames = map[string]bool{"sample": true}
	}

	if targetGLSLVersion.GreaterThan(1, 2) {
		switch lang {
		case ast.LangVertexShader:
			transform.attributeToIn = true
			transform.varyingToOut = true
		case ast.LangFragmentShader:
			transform.varyingToIn = true
			transform.declareFragColor = true
			transform.declareFragData = true
		}
		transform.depTextureFuncs = true
	}

	if (reflect.DeepEqual(transform, glslTransform{})) {
		return src, nil
	}

	transform.apply(r.Program)

	return glsl.Format(r.Program, targetGLSLVersion, exts), nil
}

func isPrecision(n interface{}) bool {
	switch n := n.(type) {
	case *ast.DeclarationStmt:
		return isPrecision(n.Decl)
	case *ast.PrecisionDecl:
		return true
	}
	return false
}

func removePrecisions(arr []interface{}) (ret []interface{}) {
	for _, n := range arr {
		if !isPrecision(n) {
			ret = append(ret, n)
		}
	}
	return
}

func clearIfPrecision(d *interface{}) {
	if isPrecision(*d) {
		*d = &ast.EmptyStmt{}
	}
}
