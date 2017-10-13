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

import static com.google.gapid.util.Colors.DARK_LUMINANCE8_THRESHOLD;
import static com.google.gapid.util.Colors.DARK_LUMINANCE_THRESHOLD;
import static com.google.gapid.util.Colors.clamp;

import com.google.common.primitives.UnsignedBytes;
import com.google.common.collect.Sets;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.proto.stream.Stream.Channel;
import com.google.gapid.util.Colors;

import java.nio.DoubleBuffer;
import java.util.Set;
import org.eclipse.swt.graphics.ImageData;
import org.lwjgl.BufferUtils;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL30;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;
import java.util.Arrays;

/**
 * An {@link Image} backed by a byte array.
 */
public abstract class ArrayImage implements Image {
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
    ByteBuffer buffer = (ByteBuffer)BufferUtils.createByteBuffer(data.length)
        .put(data)
        .flip();
    texture.loadData(width, height, internalFormat, format, type, buffer);
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
    return getPixel(x, y, data);
  }

  protected abstract PixelValue getPixel(int x, int y, byte[] src);

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

    public RGBA8Image(int width, int height, int depth, byte[] data) {
      super(width, height, depth, 4, data, GL11.GL_RGBA8, GL11.GL_RGBA, GL11.GL_UNSIGNED_BYTE);
      this.info = IntPixelInfo.compute(data);
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new RGBA8Image(w, h, d, pixels);
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
    protected PixelValue getPixel(int x, int y, byte[] data) {
      int i = 4 * (y * width + x);
      return new Pixel(
          ((data[i + 3] & 0xFF) << 24) |
          ((data[i + 0] & 0xFF) << 16) |
          ((data[i + 1] & 0xFF) << 8) |
          ((data[i + 2] & 0xFF) << 0));
    }

    @Override
    public Set<Channel> getChannels() {
      return Sets.immutableEnumSet(Channel.Red, Channel.Green, Channel.Blue, Channel.Alpha);
    }

    @Override
    public DoubleBuffer getChannel(Channel channel) {
      int count = width * height * depth;
      DoubleBuffer out = DoubleBuffer.allocate(count);
      int offset = 0;
      switch (channel) {
        case Red:
          offset = 0;
          break;
        case Green:
          offset = 1;
          break;
        case Blue:
          offset = 2;
          break;
        case Alpha:
          offset = 3;
          break;
        default:
          return out;
      }
      for (int i = 0; i < count; i++) {
        out.put((data[i * 4 + offset] & 0xFF) / 255.0);
      }
      out.rewind();
      return out;
    }

    @Override
    public boolean isHDR() {
      return false;
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

    public RGBAFloatImage(int width, int height, int depth, byte[] data) {
      super(width, height, depth, 16, data, GL30.GL_RGBA32F, GL11.GL_RGBA, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = FloatPixelInfo.compute(buffer, true);
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new RGBAFloatImage(w, h, d, pixels);
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
    protected PixelValue getPixel(int x, int y, byte[] data) {
      int i = 4 * (y * width + x);
      return new Pixel(buffer.get(i + 0), buffer.get(i + 1), buffer.get(i + 2), buffer.get(i + 3));
    }

    @Override
    public Set<Channel> getChannels() {
      return Sets.immutableEnumSet(Channel.Red, Channel.Green, Channel.Blue, Channel.Alpha);
    }

    @Override
    public DoubleBuffer getChannel(Channel channel) {
      int count = width * height * depth;
      DoubleBuffer out = DoubleBuffer.allocate(count);
      int offset = 0;
      switch (channel) {
        case Red:
          offset = 0;
          break;
        case Green:
          offset = 1;
          break;
        case Blue:
          offset = 2;
          break;
        case Alpha:
          offset = 3;
          break;
        default:
          return out;
      }
      for (int i = 0; i < count; i++) {
        out.put(buffer.get(i * 4 + offset));
      }
      out.rewind();
      return out;
    }

    @Override
    public boolean isHDR() {
      return true;
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
    public Luminance8Image(int width, int height, int depth, byte[] data) {
      super(width, height, depth, 1, data, GL11.GL_RGB8, GL11.GL_RED, GL11.GL_UNSIGNED_BYTE);
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new Luminance8Image(w, h, d, pixels);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    public Set<Channel> getChannels() {
      return Sets.immutableEnumSet(Channel.Luminance);
    }

    @Override
    public DoubleBuffer getChannel(Channel channel) {
      DoubleBuffer out = DoubleBuffer.allocate(data.length);
      if (channel == Channel.Luminance) {
        for (byte value : data) {
          out.put((value & 0xFF) / 255.0);
        }
        out.rewind();
      }
      return out;
    }

    @Override
    public boolean isHDR() {
      return false;
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
    protected PixelValue getPixel(int x, int y, byte[] src) {
      return new Pixel(src[y * width + x]);
    }

    @Override
    public PixelInfo getInfo() {
      return PixelInfo.NULL_INFO;
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

    public LuminanceFloatImage(int width, int height, int depth, byte[] data) {
      super(width, height, depth, 4, data, GL30.GL_RGB32F, GL11.GL_RED, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = FloatPixelInfo.compute(buffer, false);
    }

    @Override
    protected Image create(int w, int h, int d, byte[] pixels) {
      return new LuminanceFloatImage(w, h, d, pixels);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    public Set<Channel> getChannels() {
      return Sets.immutableEnumSet(Channel.Luminance);
    }

    @Override
    public DoubleBuffer getChannel(Channel channel) {
      int count = width * height * depth;
      DoubleBuffer out = DoubleBuffer.allocate(count);
      if (channel == Channel.Luminance) {
        for (int i = 0; i < count; i++) {
          out.put(buffer.get(i));
        }
        out.rewind();
      }
      return out;
    }

    @Override
    public boolean isHDR() {
      return true;
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
    protected PixelValue getPixel(int x, int y, byte[] data) {
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

  private static class FloatPixelInfo implements PixelInfo {
    private final float min, max;
    private final float alphaMin, alphaMax;

    private FloatPixelInfo(float min, float max, float alphaMin, float alphaMax) {
      this.min = min;
      this.max = max;
      this.alphaMin = alphaMin;
      this.alphaMax = alphaMax;
    }

    public static PixelInfo compute(FloatBuffer buffer, boolean isRGBA) {
      if (!buffer.hasRemaining()) {
        return PixelInfo.NULL_INFO;
      }

      float min = Float.POSITIVE_INFINITY, max = Float.NEGATIVE_INFINITY;
      float alphaMin, alphaMax;
      if (isRGBA) {
        alphaMin = Float.POSITIVE_INFINITY;
        alphaMax = Float.NEGATIVE_INFINITY;
        for (int i = 0, end = buffer.remaining() - 3; i <= end; ) {
          float value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
          }
          value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
          }
          value = buffer.get(i++);
          if (!Float.isNaN(value) && !Float.isInfinite(value)) {
            min = Math.min(min, value);
            max = Math.max(max, value);
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
          }
        }
      }
      return new FloatPixelInfo(min, max, alphaMin, alphaMax);
    }

    @Override
    public float getMin() {
      return min;
    }

    @Override
    public float getMax() {
      return max;
    }

    @Override
    public float getAlphaMin() {
      return alphaMin;
    }

    @Override
    public float getAlphaMax() {
      return alphaMax;
    }
  }

  private static class IntPixelInfo implements PixelInfo {
    private final float alphaMin, alphaMax;

    private IntPixelInfo(float alphaMin, float alphaMax) {
      this.alphaMin = alphaMin;
      this.alphaMax = alphaMax;
    }

    public static PixelInfo compute(byte[] rgba) {
      if (rgba.length == 0) {
        return PixelInfo.NULL_INFO;
      }

      int alphaMin = Integer.MAX_VALUE, alphaMax = Integer.MIN_VALUE;
      for (int i = 3; i < rgba.length; i += 4) {
        int value = UnsignedBytes.toInt(rgba[i]);
        alphaMin = Math.min(alphaMin, value);
        alphaMax = Math.max(alphaMax, value);
      }
      return new IntPixelInfo(alphaMin / 255f, alphaMax / 255f);
    }

    @Override
    public float getMin() {
      return 0; // Disable automatic tone-mapping.
    }

    @Override
    public float getMax() {
      return 1; // Disable automatic tone-mapping.
    }

    @Override
    public float getAlphaMin() {
      return alphaMin;
    }

    @Override
    public float getAlphaMax() {
      return alphaMax;
    }
  }
}
