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

import com.google.gapid.models.ImagesModel;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;

/**
 * A {@link Canvas} displaying a {@link LoadableImage}.
 */
public class LoadableImageWidget extends Canvas {
  private LoadableImage image;

  private LoadableImageWidget(Composite parent) {
    super(parent, SWT.BORDER);

    setSize(ImagesModel.THUMB_SIZE, ImagesModel.THUMB_SIZE);
    addListener(SWT.Paint, e -> paint(e.gc));
    addListener(SWT.Dispose, e -> image.dispose());
  }

  public static LoadableImageWidget forImage(Composite parent, LoadableImage.Builder image) {
    LoadableImageWidget result = new LoadableImageWidget(parent);
    result.image = image.build(result, result::redrawIfNotDisposed);
    return result;
  }

  public LoadableImageWidget withImageEventListener(LoadableImage.Listener listener) {
    image.addListener(listener);
    return this;
  }

  private void redrawIfNotDisposed() {
    if (!isDisposed()) {
      redraw();
    }
  }

  protected void paint(GC gc) {
    Image toDraw = image.getImage();
    Rectangle imageSize = toDraw.getBounds(), size = getBounds();
    gc.drawImage(toDraw, 0, 0, imageSize.width, imageSize.height,
        (size.width - imageSize.width) / 2, (size.height - imageSize.height) / 2,
        imageSize.width, imageSize.height);
  }

  @Override
  public Point computeSize(int wHint, int hHint, boolean changed) {
    if (image.hasFinished()) {
      Rectangle size = image.getImage().getBounds();
      return new Point(size.width, size.height);
    } else {
      return getSize();
    }
  }
}
