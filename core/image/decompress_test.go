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

package image_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/gapid/core/data/endian"
	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/astc"
	"github.com/google/gapid/core/math/sint"
	"github.com/google/gapid/core/os/device"
)

// See: https://www.khronos.org/opengles/sdk/tools/KTX/file_format_spec/
func loadKTX(data []byte) (*image.Data, error) {
	r := endian.Reader(bytes.NewBuffer(data), device.LittleEndian)

	var ident [12]byte
	r.Data(ident[:])
	if ident != [12]byte{0xAB, 0x4B, 0x54, 0x58, 0x20, 0x31, 0x31, 0xBB, 0x0D, 0x0A, 0x1A, 0x0A} {
		return nil, fmt.Errorf("Invalid header. Got: %x", ident)
	}

	if endianness := r.Uint32(); endianness != 0x04030201 {
		return nil, fmt.Errorf("Unexpected endianness")
	}

	glType := r.Uint32()
	glTypeSize := r.Uint32()
	glFormat := r.Uint32()
	glInternalFormat := r.Uint32()
	glBaseInternalFormat := r.Uint32()
	texelWidth := r.Uint32()
	texelHeight := r.Uint32()
	pixelDepth := r.Uint32()
	numberOfArrayElements := r.Uint32()
	numberOfFaces := r.Uint32()
	numberOfMipmapLevels := r.Uint32()
	bytesOfKeyValueData := r.Uint32()

	for keyValueOffset := uint32(0); keyValueOffset < bytesOfKeyValueData; {
		keyAndValueByteSize := r.Uint32()
		keyAndValue := make([]byte, keyAndValueByteSize)
		r.Data(keyAndValue)
		padding := make([]byte, 3-((keyAndValueByteSize+3)%4))
		r.Data(padding)
		keyValueOffset += 4 + keyAndValueByteSize + uint32(len(padding))
	}

	if numberOfMipmapLevels != 1 {
		return nil, fmt.Errorf("Cannot handle multiple mipmap levels (%v)", numberOfMipmapLevels)
	}
	if numberOfArrayElements != 0 {
		return nil, fmt.Errorf("Cannot handle array elements (%v)", numberOfArrayElements)
	}
	if numberOfFaces != 1 {
		return nil, fmt.Errorf("Cannot handle multiple faces (%v)", numberOfFaces)
	}
	if pixelDepth != 0 {
		return nil, fmt.Errorf("Cannot handle 3D textures (%v)", pixelDepth)
	}

	formats := map[uint32]*image.Format{
		0x9270: image.ETC2_R_U11_NORM,          // GL_COMPRESSED_R11_EAC
		0x9271: image.ETC2_R_S11_NORM,          // GL_COMPRESSED_SIGNED_R11_EAC
		0x9272: image.ETC2_RG_U11_NORM,         // GL_COMPRESSED_RG11_EAC
		0x9273: image.ETC2_RG_S11_NORM,         // GL_COMPRESSED_SIGNED_RG11_EAC
		0x9274: image.ETC2_RGB_U8_NORM,         // GL_COMPRESSED_RGB8_ETC2
		0x9275: image.ETC2_SRGB_U8_NORM,        // GL_COMPRESSED_SRGB8_ETC2
		0x9276: image.ETC2_RGBA_U8U8U8U1_NORM,  // GL_COMPRESSED_RGB8_PUNCHTHROUGH_ALPHA1_ETC2
		0x9277: image.ETC2_SRGBA_U8U8U8U1_NORM, // GL_COMPRESSED_SRGB8_PUNCHTHROUGH_ALPHA1_ETC2
		0x9278: image.ETC2_RGBA_U8_NORM,        // GL_COMPRESSED_RGBA8_ETC2_EAC
		0x9279: image.ETC2_SRGBA_U8_NORM,       // GL_COMPRESSED_SRGB8_ALPHA8_ETC2_EAC
	}
	format, ok := formats[glInternalFormat]
	if !ok {
		panic(fmt.Errorf(`Unsupported KTX format:
glType=0x%x
glTypeSize=0x%x
glFormat=0x%x
glInternalFormat=0x%x
glBaseInternalFormat=0x%x
`, glType, glTypeSize, glFormat, glInternalFormat, glBaseInternalFormat))
	}

	imageSize := r.Uint32()
	texelData := make([]byte, imageSize)
	r.Data(texelData)

	if err := r.Error(); err != nil {
		return nil, err
	}

	return &image.Data{
		Format: format,
		Width:  texelWidth,
		Height: texelHeight,
		Depth:  1,
		Bytes:  texelData,
	}, nil
}

func loadASTC(data []byte) (*image.Data, error) {
	r := endian.Reader(bytes.NewBuffer(data), device.LittleEndian)

	if got := r.Uint32(); got != 0x5ca1ab13 {
		return nil, fmt.Errorf("Invalid header. Got: %x", got)
	}

	blockWidth := uint32(r.Uint8())
	blockHeight := uint32(r.Uint8())
	blockDepth := uint32(r.Uint8())

	if blockDepth != 1 {
		return nil, fmt.Errorf("Got a block depth of %v. Only 2D textures are currently supported", blockDepth)
	}

	texelWidth := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())
	texelHeight := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())
	texelDepth := uint32(r.Uint8()) + 0x100*uint32(r.Uint8()) + 0x10000*uint32(r.Uint8())

	blocksX := (texelWidth + blockWidth - 1) / blockWidth
	blocksY := (texelHeight + blockHeight - 1) / blockHeight
	blocksZ := (texelDepth + blockDepth - 1) / blockDepth

	texelData := make([]byte, blocksX*blocksY*blocksZ*16)
	r.Data(texelData)

	if err := r.Error(); err != nil {
		return nil, err
	}

	return &image.Data{
		Format: image.NewASTC("astc", blockWidth, blockHeight, false),
		Width:  texelWidth,
		Height: texelHeight,
		Depth:  texelDepth,
		Bytes:  texelData,
	}, nil
}

