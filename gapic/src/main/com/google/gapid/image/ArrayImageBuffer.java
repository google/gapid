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

import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.image.Image.ImageBuffer;
import com.google.gapid.image.Image.PixelInfo;
import com.google.gapid.image.Image.PixelValue;
import com.google.gapid.util.Colors;

import org.eclipse.swt.graphics.ImageData;
import org.lwjgl.BufferUtils;
import org.lwjgl.opengl.GL11;
import org.lwjgl.opengl.GL30;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;

public abstract class ArrayImageBuffer implements ImageBuffer {
  public final int width, height;
  private final byte[] data;
  private final int internalFormat, format, type;

  public ArrayImageBuffer(int width, int height, byte[] data,
      int internalFormat, int format, int type) {
    this.width = width;
    this.height = height;
    this.data = data;
    this.internalFormat = internalFormat;
    this.format = format;
    this.type = type;
  }

  @Override
  public void uploadToTexture(Texture texture) {
    ByteBuffer buffer = (ByteBuffer)BufferUtils.createByteBuffer(data.length)
        .put(data)
        .flip();
    texture.loadData(width, height, internalFormat, format, type, buffer);
  }

  @Override
  public ImageData getImageData() {
    ImageData result = Images.createImageData(width, height, false);
    convert(data, result.data);
    return result;
  }

  protected abstract void convert(byte[] src, byte[] dst);

  @Override
  public PixelValue getPixel(int x, int y) {
    if (x < 0 || y < 0 || x >= width || y >= height) {
      return PixelValue.NULL_PIXEL;
    }
    return getPixel(x, y, data);
  }

  protected abstract PixelValue getPixel(int x, int y, byte[] src);

  protected static ByteBuffer buffer(byte[] data) {
    return ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
  }

  public abstract static class Builder {
    public final int width, height;
    public final byte[] data;
    private final int pixelSize;

    public Builder(int width, int height, int pixelSize) {
      this.width = width;
      this.height = height;
      this.data = new byte[pixelSize * width * height];
      this.pixelSize = pixelSize;
    }

    public Builder update(byte[] src, int x, int y, int w, int h) {
      if (x == 0 && w == width) {
        // Copying complete rows of pixels is easy.
        System.arraycopy(src, 0, data, pixelSize * y * w, pixelSize * w * h);
      } else {
        // Copy one (incomplete) row at a time.
        for (int row = 0, p = y * width, s = 0; row < h; row++, p += width, s += w * pixelSize) {
          System.arraycopy(src, s, data, pixelSize * (p + x), pixelSize * w);
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

    protected abstract ArrayImageBuffer build();
  }

  public static class RGBA8ImageBuffer extends ArrayImageBuffer {
    public RGBA8ImageBuffer(int width, int height, byte[] data) {
      super(width, height, data, GL11.GL_RGBA8, GL11.GL_RGBA, GL11.GL_UNSIGNED_BYTE);
    }

    @Override
    protected void convert(byte[] src, byte[] dst) {
      for (int row = 0, d = 0, p = 4 * (height - 1) * width; row < height; row++, p -= 4 * width) {
        for (int col = 0, s = p; col < width; col++, s += 4, d += 3) {
          dst[d + 0] = src[s + 0];
          dst[d + 1] = src[s + 1];
          dst[d + 2] = src[s + 2];
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
    public PixelInfo getInfo() {
      return PixelInfo.NULL_INFO;
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

  public static class RGBAFloatImageBuffer extends ArrayImageBuffer {
    private final FloatBuffer buffer;
    private final PixelInfo info;

    public RGBAFloatImageBuffer(int width, int height, byte[] data) {
      super(width, height, data, GL30.GL_RGBA32F, GL11.GL_RGBA, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = FloatPixelInfo.compute(buffer);
    }

    @Override
    protected void convert(byte[] src, byte[] dst) {
      for (int row = 0, d = 0, p = 4 * (height - 1) * width; row < height; row++, p -= 4 * width) {
        for (int col = 0, s = p; col < width; col++, s += 4, d += 3) {
          dst[d + 0] = clamp(buffer.get(s + 0));
          dst[d + 1] = clamp(buffer.get(s + 1));
          dst[d + 2] = clamp(buffer.get(s + 2));
        }
      }
    }

    @Override
    protected PixelValue getPixel(int x, int y, byte[] data) {
      int i = 4 * (y * width + x);
      return new Pixel(buffer.get(i + 0), buffer.get(i + 1), buffer.get(i + 2), buffer.get(i + 3));
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

  // TODO: The client may not actually need to distinguish between luminance and RGBA
  public static class Luminance8ImageBuffer extends ArrayImageBuffer {
    public Luminance8ImageBuffer(int width, int height, byte[] data) {
      super(width, height, data, GL11.GL_RGB8, GL11.GL_RED, GL11.GL_UNSIGNED_BYTE);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    protected void convert(byte[] src, byte[] dst) {
      for (int row = 0, d = 0, p = (height - 1) * width; row < height; row++, p -= width) {
        for (int col = 0, s = p; col < width; col++, s ++, d += 3) {
          dst[d + 0] = src[s];
          dst[d + 1] = src[s];
          dst[d + 2] = src[s];
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

  //TODO: The client may not actually need to distinguish between luminance and RGBA
  public static class LuminanceFloatImageBuffer extends ArrayImageBuffer {
    private final FloatBuffer buffer;
    private final PixelInfo info;

    public LuminanceFloatImageBuffer(int width, int height, byte[] data) {
      super(width, height, data, GL30.GL_RGB32F, GL11.GL_RED, GL11.GL_FLOAT);
      this.buffer = buffer(data).asFloatBuffer();
      this.info = FloatPixelInfo.compute(buffer);
    }

    @Override
    public void uploadToTexture(Texture texture) {
      super.uploadToTexture(texture);
      texture.setSwizzle(GL11.GL_RED, GL11.GL_RED, GL11.GL_RED, GL11.GL_ONE);
    }

    @Override
    protected void convert(byte[] src, byte[] dst) {
      for (int row = 0, d = 0, p = (height - 1) * width; row < height; row++, p -= width) {
        for (int col = 0, s = p; col < width; col++, s ++, d += 3) {
          byte value = clamp(buffer.get(s));
          dst[d + 0] = value;
          dst[d + 1] = value;
          dst[d + 2] = value;
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

    private FloatPixelInfo(float min, float max) {
      this.min = min;
      this.max = max;
    }

    public static PixelInfo compute(FloatBuffer buffer) {
      if (!buffer.hasRemaining()) {
        return PixelInfo.NULL_INFO;
      }

      float min = Float.POSITIVE_INFINITY, max = Float.NEGATIVE_INFINITY;
      for (int i = 0; i < buffer.remaining(); i++) {
        float value = buffer.get(i);
        min = Math.min(min, value);
        max = Math.max(max, value);
      }
      return new FloatPixelInfo(min, max);
    }

    @Override
    public float getMin() {
      return min;
    }

    @Override
    public float getMax() {
      return max;
    }
  }
}
