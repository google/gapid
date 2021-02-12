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

#include <assert.h>
#include <stdlib.h>
#include <string.h>
#include "etc2.h"
#include "third_party/etc2comp/EtcLib/Etc/EtcImage.h"

#include <vector>

static_assert(sizeof(etc_error) >= sizeof(Etc::Image::EncodingStatus),
              "etc_error should superset of Etc::Image::EncodingStatus to "
              "protect against overflow");

namespace {
const uint32_t MIN_JOBS = 8;
const uint32_t MAX_JOBS = 1024;
const float EFFORT = 10.0f;
const Etc::ErrorMetric ERROR_METRIC = Etc::ErrorMetric::NUMERIC;
}  // namespace

Etc::Image::Format convert_etc_format(etc_format format) {
  switch (format) {
    case ETC2_RGB_U8_NORM:
      return Etc::Image::Format::RGB8;
    case ETC2_RGBA_U8_NORM:
      return Etc::Image::Format::RGBA8;
    case ETC2_RGBA_U8U8U8U1_NORM:
      return Etc::Image::Format::RGB8A1;
    case ETC2_SRGB_U8_NORM:
      return Etc::Image::Format::SRGB8;
    case ETC2_SRGBA_U8_NORM:
      return Etc::Image::Format::SRGBA8;
    case ETC2_SRGBA_U8U8U8U1_NORM:
      return Etc::Image::Format::SRGB8A1;
    case ETC2_R_U11_NORM:
      return Etc::Image::Format::R11;
    case ETC2_RG_U11_NORM:
      return Etc::Image::Format::RG11;
    case ETC2_R_S11_NORM:
      return Etc::Image::Format::SIGNED_R11;
    case ETC2_RG_S11_NORM:
      return Etc::Image::Format::SIGNED_RG11;
    case ETC1_RGB_U8_NORM:
      return Etc::Image::Format::ETC1;
    default:
      return Etc::Image::Format::UNKNOWN;
  }
}

void read_image(const uint8_t* input_image, uint32_t width, uint32_t height,
                std::vector<Etc::ColorFloatRGBA>& output) {
  const uint8_t BYTE_PER_PIXEL = 4;
  for (uint32_t h = 0; h < height; ++h) {
    const uint8_t* src = &input_image[(h * width) * BYTE_PER_PIXEL];
    for (uint32_t w = 0; w < width; ++w) {
      output.push_back(std::move(Etc::ColorFloatRGBA::ConvertFromRGBA8(
          src[0], src[1], src[2], src[3])));
      src += BYTE_PER_PIXEL;
    }
  }
}

extern "C" etc_error compress_etc(const uint8_t* input_image,
                                  uint8_t* output_image, uint32_t width,
                                  uint32_t height, etc_format format) {
  std::vector<Etc::ColorFloatRGBA> source_image;
  source_image.reserve(width * height);

  read_image(input_image, width, height, source_image);
  Etc::Image image(reinterpret_cast<float*>(source_image.data()), width, height,
                   ERROR_METRIC);
  image.m_bVerboseOutput = false;

  auto image_format = convert_etc_format(format);
  if (image_format == Etc::Image::Format::UNKNOWN) {
    return static_cast<etc_error>(
        Etc::Image::EncodingStatus::ERROR_UNKNOWN_FORMAT);
  }

  auto status =
      image.Encode(image_format, ERROR_METRIC, EFFORT, MIN_JOBS, MAX_JOBS);
  // We don't need to care about warnings as compression only used for
  // experiments. The users can act on warnings when they actually compress
  // their textures with an appropiate compression tool.
  if (status > Etc::Image::EncodingStatus::ERROR_THRESHOLD) {
    return static_cast<etc_error>(status);
  }

  memcpy(output_image, image.GetEncodingBits(), image.GetEncodingBitsBytes());
  return static_cast<etc_error>(Etc::Image::EncodingStatus::SUCCESS);
}

extern "C" char* get_etc_error_string(etc_error error_code) {
  // This function will cause a minor memory leak to be able to return all the
  // errors produced. This function will never be called in a well behaving
  // scenario. If this method is called, compression, therefore the underlying
  // operation e.g. experiments will fail and program is likely to be closed
  // soon after.

  char* error_string = (char*)calloc(512, sizeof(char));
  auto status = static_cast<Etc::Image::EncodingStatus>(error_code);
  if (status == Etc::Image::EncodingStatus::SUCCESS) {
    strcpy(error_string, "Compression Succeed");
    return error_string;
  }

  char* current = error_string;
  int written = sprintf(current, "[");
  current += written;

  if (status > Etc::Image::EncodingStatus::ERROR_THRESHOLD) {
    if (status & Etc::Image::EncodingStatus::ERROR_UNKNOWN_FORMAT) {
      written = sprintf(current, "\"Error: Unknown Image Format\"");
      current += written;
    }
    if (status & Etc::Image::EncodingStatus::ERROR_UNKNOWN_ERROR_METRIC) {
      written = sprintf(current, "\"Error: Unknown Error Metric\"");
      current += written;
    }
    if (status & Etc::Image::EncodingStatus::ERROR_ZERO_WIDTH_OR_HEIGHT) {
      written = sprintf(current, "\"Error: Image width or height is zero\"");
      current += written;
    }
  }

  if (status > Etc::Image::EncodingStatus::WARNING_THRESHOLD) {
    written = sprintf(current, "\"Warning with the Encoding Status Bits: %d\"",
                      status);
    current += written;
  }

  sprintf(current, "]");

  return error_string;
}