func TestDecompressors(t *testing.T) {
	// For these tests we need to check that the S16_NORM formats match the
	// U8_NORM PNGs. There's no generic way to do this, so we declare our
	// expected converter here.
	image.RegisterConverter(image.R_S16_NORM, image.RGBA_U8_NORM, s16ToU8)
	image.RegisterConverter(image.RG_S16_NORM, image.RGBA_U8_NORM, s16ToU8)

	for _, test := range []struct {
		fmt *image.Format
		ext string
	}{
		{image.ETC2_RGBA_U8U8U8U1_NORM, ".ktx"},
		{image.ETC2_RGBA_U8_NORM, ".ktx"},
		{image.ETC2_RGB_U8_NORM, ".ktx"},
		{image.ETC2_RG_S11_NORM, ".ktx"},
		{image.ETC2_RG_U11_NORM, ".ktx"},
		{image.ETC2_R_S11_NORM, ".ktx"},
		{image.ETC2_R_U11_NORM, ".ktx"},
		{image.S3_DXT1_RGB, ".bin"},
		{image.S3_DXT1_RGBA, ".bin"},
		{image.S3_DXT3_RGBA, ".bin"},
		{image.S3_DXT5_RGBA, ".bin"},
		{astc.RGBA_4x4, ".astc"},
		{image.RGTC1_BC4_R_U8_NORM, ".bin"},
		{image.RGTC1_BC4_R_S8_NORM, ".bin"},
		{image.RGTC2_BC5_RG_U8_NORM, ".bin"},
		{image.RGTC2_BC5_RG_S8_NORM, ".bin"},
	} {
		name := test.fmt.Name
		inPath := filepath.Join("test_data", name+test.ext)
		refPath := filepath.Join("test_data", name+".png")

		refPNGData, err := ioutil.ReadFile(refPath)
		if err != nil {
			t.Errorf("Failed to read  '%s': %v", refPath, err)
			continue
		}
		refPNG, err := image.PNGFrom(refPNGData)
		if err != nil {
			t.Errorf("Failed to read PNG '%s': %v", refPath, err)
			continue
		}

		ref, err := refPNG.Convert(image.RGBA_U8_NORM)
		if err != nil {
			t.Errorf("Failed to convert '%s' from PNG to %v: %v", refPath, image.RGBA_U8_NORM, err)
			continue
		}

		data, err := ioutil.ReadFile(inPath)
		if err != nil {
			t.Errorf("Failed to read '%s': %v", inPath, err)
			continue
		}

		var in *image.Data
		switch test.ext {
		case ".ktx":
			ktx, err := loadKTX(data)
			if err != nil {
				t.Errorf("Failed to read '%s': %v", inPath, err)
				continue
			}

			if ktx.Format.Key() != test.fmt.Key() {
				t.Errorf("%v was not the expected format. Expected %v, got %v",
					inPath, test.fmt, ktx.Format)
				continue
			}
			in = ktx

		case ".astc":
			astc, err := loadASTC(data)
			if err != nil {
				t.Errorf("Failed to read '%s': %v", inPath, err)
				continue
			}

			if astc.Format.Key() != test.fmt.Key() {
				t.Errorf("%v was not the expected format. Expected %v, got %v",
					inPath, test.fmt, astc.Format)
				continue
			}
			in = astc

		case ".bin":
			in = &image.Data{
				Bytes:  data,
				Width:  ref.Width,
				Height: ref.Height,
				Depth:  1,
				Format: test.fmt,
			}

		default:
			panic("Unknown extension: " + test.ext)
		}

		out, err := in.Convert(image.RGBA_U8_NORM)
		if err != nil {
			t.Errorf("Failed to convert '%s' from %v to %v: %v", inPath, test.fmt.Name, image.RGBA_U8_NORM.Name, err)
			continue
		}

		diff, err := image.Difference(out, ref)
		if err != nil {
			t.Errorf("Difference returned error: %v", err)
			continue
		}

		outputPath := test.fmt.Name + "-output.png"
		errorPath := test.fmt.Name + "-error.png"
		if diff != 0 {
			t.Errorf("%v produced unexpected difference when decompressing (%v)", test.fmt.Name, diff)
			if outPNG, err := out.Convert(image.PNG); err == nil {
				ioutil.WriteFile(outputPath, outPNG.Bytes, 0666)
			} else {
				t.Errorf("Could not write output file: %v", err)
			}
			for i := range out.Bytes {
				g, e := int(out.Bytes[i]), int(ref.Bytes[i])
				if g != e {
					out.Bytes[i] = 255 // Highlight errors
				}
			}
			if outPNG, err := out.Convert(image.PNG); err == nil {
				ioutil.WriteFile(errorPath, outPNG.Bytes, 0666)
			} else {
				t.Errorf("Could not write error file: %v", err)
			}
		} else {
			os.Remove(outputPath)
			os.Remove(errorPath)
		}
	}
}

func s16ToU8(src []byte, w, h, d int) ([]byte, error) {
	pixels := w * h * d
	channels := len(src) / (pixels * 2)
	out := make([]byte, 0, pixels*4)
	for i := 0; i < pixels; i++ {
		pixel := [4]byte{0, 0, 0, 255}
		for c := 0; c < channels; c++ {
			s16 := int(int16((uint16(src[1]) << 8) | uint16(src[0])))
			pixel[c] = sint.Byte(s16 >> 7)
			src = src[2:]
		}
		out = append(out, pixel[0], pixel[1], pixel[2], pixel[3])
	}
	return out, nil
}
