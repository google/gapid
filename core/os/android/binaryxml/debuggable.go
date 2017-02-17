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

package binaryxml

import (
	"fmt"
	"io"
)

const debuggableAttr uint32 = 0x0101000f

func startElementVisitor(path string, f func(*xmlContext, *xmlStartElement)) chunkVisitor {
	return func(ctx *xmlContext, c chunk, when int) {
		xse, ok := c.(*xmlStartElement)
		if ok && when == afterContextChange && ctx.path() == path {
			f(ctx, xse)
		}
	}
}

// setManifestApplicationDebuggable sets android:debuggable="true" under the <application/> element of the manifest.
// The function returns true on success. It will fail if it cannot find the application element.
func setManifestApplicationDebuggableAttributeToTrue(xml *xmlTree) (success bool) {
	success = false
	xml.visit(startElementVisitor("manifest/application", func(ctx *xmlContext, xse *xmlStartElement) {
		refDebuggable := xml.ensureAttributeNameMapsToResource(debuggableAttr, "debuggable")

		if at, ok := xse.attributes.forName(refDebuggable); ok {
			at.typedValue = valIntBoolean(true)
		} else {
			xse.addAttribute(&xmlAttribute{
				namespace:  ctx.strings.ref("http://schemas.android.com/apk/res/android"),
				name:       refDebuggable,
				rawValue:   invalidStringPoolRef,
				typedValue: valIntBoolean(true),
			})
		}
		success = true
	}))
	return success
}

// SetDebuggableFlag takes a Reader that produces a manifest binary xml,
// modifies it to set android:debuggable="true" under the <application/> element
// and writes it to the provided Writer.
func SetDebuggableFlag(r io.Reader, w io.Writer) error {
	tree, err := decodeXmlTree(r)
	if err != nil {
		return err

	}

	if !setManifestApplicationDebuggableAttributeToTrue(tree) {
		return fmt.Errorf("error modifying manifest")
	}
	_, err = w.Write(tree.encode())
	return err
}
