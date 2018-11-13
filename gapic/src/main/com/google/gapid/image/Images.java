/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.gapid.image;

import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.stream.Stream;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Streams;

import org.eclipse.jface.resource.ImageDescriptor;
import org.eclipse.jface.resource.ResourceManager;
import org.eclipse.swt.graphics.Device;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.PaletteData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.internal.DPIUtil;

import java.util.Set;

/**
 * Utilities to deal with images.
 */
public class Images {
  public static final Image.Format FMT_RGBA_U8_NORM = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_RGBA_U8_NORM))
      .build();
  public static final Image.Format FMT_DEPTH_U8_NORM = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_DEPTH_U8_NORM))
      .build();
  public static final Image.Format FMT_RGBA_FLOAT = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_RGBA_FLOAT))
      .build();
  public static final Image.Format FMT_LUMINANCE_FLOAT = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_LUMINANCE_FLOAT))
      .build();
  public static final Image.Format FMT_DEPTH_FLOAT = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_DEPTH_FLOAT))
      .build();
  public static final Image.Format FMT_COUNT_U8 = Image.Format.newBuilder()
      .setUncompressed(Image.FmtUncompressed.newBuilder().setFormat(Streams.FMT_COUNT_U8))
      .build();

  public static final Set<Stream.Channel> COLOR_CHANNELS = Sets.immutableEnumSet(
      Stream.Channel.Red, Stream.Channel.Green, Stream.Channel.Blue, Stream.Channel.Alpha,
      Stream.Channel.Luminance, Stream.Channel.ChromaU, Stream.Channel.ChromaV);

  public static final Set<Stream.Channel> DEPTH_CHANNELS = Sets.immutableEnumSet(
      Stream.Channel.Depth);

  public static final Set<Stream.Channel> RGB_CHANNELS = Sets.immutableEnumSet(
      Stream.Channel.Red, Stream.Channel.Green, Stream.Channel.Blue);

  public static final Set<Stream.Channel> LUMINANCE_CHANNELS = Sets.immutableEnumSet(
      Stream.Channel.Luminance);

  public static final Set<Stream.Channel> COUNT_CHANNELS = Sets.immutableEnumSet(
      Stream.Channel.Count);

  private Images() {
  }

  public static ImageData createImageData(int width, int height, boolean hasAlpha) {
    ImageData result =
        new ImageData(width, height, 24, new PaletteData(0xFF0000, 0x00FF00, 0x0000FF));
    result.alphaData = hasAlpha ? new byte[width * height] : null;
    return result;
  }

  /**
   * Auto scales the given image to the devices DPI setting.
   */
  public static org.eclipse.swt.graphics.Image createAutoScaledImage(
      Device device, ImageData data) {
    return new org.eclipse.swt.graphics.Image(device, data);
  }

  /**
   * Auto scales the given image to the devices DPI setting.
   */
  public static org.eclipse.swt.graphics.Image createAutoScaledImage(
      ResourceManager resources, ImageData data) {
    return resources.createImage(
        ImageDescriptor.createFromImageDataProvider(zoom -> (zoom == 100) ? data : null));
  }

  public static org.eclipse.swt.graphics.Image createNonScaledImage(Device device, ImageData data) {
    return new org.eclipse.swt.graphics.Image(device,
        new DPIUtil.AutoScaleImageDataProvider(device, data, DPIUtil.getDeviceZoom()));
  }

  public static org.eclipse.swt.graphics.Image createNonScaledImage(
      ResourceManager resources, ImageData data) {
    return resources.createImage(new ImageDescriptor() {
      @Override
      public org.eclipse.swt.graphics.Image createImage(boolean ignored, Device device) {
        return createNonScaledImage(device, data);
      }

      @Override
      public ImageData getImageData() {
        throw new AssertionError();
      }
    });
  }

  public static Point getSize(org.eclipse.swt.graphics.Image image) {
    Rectangle bounds = image.getBounds();
    return new Point(bounds.width, bounds.height);
  }

  public static ListenableFuture<ImageData> noAlpha(ListenableFuture<ImageData> image) {
    return MoreFutures.transform(image, data -> {
      data.alphaData = null;
      return data;
    });
  }

  public static Image.Format getFormatToRequest(Image.Format format) {
      if (isColorFormat(format)) {
        return are8BitsEnough(format, COLOR_CHANNELS) ? FMT_RGBA_U8_NORM :
            (getChannelCount(format, COLOR_CHANNELS) == 1 ? FMT_LUMINANCE_FLOAT : FMT_RGBA_FLOAT);
      } else if (isCountFormat(format)) {
        // Currently only one count format
        return FMT_COUNT_U8;
      } else {
        return are8BitsEnough(format, DEPTH_CHANNELS) ? FMT_DEPTH_U8_NORM : FMT_DEPTH_FLOAT;
      }
  }

  public static int getChannelCount(Image.Format format, Set<Stream.Channel> interestedChannels) {
    switch (format.getFormatCase()) {
      case UNCOMPRESSED:
        return getChannelCount(format.getUncompressed().getFormat(), interestedChannels);
      case ETC2_R_U11_NORM:
      case ETC2_R_S11_NORM:
        return 1;
      case ETC2_RG_U11_NORM:
      case ETC2_RG_S11_NORM:
        return 2;
      case ATC_RGB_AMD:
      case ETC1_RGB_U8_NORM:
      case ETC2_RGB_U8_NORM:
      case S3_DXT1_RGB:
        return 3;
      case ASTC:
      case ATC_RGBA_EXPLICIT_ALPHA_AMD:
      case ATC_RGBA_INTERPOLATED_ALPHA_AMD:
      case ETC2_RGBA_U8_NORM:
      case ETC2_RGBA_U8U8U8U1_NORM:
      case PNG:
      case S3_DXT1_RGBA:
      case S3_DXT3_RGBA:
      case S3_DXT5_RGBA:
        return 4;
      default:
        return 0;
    }
  }

  public static int getChannelCount(Stream.Format format, Set<Stream.Channel> interestedChannels) {
    int result = 0;
    for (Stream.Component c : format.getComponentsList()) {
      if (interestedChannels.contains(c.getChannel())) {
        result++;
      }
    }
    return result;
  }

  public static boolean are8BitsEnough(
      Image.Format format, Set<Stream.Channel> interestedChannels) {
    switch (format.getFormatCase()) {
      case UNCOMPRESSED:
        return are8BitsEnough(format.getUncompressed().getFormat(), interestedChannels);
      default:
        // All Compressed formats can fully be represented as 8 bits (at this time).
        return true;
    }
  }

  public static boolean are8BitsEnough(
      Stream.Format format, Set<Stream.Channel> interestedChannels) {
    for (Stream.Component c : format.getComponentsList()) {
      if (interestedChannels.contains(c.getChannel()) && !are8BitsEnough(c.getDataType())) {
        return false;
      }
    }
    return true;
  }

  public static boolean are8BitsEnough(Stream.DataType type) {
    switch (type.getKindCase()) {
      case INTEGER:
        return type.getInteger().getBits() <= (type.getSigned() ? 7 : 8);
      default:
        // TODO: some small floats could actually be represented as 8bit...?
        return false;
    }
  }

  public static boolean isColorFormat(Image.Format format) {
    switch (format.getFormatCase()) {
      case UNCOMPRESSED: return isColorFormat(format.getUncompressed().getFormat());
      default: return true; // Compressed images have RGBA converters.
    }
  }

  public static boolean isColorFormat(Stream.Format format) {
    for (Stream.Component c : format.getComponentsList()) {
      if (COLOR_CHANNELS.contains(c.getChannel())) {
        return true;
      }
    }
    return false;
  }

  public static boolean isCountFormat(Image.Format format) {
    switch (format.getFormatCase()) {
      case UNCOMPRESSED: return isCountFormat(format.getUncompressed().getFormat());
      default: return false;
    }
  }

  public static boolean isCountFormat(Stream.Format format) {
    for (Stream.Component c : format.getComponentsList()) {
      if (COUNT_CHANNELS.contains(c.getChannel())) {
        return true;
      }
    }
    return false;
  }

  /**
   * Image formats handled by the UI.
   */
  public static enum Format {
    Color8(FMT_RGBA_U8_NORM, 4 * 1) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.RGBA8Image(key, width, height, depth, data);
      }
    },
    Depth8(FMT_DEPTH_U8_NORM, 1 *1) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.Luminance8Image(key, width, height, depth, data);
      }
    },
    ColorFloat(FMT_RGBA_FLOAT, 4 * 4) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.RGBAFloatImage(key, width, height, depth, data);
      }
    },
    DepthFloat(FMT_DEPTH_FLOAT, 1 * 4) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.LuminanceFloatImage(key, width, height, depth, data);
      }
    },
    LuminanceFloat(FMT_LUMINANCE_FLOAT, 1 * 4) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.LuminanceFloatImage(key, width, height, depth, data);
      }
    },
    Count8(FMT_COUNT_U8, 1 * 1) {
      @Override
      protected ArrayImage build(
          com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data) {
        return new ArrayImage.Count8Image(key, width, height, depth, data);
      }
    };

    public final Image.Format format;
    public final int pixelSize;

    private Format(Image.Format format, int pixelSize) {
      this.format = format;
      this.pixelSize = pixelSize;
    }

    public static Format from(Image.Format format) {
      if (isColorFormat(format)) {
        return are8BitsEnough(format, COLOR_CHANNELS) ? Color8 :
            (getChannelCount(format, COLOR_CHANNELS) == 1 ? LuminanceFloat : ColorFloat);
      } else if (isCountFormat(format)) {
        // Currently only one count format
        return Count8;
      } else {
        return are8BitsEnough(format, DEPTH_CHANNELS) ? Depth8 : DepthFloat;
      }
    }

    public ArrayImage.Builder builder(
        com.google.gapid.image.Image.Key key, int width, int height, int depth) {
      return new ArrayImage.Builder(width, height, depth, pixelSize) {
        @Override
        protected ArrayImage build() {
          return Format.this.build(key, width, height, depth, data);
        }
      };
    }

    protected abstract ArrayImage build(
        com.google.gapid.image.Image.Key key, int width, int height, int depth, byte[] data);
  }
}
