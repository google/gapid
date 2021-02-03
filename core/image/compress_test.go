// Copyright (C) 2021 Google Inc.
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/image/astc"
)

type testImageInfo struct {
	fmt *image.Format
	ext string
}

func TestCompressors(t *testing.T) {
	// TODO: We may need S16toU8 converter.

	testImageInfos := []testImageInfo{
		{astc.RGBA_4x4, ".png"},
	}

	for _, testImage := range testImageInfos {
		uncompressedImg, err := getUncompressedImage(testImage.fmt.Name)
		if err != nil {
			t.Error(err)
			continue
		}

		compressedImage, err := uncompressedImg.Convert(testImage.fmt)
		if err != nil {
			t.Errorf("Failed to convert '%s' from %v to %v: %v", testImage.fmt.Name, uncompressedImg.Format.Name, testImage.fmt.Name, err)
			continue
		}

		refImg, err := getRefImage(testImage.fmt.Name)
		if err != nil {
			t.Error(err)
			continue
		}

		errs := outputDifference(testImage.fmt.Name, refImg, compressedImage)
		if errs != nil && len(errs) > 0 {
			for _, err := range errs {
				t.Error(err)
			}
			continue
		}
	}
}

func getRefImage(name string) (*image.Data, error) {
	imagePath := filepath.Join("test_data", name+".astc")

	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read  '%s': %v", imagePath, err)
	}

	imageASTC, err := image.ASTCFrom(imageData)
	if err != nil {
		return nil, fmt.Errorf("Failed to read '%s': %v", name, err)
	}

	return imageASTC, nil
}

func getUncompressedImage(name string) (*image.Data, error) {
	imagePath := filepath.Join("test_data", name+".png")

	imagePNGData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read  '%s': %v", imagePath, err)
	}
	imagePNG, err := image.PNGFrom(imagePNGData)
	if err != nil {
		return nil, fmt.Errorf("Failed to read PNG '%s': %v", imagePath, err)
	}

	img, err := imagePNG.Convert(image.RGBA_U8_NORM)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert '%s' from PNG to %v: %v", imagePath, image.RGBA_U8_NORM, err)
	}

	return img, nil
}

func outputDifference(name string, refImage *image.Data, finalImage *image.Data) []error {
	diff, err := image.Difference(finalImage, refImage)
	if err != nil {
		return []error{fmt.Errorf("Difference returned error: %v", err)}
	}

	outputPath := name + "-output.png"
	errorPath := name + "-error.png"

	astcErrorMargin := float32(0.0001)
	if diff <= astcErrorMargin {
		os.Remove(outputPath)
		os.Remove(errorPath)
		return nil
	}

	errs := make([]error, 0, 0)
	errs = append(errs, fmt.Errorf("%v produced unexpected difference due to compression (%v)", name, diff))
	if outPNG, err := finalImage.Convert(image.PNG); err == nil {
		ioutil.WriteFile(outputPath, outPNG.Bytes, 0666)
	} else {
		errs = append(errs, fmt.Errorf("Could not write output file: %v", err))
	}
	for i := range finalImage.Bytes {
		g, e := int(finalImage.Bytes[i]), int(refImage.Bytes[i])
		if g != e {
			finalImage.Bytes[i] = 255 // Highlight errors
		}
	}
	if outPNG, err := finalImage.Convert(image.PNG); err == nil {
		ioutil.WriteFile(errorPath, outPNG.Bytes, 0666)
	} else {
		errs = append(errs, fmt.Errorf("Could not write error file: %v", err))
	}

	return errs
}
