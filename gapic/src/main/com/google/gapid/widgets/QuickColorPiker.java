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
package com.google.gapid.widgets;

import static com.google.gapid.image.Images.createImageData;

import com.google.gapid.image.Images;
import com.google.gapid.util.MouseAdapter;

import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.RGB;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;

/**
 * A quick-and-dirty color picker that sacrifices some colors for the sake of compactness.
 * Not all colors will be "pickable" as a rectangle in HSL colorspace is shown keeping S fixed
 * at 1 (i.e. showing the surface of the HSL cylinder).
 *
 * Do not use this as a general purpose color picker, as some colors are not shown (e.g. grays).
 */
public class QuickColorPiker extends Canvas {
  private static final int SEXTANT = 60;
  private static final int WIDTH = 6 * SEXTANT;

  protected final ImageData imageData;
  private final Image image;

  public QuickColorPiker(Composite parent, int height, Listener listener) {
    super(parent, SWT.NONE);
    imageData = generate(height);
    image = Images.createAutoScaledImage(getDisplay(), imageData);
    setCursor(getDisplay().getSystemCursor(SWT.CURSOR_CROSS));

    addListener(SWT.Paint, e -> e.gc.drawImage(image, 0, 0));
    MouseAdapter mouseHandler = new MouseAdapter() {
      private RGB color = new RGB(0, 0, 0);

      @Override
      public void mouseDown(MouseEvent e) {
        update(e.x, e.y);
      }

      @Override
      public void mouseUp(MouseEvent e) {
        update(e.x, e.y);
      }

      @Override
      public void mouseMove(MouseEvent e) {
        if ((e.stateMask & SWT.BUTTON1) != 0) {
          update(e.x, e.y);
        }
      }

      private void update(int x, int y) {
        x = Math.max(0, Math.min(x, imageData.width - 1));
        y = Math.max(0, Math.min(y, imageData.height - 1));

        int idx = (y * imageData.bytesPerLine) + (x * 3);
        RGB newColor = new RGB(
            imageData.data[idx + 0] & 0xFF,
            imageData.data[idx + 1] & 0xFF,
            imageData.data[idx + 2] & 0xFF);
        if (!color.equals(newColor)) {
          color = newColor;
          listener.onColorChanged(newColor);
        }
      }
    };
    addMouseListener(mouseHandler);
    addMouseMoveListener(mouseHandler);
    addListener(SWT.Dispose, e -> image.dispose());
  }

  @Override
  public Point computeSize(int wHint, int hHint, boolean changed) {
    return new Point(imageData.width, imageData.height);
  }

  private static ImageData generate(int height) {
    ImageData image = createImageData(WIDTH, height, false);
    // Iterate over H and L, keeping S fixed at 1.
    for (int y = 0, i = 0; y < height; y++) {
      for (int h = 0; h < WIDTH; h++, i += 3) {
        float l = 1 - (float)y / (height - 1); // Top of image, y = 0, L = 1.

        // Convert HSL to RGB (https://en.wikipedia.org/wiki/HSL_and_HSV). Simplified with S = 1.
        float c = (1 - Math.abs(2 * l - 1)); // chroma
        float x = c * (1 - Math.abs((((float)h / SEXTANT) % 2) - 1));
        float r, g, b;
        switch (h / SEXTANT) {
          case 0: r = c; g = x; b = 0; break;
          case 1: r = x; g = c; b = 0; break;
          case 2: r = 0; g = c; b = x; break;
          case 3: r = 0; g = x; b = c; break;
          case 4: r = x; g = 0; b = c; break;
          case 5: r = c; g = 0; b = x; break;
          // This should not happen, if it does - really, it won't - simply ignore that column.
          default: continue;
        }
        float m = l - c / 2; // lightness adjustment
        image.data[i + 0] = (byte)((r + m) * 255);
        image.data[i + 1] = (byte)((g + m) * 255);
        image.data[i + 2] = (byte)((b + m) * 255);
      }
    }
    return image;
  }

  public static interface Listener {
    public void onColorChanged(RGB rgb);
  }
}
