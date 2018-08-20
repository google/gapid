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

import static com.google.gapid.util.Buffers.nativeBuffer;
import static com.google.gapid.util.Caches.getUnchecked;
import static com.google.gapid.util.Caches.softCache;
import static com.google.gapid.util.Colors.DARK_LUMINANCE8_THRESHOLD;
import static com.google.gapid.util.Colors.DARK_LUMINANCE_THRESHOLD;
import static com.google.gapid.util.Colors.clamp;

import com.google.common.cache.Cache;
import com.google.common.primitives.UnsignedBytes;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.image.Histogram.Binner;
import com.google.gapid.proto.stream.Stream;
import com.google.gapid.util.Colors;

import org.eclipse.swt.graphics.ImageData;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL30;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;
import java.util.Arrays;
import java.util.Set;

/**
 * An {@link Image} backed by a byte array.
 */
public abstract class ArrayImage implements com.google.gapid.image.Image {
  protected static final Cache<Image.Key, PixelInfo> PIXEL_INFO_CACHE = softCache();

  public final int width, height, depth, bytesPerPixel;
  protected final byte[] data;
  private final int internalFormat, format, type;

  public ArrayImage(int width, int height, int depth, int bytesPerPixel, byte[] data,
      int internalFormat, int format, int type) {
    this.width = width;
    this.height = height;
    this.depth = depth;
    this.bytesPerPixel = bytesPerPixel;
    this.data = data;
    this.internalFormat = internalFormat;
    this.format = format;
    this.type = type;
  }

  @Override
  public int getWidth() {
    return width;
  }

  @Override
  public int getHeight() {
    return height;
  }

  @Override
  public int getDepth() {
    return depth;
  }

  @Override
  public Image getSlice(int z) {
    int sliceSize = width * height * bytesPerPixel;
    byte[] sliceData = Arrays.copyOfRange(data, sliceSize * z, sliceSize * (z + 1));
    return create(width, height, 1, sliceData);
  }

  /**
   * Constructs and returns a new {@link Image} of the same format with the given
   * dimensions and data.
   */
  protected abstract Image create(int w, int h, int d, byte[] pixels);

  @Override
  public void uploadToTexture(Texture texture) {
    texture.loadData(width, height, internalFormat, format, type, nativeBuffer(data));
  }

  @Override
  public ImageData getImageData() {
    ImageData result = Images.createImageData(width, height, true);
    convert2D(data, result.data, result.alphaData, result.bytesPerLine);
    return result;
  }

