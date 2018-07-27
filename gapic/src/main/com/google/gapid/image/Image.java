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

import com.google.common.collect.Lists;
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.image.Histogram.Binner;
import com.google.gapid.proto.image.Image.ID;
import com.google.gapid.proto.stream.Stream;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.PaletteData;
import org.eclipse.swt.graphics.RGB;
import org.lwjgl.opengl.GL11;

import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.Set;

/**
 * Image pixel data of a texture, framebuffer, etc.
 */
public interface Image {
  /**
   * @return the width in pixels of this image.
   */
  public int getWidth();

  /**
   * @return the height in pixels of this image.
   */
  public int getHeight();

  /**
   * @return the depth in pixels of this image.
   */
  public int getDepth();

  /**
   * Returns the 2D slice at the specified z-depth of a 3D image.
   */
  public Image getSlice(int z);

  /**
   * Uploads this image data to the given texture.
   */
  public void uploadToTexture(Texture texture);

  /**
   * Converts this image data to a SWT {@link ImageData} object.
   */
  public ImageData getImageData();

  /**
   * @return the {@link PixelValue} at the given pixel location.
   */
  public PixelValue getPixel(int x, int y, int z);

  /**
   * @return all the channels of this image.
   */
  public Set<Stream.Channel> getChannels();

  public enum ImageType {
    LDR, // Regular image, no high-dynamic-range
    HDR, // An image with high-dynamic-range data
    COUNT, // An image representing integer count values
  }

  /**
   * @return the type of this image
   */
  public ImageType getType();

  /**
   * Bins this image's channel data with the given {@link Histogram.Binner}.
   */
  public void bin(Histogram.Binner binner);

  /**
   * @return the {@link PixelInfo} for this buffer.
   */
  public PixelInfo getInfo();

  public static final ImageData EMPTY_IMAGE =
      new ImageData(1, 1, 1, new PaletteData(new RGB(0, 0, 0)));

  public static final Image EMPTY = new Image() {
    @Override
    public int getWidth() {
      return 1;
    }

    @Override
    public int getHeight() {
      return 1;
    }

    @Override
    public int getDepth() {
      return 1;
    }

    @Override
    public Image getSlice(int z) {
      return this;
    }

    @Override
    public void uploadToTexture(Texture texture) {
      texture.loadData(0, 0, GL11.GL_RGB, GL11.GL_RGB, GL11.GL_UNSIGNED_BYTE, null);
    }

    @Override
    public ImageData getImageData() {
      return EMPTY_IMAGE;
    }

    @Override
    public PixelValue getPixel(int x, int y, int z) {
      return PixelValue.NULL_PIXEL;
    }

    @Override
    public Set<Stream.Channel> getChannels() {
      return Collections.emptySet();
    }

    @Override
    public ImageType getType() {
      return ImageType.LDR;
    }

    @Override
    public void bin(Binner binner) {
      // Do nothing.
    }

    @Override
    public PixelInfo getInfo() {
      return PixelInfo.NULL_INFO;
    }
  };

  /**
   * Information about a specific pixel in an image.
   */
  public static interface PixelValue {
    public static final PixelValue NULL_PIXEL = new PixelValue() {
      @Override
      public boolean isDark() {
        return false;
      }

      @Override
      public String toString() {
        return "[null]";
      }
    };

    /**
     * @return whether this pixel is considered to be a dark color (based on its luminance).
     */
    public boolean isDark();

    /**
     * @return a text representation of this pixel that can be displayed to the user.
     */
    @Override
    public String toString();
  }

  /**
   * Information about all the pixels in an image.
   */
  public static interface PixelInfo {
    public static final PixelInfo NULL_INFO = new PixelInfo() {
      @Override
      public double getMin() {
        return 0;
      }

      @Override
      public double getMax() {
        return 1;
      }

      @Override
      public double getAverage() {
        return 0.5f;
      }

      @Override
      public double getAlphaMin() {
        return 1;
      }

      @Override
      public double getAlphaMax() {
        return 1;
      }
    };

    /**
     * Returns the minimum value across all channels of the image data. Used for tone mapping.
     */
    public double getMin();

    /**
     * Returns the maximum value across all channels of the image data. Used for tone mapping.
     */
    public double getMax();

    /**
     * Returns the average value across all channels of the image date. Used for tone mapping.
     */
    public double getAverage();

    /**
     * @return the minimum alpha value of the image data.
     */
    public double getAlphaMin();

    /**
     * @return the maximum alpha value of the image data.
     */
    public double getAlphaMax();
  }

  /**
   * Key that can be used for caching images and image related data.
   */
  public interface Key {
    public static final Key EMPTY_KEY =
        new SingleIdKey(com.google.gapid.proto.image.Image.ID.getDefaultInstance());

    public static Key of(com.google.gapid.proto.image.Image.Info info) {
      return new SingleIdKey(info.getBytes());
    }

    public static Key of(com.google.gapid.proto.image.Image.Info[] infos) {
      com.google.gapid.proto.image.Image.ID[] ids =
          new com.google.gapid.proto.image.Image.ID[infos.length];
      for (int i = 0; i < ids.length; i++) {
        ids[i] = infos[i].getBytes();
      }
      return of(ids);
    }

    public static Key of(com.google.gapid.proto.image.Image.ID[] ids) {
      if (ids.length == 0) {
        return EMPTY_KEY;
      } else if (ids.length == 1) {
        return new SingleIdKey(ids[0]);
      }
      return new MultiIdKey(ids);
    }

    public static class Builder {
      private final List<com.google.gapid.proto.image.Image.ID> ids = Lists.newArrayList();

      public void add(com.google.gapid.proto.image.Image.Info info) {
        ids.add(info.getBytes());
      }

      public void add(com.google.gapid.proto.image.Image.Info[] infos) {
        for (com.google.gapid.proto.image.Image.Info info : infos) {
          ids.add(info.getBytes());
        }
      }

      public Key build() {
        return Key.of(ids.toArray(new com.google.gapid.proto.image.Image.ID[ids.size()]));
      }
    }
  }

  public static class SingleIdKey implements Key {
    private final com.google.gapid.proto.image.Image.ID id;

    protected SingleIdKey(com.google.gapid.proto.image.Image.ID id) {
      this.id = id;
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof SingleIdKey)) {
        return false;
      }
      return id.equals(((SingleIdKey)obj).id);
    }

    @Override
    public int hashCode() {
      return id.hashCode();
    }
  }

  public static class MultiIdKey implements Key {
    private final com.google.gapid.proto.image.Image.ID[] ids;
    private final int h;

    public MultiIdKey(ID[] ids) {
      this.ids = ids;
      this.h = Arrays.hashCode(ids);
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof MultiIdKey)) {
        return false;
      }
      return Arrays.equals(ids, ((MultiIdKey)obj).ids);
    }

    @Override
    public int hashCode() {
      return h;
    }
  }
}
