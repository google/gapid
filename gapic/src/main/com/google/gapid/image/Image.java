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

import com.google.gapid.glviewer.gl.Texture;

import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.PaletteData;
import org.eclipse.swt.graphics.RGB;
import org.lwjgl.opengl.GL11;

public interface Image {
  public int getWidth();
  public int getHeight();
  public ImageBuffer getData();

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
    public ImageBuffer getData() {
      return ImageBuffer.EMPTY_BUFFER;
    }
  };

  public static interface ImageBuffer {
    public static final ImageBuffer EMPTY_BUFFER = new ImageBuffer() {
      @Override
      public void uploadToTexture(Texture texture) {
        texture.loadData(0, 0, GL11.GL_RGB, GL11.GL_RGB, GL11.GL_UNSIGNED_BYTE, null);
      }

      @Override
      public ImageData getImageData() {
        return EMPTY_IMAGE;
      }

      @Override
      public PixelValue getPixel(int x, int y) {
        return PixelValue.NULL_PIXEL;
      }

      @Override
      public PixelInfo getInfo() {
        return PixelInfo.NULL_INFO;
      }
    };

    public void uploadToTexture(Texture texture);
    public ImageData getImageData();
    public PixelValue getPixel(int x, int y);
    public PixelInfo getInfo();
  }

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

    public boolean isDark();
    @Override
    public String toString();
  }

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
    };

    public float getMin();
    public float getMax();
  }
}