  protected abstract void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride);

  @Override
  public PixelValue getPixel(int x, int y, int z) {
    if (x < 0 || y < 0 || z < 0 || x >= width || y >= height || z > depth) {
      return PixelValue.NULL_PIXEL;
    }
    return getPixel(x, y);
  }

  protected abstract PixelValue getPixel(int x, int y);

  protected static ByteBuffer buffer(byte[] data) {
    return ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
  }

  /**
   * An {@link ArrayImage} builder.
   */
  public abstract static class Builder {
    public final int width, height, depth;
    public final byte[] data;
    private final int pixelSize;

    public Builder(int width, int height, int depth, int pixelSize) {
      this.width = width;
      this.height = height;
      this.depth = depth;
      this.data = new byte[pixelSize * width * height * depth];
      this.pixelSize = pixelSize;
    }

    public Builder update(byte[] src, int x, int y, int z, int w, int h, int d) {
      if (x == 0 && y == 0 && w == width && h == height) {
        // Simple case. Bulk copy.
        System.arraycopy(src, 0, data, pixelSize * w * h * z, pixelSize * w * h * d);
        return this;
      }

      for (int slice = 0; slice < d; slice++) {
        int dstOffset = pixelSize * slice * width * height;
        int srcOffset = pixelSize * slice * w * h;
        if (x == 0 && w == width) {
          // Copying complete rows of pixels is easy.
          System.arraycopy(src, srcOffset, data, dstOffset + pixelSize * y * w, pixelSize * w * h);
        } else {
          // Copy one (incomplete) row at a time.
          for (int row = 0, p = y * width, s = 0; row < h; row++, p += width, s += w * pixelSize) {
            System.arraycopy(src, srcOffset + s, data, dstOffset + pixelSize * (p + x), pixelSize * w);
          }
        }
      }
      return this;
    }

    public Builder flip() {
      int s = pixelSize * width;
      byte[] row = new byte[s];
      for (int y = 0, i = 0, j = data.length - s; y < height / 2; y++, i += s, j -= s) {
        System.arraycopy(data, i, row, 0, s);
        System.arraycopy(data, j, data, i, s);
        System.arraycopy(row, 0, data, j, s);
      }
      return this;
    }

    protected abstract ArrayImage build();
  }

  /**
   * An {@link ArrayImage} that represents an RGBA image with 8bit color channels.
   */
  public static class RGBA8Image extends ArrayImage {
    private final PixelInfo info;

    public RGBA8Image(Image.Key key, int width, int height, int depth, byte[] data) {
      this(width, height, depth, data,
          getUnchecked(PIXEL_INFO_CACHE, key, () -> IntPixelInfo.compute(data, true)));
    }

    private RGBA8Image(int width, int height, int depth, byte[] data, PixelInfo info) {
      super(width, height, depth, 4, data, GL11.GL_RGBA8, GL11.GL_RGBA, GL11.GL_UNSIGNED_BYTE);
      this.info = info;
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new RGBA8Image(w, h, d, pixels, info);
    }

    @Override
    protected void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride) {
      for (int row = 0, di = 0, si = 4 * (height - 1) * width, ai = 0; row < height;
          row++, si -= 4 * width, di += stride) {
        for (int col = 0, s = si, d = di; col < width; col++, s += 4, d += 3, ai++) {
          dst[d + 0] = src[s + 0];
          dst[d + 1] = src[s + 1];
          dst[d + 2] = src[s + 2];
          alpha[ai] = src[s + 3];
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y) {
      int i = 4 * (y * width + x);
      return new Pixel(
          ((data[i + 3] & 0xFF) << 24) |
          ((data[i + 0] & 0xFF) << 16) |
          ((data[i + 1] & 0xFF) << 8) |
          ((data[i + 2] & 0xFF) << 0));
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Images.RGB_CHANNELS;
    }

    @Override
    public Image.ImageType getType() {
      return Image.ImageType.LDR;
    }

    @Override
    public void bin(Binner binner) {
      for (int i = 0, end = data.length - 3; i < end; ) {
        binner.bin(UnsignedBytes.toInt(data[i++]) / 255f, Stream.Channel.Red);
        binner.bin(UnsignedBytes.toInt(data[i++]) / 255f, Stream.Channel.Green);
        binner.bin(UnsignedBytes.toInt(data[i++]) / 255f, Stream.Channel.Blue);
        i++; // skip alpha
      }
    }

    @Override
    public PixelInfo getInfo() {
      return info;
    }

    private static class Pixel implements PixelValue {
      private final int argb;

      public Pixel(int argb) {
        this.argb = argb;
      }

      @Override
      public String toString() {
        return String.format("ARGB: %08x", argb);
      }

      @Override
      public boolean isDark() {
        return Colors.getLuminance(argb) < DARK_LUMINANCE_THRESHOLD;
      }
    }
  }

  /**
   * An {@link ArrayImage} that represents an RGBA image with 32bit float color channels.
   */
  public static class RGBAFloatImage extends ArrayImage {
    private final FloatBuffer buffer;
    private final PixelInfo info;

    public RGBAFloatImage(Image.Key key, int width, int height, int depth, byte[] data) {
      super(width, height, depth, 16, data, GL30.GL_RGBA32F, GL11.GL_RGBA, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = getUnchecked(PIXEL_INFO_CACHE, key, () -> FloatPixelInfo.compute(buffer, true));
    }

    private RGBAFloatImage(int width, int height, int depth, byte[] data, PixelInfo info) {
      super(width, height, depth, 16, data, GL30.GL_RGBA32F, GL11.GL_RGBA, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = info;
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new RGBAFloatImage(w, h, d, pixels, info);
    }

    @Override
    protected void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride) {
      for (int row = 0, di = 0, si = 4 * (height - 1) * width, ai = 0; row < height;
          row++, si -= 4 * width, di += stride) {
        for (int col = 0, s = si, d = di; col < width; col++, s += 4, d += 3, ai++) {
          dst[d + 0] = clamp(buffer.get(s + 0));
          dst[d + 1] = clamp(buffer.get(s + 1));
          dst[d + 2] = clamp(buffer.get(s + 2));
          alpha[ai] = clamp(buffer.get(s + 3));
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y) {
      int i = 4 * (y * width + x);
      return new Pixel(buffer.get(i + 0), buffer.get(i + 1), buffer.get(i + 2), buffer.get(i + 3));
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Images.RGB_CHANNELS;
    }

    @Override
    public Image.ImageType getType() {
      return Image.ImageType.HDR;
    }

    @Override
    public void bin(Histogram.Binner binner) {
      for (int i = 0, end = buffer.remaining() - 3; i <= end; ) {
        float value = buffer.get(i++);
        if (!Float.isNaN(value) && !Float.isInfinite(value)) {
          binner.bin(value, Stream.Channel.Red);
        }
        value = buffer.get(i++);
        if (!Float.isNaN(value) && !Float.isInfinite(value)) {
          binner.bin(value, Stream.Channel.Green);
        }
        value = buffer.get(i++);
        if (!Float.isNaN(value) && !Float.isInfinite(value)) {
          binner.bin(value, Stream.Channel.Blue);
        }
        i++; // Skip alpha.
      }
    }

    @Override
    public PixelInfo getInfo() {
      return info;
    }

    private static class Pixel implements PixelValue {
      private final float r, g, b, a;

      public Pixel(float r, float g, float b, float a) {
        this.r = r;
        this.g = g;
        this.b = b;
        this.a = a;
      }

      @Override
      public String toString() {
        return String.format("ARGB(%f, %f, %f, %f)", a, r, g, b);
      }

      @Override
      public boolean isDark() {
        return Colors.getLuminance(r, g, b) < DARK_LUMINANCE_THRESHOLD;
      }
    }
  }

  /**
   * An {@link ArrayImage} that represents an 8bit luminance image.
   */
  // TODO: The client may not actually need to distinguish between luminance and RGBA
  public static class Luminance8Image extends ArrayImage {
    private final PixelInfo info;

    public Luminance8Image(Image.Key key, int width, int height, int depth, byte[] data) {
      this(width, height, depth, data,
          getUnchecked(PIXEL_INFO_CACHE, key, () -> IntPixelInfo.compute(data, false)));
    }

    private Luminance8Image(int width, int height, int depth, byte[] data, PixelInfo info) {
      super(width, height, depth, 1, data, GL11.GL_RGB8, GL11.GL_RED, GL11.GL_UNSIGNED_BYTE);
      this.info = info;
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new Luminance8Image(w, h, d, pixels, info);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Images.LUMINANCE_CHANNELS;
    }

    @Override
    public Image.ImageType getType() {
      return Image.ImageType.LDR;
    }

    @Override
    public void bin(Binner binner) {
      for (int i = 0; i < data.length; i++) {
        binner.bin(UnsignedBytes.toInt(data[i]) / 255.0f, Stream.Channel.Luminance);
      }
    }

    @Override
    protected void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride) {
      for (int row = 0, di = 0, si = (height - 1) * width, ai = 0; row < height;
          row++, si -= width, di += stride) {
        for (int col = 0, s = si, d = di; col < width; col++, s++, d += 3, ai++) {
          dst[d + 0] = src[s];
          dst[d + 1] = src[s];
          dst[d + 2] = src[s];
          alpha[ai] = -1;
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y) {
      return new Pixel(data[y * width + x]);
    }

    @Override
    public PixelInfo getInfo() {
      return info;
    }

    private static class Pixel implements PixelValue {
      private final int luminance;

      public Pixel(byte luminance) {
        this.luminance = luminance & 0xFF;
      }

      @Override
      public String toString() {
        return String.format("Y = %02x", luminance);
      }

      @Override
      public boolean isDark() {
        return luminance < DARK_LUMINANCE8_THRESHOLD;
      }
    }
  }

  /**
   * An {@link ArrayImage} that represents a 32bit float luminance image.
   */
  //TODO: The client may not actually need to distinguish between luminance and RGBA
  public static class LuminanceFloatImage extends ArrayImage {
    private final FloatBuffer buffer;
    private final PixelInfo info;

    public LuminanceFloatImage(Image.Key key, int width, int height, int depth, byte[] data) {
      super(width, height, depth, 4, data, GL30.GL_RGB32F, GL11.GL_RED, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = getUnchecked(PIXEL_INFO_CACHE, key, () -> FloatPixelInfo.compute(buffer, false));
    }

    private LuminanceFloatImage(int width, int height, int depth, byte[] data, PixelInfo info) {
      super(width, height, depth, 4, data, GL30.GL_RGB32F, GL11.GL_RED, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = info;
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new LuminanceFloatImage(w, h, d, pixels, info);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Images.LUMINANCE_CHANNELS;
    }

    @Override
    public Image.ImageType getType() {
      return Image.ImageType.HDR;
    }

    @Override
    public void bin(Binner binner) {
      for (int i = 0; i < buffer.remaining(); i++) {
        float value = buffer.get(i);
        if (!Float.isNaN(value) && !Float.isInfinite(value)) {
          binner.bin(value, Stream.Channel.Luminance);
        }
      }
    }

    @Override
    protected void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride) {
      for (int row = 0, di = 0, si = (height - 1) * width, ai = 0; row < height;
          row++, si -= width, di += stride) {
        for (int col = 0, s = si, d = di; col < width; col++, s++, d += 3, ai++) {
          byte value = clamp(buffer.get(s));
          dst[d + 0] = value;
          dst[d + 1] = value;
          dst[d + 2] = value;
          alpha[ai] = -1;
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y) {
      return new Pixel(buffer.get(y * width + x));
    }

    @Override
    public PixelInfo getInfo() {
      return info;
    }

    private static class Pixel implements PixelValue {
      private final float luminance;

      public Pixel(float luminance) {
        this.luminance = luminance;
      }

      @Override
      public String toString() {
        return "Y = " + luminance;
      }

      @Override
      public boolean isDark() {
        return luminance < DARK_LUMINANCE_THRESHOLD;
      }
    }
  }

  /**
   * An {@link ArrayImage} that represents an 8bit count image.
   */
  public static class Count8Image extends ArrayImage {
    private final PixelInfo info;

    public Count8Image(Image.Key key, int width, int height, int depth, byte[] data) {
      this(width, height, depth, data,
          getUnchecked(PIXEL_INFO_CACHE, key, () -> IntPixelInfo.compute(data, false)));
    }

    private Count8Image(int width, int height, int depth, byte[] data, PixelInfo info) {
      super(width, height, depth, 1, data, GL11.GL_RGB8, GL11.GL_RED, GL11.GL_UNSIGNED_BYTE);
      this.info = info;
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new Count8Image(w, h, d, pixels, getInfo());
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Images.COUNT_CHANNELS;
    }
    @Override
    public Image.ImageType getType() {
      return Image.ImageType.COUNT;
    }

    @Override
    public void bin(Binner binner) {
      for (int i = 0; i < data.length; i++) {
        binner.bin(UnsignedBytes.toInt(data[i]) / 255.0f, Stream.Channel.Count);
      }
    }

    @Override
    protected void convert2D(byte[] src, byte[] dst, byte[] alpha, int stride) {
      for (int row = 0, di = 0, si = (height - 1) * width, ai = 0; row < height;
          row++, si -= width, di += stride) {
        for (int col = 0, s = si, d = di; col < width; col++, s++, d += 3, ai++) {
          dst[d + 0] = src[s];
          dst[d + 1] = src[s];
          dst[d + 2] = src[s];
          alpha[ai] = -1;
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y) {
      return new Pixel(data[y * width + x]);
    }

    @Override
    public PixelInfo getInfo() {
      return info;
    }

    private static class Pixel implements PixelValue {
      private final int count;

      public Pixel(byte count) {
        this.count = UnsignedBytes.toInt(count);
      }

      @Override
      public String toString() {
        return "Count = " + count;
      }

      @Override
      public boolean isDark() {
        return count < DARK_LUMINANCE8_THRESHOLD;
      }
    }
  }

  private static class FloatPixelInfo implements PixelInfo {
    private final double min, max, average;
    private final double alphaMin, alphaMax;

    private FloatPixelInfo(
        double min, double max, double average, double alphaMin, double alphaMax) {
      this.min = min;
      this.max = max;
      this.average = average;
      this.alphaMin = alphaMin;
      this.alphaMax = alphaMax;
    }

    public static PixelInfo compute(FloatBuffer buffer, boolean isRGBA) {
      if (!buffer.hasRemaining()) {
        return PixelInfo.NULL_INFO;
      }

      double min = Double.POSITIVE_INFINITY, max = Double.NEGATIVE_INFINITY, alphaMin, alphaMax;
      double average = 0;
      long count = 0;
      if (isRGBA) {
        alphaMin = Float.POSITIVE_INFINITY;
        alphaMax = Float.NEGATIVE_INFINITY;
        for (int i = 0, end = buffer.remaining() - 3; i <= end; ) {
          float value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
            average += value;
            count++;
          }
          value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
            average += value;
            count++;
          }
          value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
            average += value;
            count++;
          }
          value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            alphaMin = Math.min(alphaMin, value);
            alphaMax = Math.max(alphaMax, value);
          }
        }
      } else {
        alphaMin = alphaMax = 1;
        for (int i = 0; i < buffer.remaining(); i++) {
          float value = buffer.get(i);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
            average += value;
            count++;
          }
        }
      }
      return new FloatPixelInfo(
          min, max, (count == 0) ? 0.5 : (average / count), alphaMin, alphaMax);
    }

    @Override
    public double getMin() {
      return min;
    }

    @Override
    public double getMax() {
      return max;
    }

    @Override
    public double getAverage() {
      return average;
    }

    @Override
    public double getAlphaMin() {
      return alphaMin;
    }

    @Override
    public double getAlphaMax() {
      return alphaMax;
    }
  }

  private static class IntPixelInfo implements PixelInfo {
    private final double min, max, average;
    private final double alphaMin, alphaMax;

    private IntPixelInfo(double min, double max, double average, double alphaMin, double alphaMax) {
      this.min = min;
      this.max = max;
      this.average = average;
      this.alphaMin = alphaMin;
      this.alphaMax = alphaMax;
    }

    public static PixelInfo compute(byte[] data, boolean isRGBA) {
      if (data.length == 0) {
        return PixelInfo.NULL_INFO;
      }

      int min = 255, max = 0, alphaMin, alphaMax;
      double average = 0;
      if (isRGBA) {
        alphaMin = 255; alphaMax = 0;
        for (int i = 0, end = data.length - 3; i < end; ) {
          int value = UnsignedBytes.toInt(data[i++]);
          min = Math.min(min, value);
          max = Math.max(max, value);
          average += value;

          value = UnsignedBytes.toInt(data[i++]);
          min = Math.min(min, value);
          max = Math.max(max, value);
          average += value;

          value = UnsignedBytes.toInt(data[i++]);
          min = Math.min(min, value);
          max = Math.max(max, value);
          average += value;

          value = UnsignedBytes.toInt(data[i++]);
          alphaMin = Math.min(alphaMin, value);
          alphaMax = Math.max(alphaMax, value);
        }
        average /= ((data.length / 4) * 3); // Truncate-divide first on purpose.
      } else {
        alphaMin = alphaMax = 255;
        for (int i = 0; i < data.length; i++) {
          int value = UnsignedBytes.toInt(data[i]);
          min = Math.min(min, value);
          max = Math.max(max, value);
          average += value;
        }
        average /= data.length;
      }
      return new IntPixelInfo(
          min / 255.0, max / 255.0, average / 255.0, alphaMin / 255.0, alphaMax / 255.0);
    }

    @Override
    public double getMin() {
      return min;
    }

    @Override
    public double getMax() {
      return max;
    }

    @Override
    public double getAverage() {
      return average;
    }

    @Override
    public double getAlphaMin() {
      return alphaMin;
    }

    @Override
    public double getAlphaMax() {
      return alphaMax;
    }
  }
}
