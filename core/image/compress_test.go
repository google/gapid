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
	"github.com/google/gapid/core/image/etc"
)

type testImageInfo struct {
	compressed   *image.Format
	ext          string
	uncompressed *image.Format
}

func TestCompressors(t *testing.T) {
	// TODO(melihyalcin): We don't have a converter for'png' to 'RG_S16_NORM' or 'R_S16_NORM'
	// If we need this conversion, we can add the converter then we can read
	// the png and test it

	// TODO(melihyalcin): Conversion quality for ETC2_R_U11_NORM is low (~0.20) we don't need
	// to test that quality for our use case. If we need, we have to look at how to improve it.
	testImageInfos := []testImageInfo{
		{astc.RGBA_4x4, ".astc", image.RGBA_U8_NORM},
		{etc.ETC2_RGBA_U8U8U8U1_NORM, ".ktx", image.RGBA_U8_NORM},
		{etc.ETC2_RGBA_U8_NORM, ".ktx", image.RGBA_U8_NORM},
		{etc.ETC2_RGB_U8_NORM, ".ktx", image.RGB_U8_NORM},
		// {etc.ETC2_RG_S11_NORM, ".ktx", image.RG_S16_NORM},
		// {etc.ETC2_RG_U11_NORM, ".ktx", image.RG_U16_NORM},
		// {etc.ETC2_R_S11_NORM, ".ktx", image.R_S16_NORM},
		// {etc.ETC2_R_U11_NORM, ".ktx", image.R_U16_NORM},
	}

	for _, testImage := range testImageInfos {
		uncompressedImg, err := getUncompressedImage(testImage)
		if err != nil {
			t.Error(err)
			continue
		}

		compressedImage, err := uncompressedImg.Convert(testImage.compressed)
		if err != nil {
			t.Errorf("Failed to convert '%s' from %v to %v: %v", testImage.compressed.Name, uncompressedImg.Format.Name, testImage.compressed.Name, err)
			continue
		}

		refImg, err := getRefImage(testImage)
		if err != nil {
			t.Error(err)
			continue
		}

		errs := outputDifference(testImage.compressed.Name, refImg, compressedImage)
		if errs != nil && len(errs) > 0 {
			for _, err := range errs {
				t.Error(err)
			}
			continue
		}
	}
}

func getRefImage(testImage testImageInfo) (*image.Data, error) {
	imagePath := filepath.Join("test_data", testImage.compressed.Name+testImage.ext)
	imageData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read  '%s': %v", imagePath, err)
	}

	switch testImage.ext {
	case ".astc":
		imageASTC, err := image.ASTCFrom(imageData)
		if err != nil {
			return nil, fmt.Errorf("Failed to read '%s': %v", testImage.compressed.Name, err)
		}

		if imageASTC.Format.Key() != testImage.compressed.Key() {
			return nil, fmt.Errorf("%v was not the expected format. Expected %v, got %v",
				imagePath, testImage.compressed.Name, imageASTC.Format.Name)
		}

		return imageASTC, nil
	case ".ktx":
		imageKTX, err := loadKTX(imageData)
		if err != nil {
			return nil, fmt.Errorf("Failed to read '%s': %v", imagePath, err)
		}

		if imageKTX.Format.Key() != testImage.compressed.Key() {
			return nil, fmt.Errorf("%v was not the expected format. Expected %v, got %v",
				imagePath, testImage.compressed.Name, imageKTX.Format.Name)
		}

		return imageKTX, nil
	default:
		return nil, fmt.Errorf("This should not happen: Unknown Format")
	}
}

func getUncompressedImage(testImage testImageInfo) (*image.Data, error) {
	imagePath := filepath.Join("test_data", testImage.compressed.Name+".png")

	imagePNGData, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read  '%s': %v", imagePath, err)
	}
	imagePNG, err := image.PNGFrom(imagePNGData)
	if err != nil {
		return nil, fmt.Errorf("Failed to read PNG '%s': %v", imagePath, err)
	}

	img, err := imagePNG.Convert(testImage.uncompressed)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert '%s' from PNG to %v: %v", imagePath, testImage.uncompressed.Name, err)
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

	errorMargin := float32(0.01)
	if diff <= errorMargin {
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
