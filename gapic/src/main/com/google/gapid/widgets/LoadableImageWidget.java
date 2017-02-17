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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Thumbnails;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;

public class LoadableImageWidget extends Canvas {
  private LoadableImage image;

  private LoadableImageWidget(Composite parent) {
    super(parent, SWT.BORDER);

    setSize(Thumbnails.THUMB_SIZE, Thumbnails.THUMB_SIZE);
    addListener(SWT.Paint, e -> paint(e.gc));
  }

  public static LoadableImageWidget forImageData(
      Composite parent, ListenableFuture<ImageData> future, LoadingIndicator loading) {
    LoadableImageWidget result = new LoadableImageWidget(parent);
    result.image = LoadableImage.forImageData(result, future, loading, result::redrawIfNotDisposed);
    return result;
  }

  public static LoadableImageWidget forImage(
      Composite parent, ListenableFuture<Image> future, LoadingIndicator loading) {
    LoadableImageWidget result = new LoadableImageWidget(parent);
    result.image = LoadableImage.forImage(result, future, loading, result::redrawIfNotDisposed);
    return result;
  }

  public static LoadableImageWidget forSmallImageData(
      Composite parent, ListenableFuture<ImageData> future, LoadingIndicator loading) {
    LoadableImageWidget result = new LoadableImageWidget(parent);
    result.image = LoadableImage.forSmallImageData(
        result, future, loading, result::redrawIfNotDisposed);
    return result;
  }

  public static LoadableImageWidget forSmallImage(
      Composite parent, ListenableFuture<Image> future, LoadingIndicator loading) {
    LoadableImageWidget result = new LoadableImageWidget(parent);
    result.image = LoadableImage.forSmallImage(result, future, loading, result::redrawIfNotDisposed);
    return result;
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
  public void dispose() {
    image.dispose();
    super.dispose();
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
