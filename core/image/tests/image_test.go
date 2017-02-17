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

package tests

import (
	"io/ioutil"
	"testing"

	"github.com/google/gapid/core/image"
)

func TestDecompressors(t *testing.T) {
	rgba, png := image.RGBA_U8_NORM, image.PNG
	for _, test := range []struct {
		name   string
		format *image.Format
		w, h   uint32
	}{
		{name: "etc1_rgb8", format: image.ETC1_RGB8, w: 700, h: 530},
		{name: "etc2_rgb8", format: image.ETC2_RGB8, w: 700, h: 530},
		{name: "etc2_rgb8_eac", format: image.ETC2_RGBA8_EAC, w: 700, h: 530},
		{name: "s3_dxt1_rgb", format: image.S3_DXT1_RGB, w: 700, h: 530},
		{name: "s3_dxt1_rgba", format: image.S3_DXT1_RGBA, w: 700, h: 530},
		{name: "s3_dxt3_rgba", format: image.S3_DXT3_RGBA, w: 700, h: 530},
		{name: "s3_dxt5_rgba", format: image.S3_DXT5_RGBA, w: 700, h: 530},
	} {
		inPath := test.name + ".bin"
		refPath := test.name + ".png"

		inData, err := ioutil.ReadFile(inPath)
		if err != nil {
			t.Errorf("Failed to read '%s': %v", inPath, err)
		}
		in := image.Image2D{Data: inData, Width: test.w, Height: test.h, Format: test.format}

		refPNGData, err := ioutil.ReadFile(refPath)
		if err != nil {
			t.Errorf("Failed to read  '%s': %v", refPath, err)
		}
		refPNG := image.Image2D{Data: refPNGData, Width: test.w, Height: test.h, Format: png}

		ref, err := refPNG.Convert(rgba)
		if err != nil {
			t.Errorf("Failed to convert '%s' from %v to %v: %v", refPath, png, rgba, err)
		}

		out, err := in.Convert(rgba)
		if err != nil {
			t.Errorf("Failed to convert '%s' from %v to %v: %v", inPath, test.format, rgba, err)
		}

		diff, err := image.Difference(out, ref)
		if err != nil {
			t.Errorf("Difference returned error: %v", err)
		}

		if diff != 0 {
			t.Errorf("%v produced unexpected difference when decompressing (%v)", test.name, diff)
			if outPNG, err := out.Convert(png); err == nil {
				ioutil.WriteFile(test.name+"-output.png", outPNG.Data, 0666)
			} else {
				t.Errorf("Could not write output file: %v", err)
			}
			for i := range out.Data {
				g, e := int(out.Data[i]), int(ref.Data[i])
				if g != e {
					out.Data[i] = 255 // Highlight errors
				}
			}
			if outPNG, err := out.Convert(png); err == nil {
				ioutil.WriteFile(test.name+"-error.png", outPNG.Data, 0666)
			} else {
				t.Errorf("Could not write error file: %v", err)
			}

		}
	}
}
