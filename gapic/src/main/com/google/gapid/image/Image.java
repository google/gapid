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
import com.google.gapid.glviewer.gl.Texture;
import com.google.gapid.proto.stream.Stream.Channel;
import java.nio.DoubleBuffer;
import java.util.Set;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.PaletteData;
import org.eclipse.swt.graphics.RGB;
import org.lwjgl.opengl.GL11;

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
  public Set<Channel> getChannels();

  /**
   * @return all the pixel values of the given channel.
   */
  public DoubleBuffer getChannel(Channel channel);

  /**
   * @return true if this image contains high-dynamic-range data.
   */
  public boolean isHDR();

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
    public Set<Channel> getChannels() {
      return Sets.newIdentityHashSet();
    }

    @Override
    public DoubleBuffer getChannel(Channel channel) {
      return DoubleBuffer.allocate(0);
    }

    @Override
    public boolean isHDR() {
      return false;
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
      public float getMin() {
        return 0;
      }

      @Override
      public float getMax() {
        return 1;
      }

      @Override
      public float getAlphaMin() {
        return 1;
      }

      @Override
      public float getAlphaMax() {
        return 1;
      }
    };

    /**
     * @return the minimum value across all channels of the image data. Used for tone mapping.
     */
    public float getMin();

    /**
     * @return the maximum value across all channels of the image data. Used for tone mapping.
     */
    public float getMax();

    /**
     * @return the minimum alpha value of the image data.
     */
    public float getAlphaMin();

    /**
     * @return the maximum alpha value of the image data.
     */
    public float getAlphaMax();
  }
}
