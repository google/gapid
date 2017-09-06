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

import static com.google.gapid.widgets.Widgets.redrawIfNotDisposed;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.gapid.image.Images;
import com.google.gapid.util.Loadable;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.function.Function;

/**
 * A {@link Composite} whith the {@link Loadable} mixin. Will show a loading indicator while the
 * underlying data is being loaded and then swap over to the main contents once it is done.
 */
public class LoadablePanel<C extends Control> extends Composite implements Loadable {
  private static final int LOADING_INDICATOR_DELAY_MS = 250;

  private final Widgets widgets;
  private final Control loading;
  private final MessageWidget message;
  private final C contents;
  private boolean shouldShowLoading;

  public LoadablePanel(
      Composite parent, Widgets widgets, Function<LoadablePanel<C>, C> createContents) {
    super(parent, SWT.NONE);
    this.widgets = widgets;

    setLayout(new StackLayout());

    loading = new LoadingWidget(this, widgets.loading);
    message = new MessageWidget(this);
    contents = createContents.apply(this);
  }

  public static <C extends Control> LoadablePanel<C> create(
      Composite parent, Widgets widgets, Function<LoadablePanel<C>, C> createContents) {
    return new LoadablePanel<C>(parent, widgets, createContents);
  }

  public C getContents() {
    return contents;
  }

  @Override
  public StackLayout getLayout() {
    return (StackLayout)super.getLayout();
  }

  public boolean isLoading() {
    return shouldShowLoading || getLayout().topControl == loading;
  }

  @Override
  public void startLoading() {
    shouldShowLoading = true;

    // If we start and then stop quickly the UI would flash for no reason. Give it some time
    // before we actually show the loading indicator.
    scheduleIfNotDisposed(this, LOADING_INDICATOR_DELAY_MS, () -> {
      if (shouldShowLoading) {
        getLayout().topControl = loading;
        requestLayout();
      }
    });
  }

  @Override
  public void stopLoading() {
    shouldShowLoading = false;
    getLayout().topControl = contents;
    requestLayout();
  }

  @Override
  public void showMessage(MessageType type, String text) {
    shouldShowLoading = false;
    message.setText(text, getImage(type));
    getLayout().topControl = message;
    requestLayout();
  }

  private Image getImage(MessageType type) {
    switch (type) {
      case Smile: return widgets.theme.smile();
      case Error: return widgets.theme.error();
      default: return null;
    }
  }

  private static class LoadingWidget extends Canvas {
    public LoadingWidget(Composite parent, LoadingIndicator loading) {
      super(parent, SWT.DOUBLE_BUFFERED);

      addListener(SWT.Paint, e -> {
        loading.paint(e.gc, 0, 0, getSize());
        loading.scheduleForRedraw(() -> redrawIfNotDisposed(this));
      });
    }
  }

  /**
   * {@link Canvas} that renders the
   * {@link Loadable#showMessage(com.google.gapid.util.Loadable.MessageType, String)} message.
   */
  private static class MessageWidget extends Canvas {
    private String text = "";
    private Image image;

    public MessageWidget(Composite parent) {
      super(parent, SWT.DOUBLE_BUFFERED);

      addListener(SWT.Paint, e -> {
        if (text.isEmpty()) {
          return;
        }

        Point imageSize = (image == null) ? new Point(0, 0) : Images.getSize(image);
        int border = (image == null) ? 0 : 5;
        Point textSize = e.gc.textExtent(text, SWT.DRAW_TRANSPARENT | SWT.DRAW_DELIMITER);
        Point size = getSize();

        int x = (size.x - imageSize.x - border - textSize.x) / 2;
        if (image != null) {
          e.gc.drawImage(image, x, (size.y - imageSize.y) / 2);
          x += imageSize.x + border;
        }
        e.gc.drawText(
            text, x, (size.y - textSize.y) / 2, SWT.DRAW_TRANSPARENT | SWT.DRAW_DELIMITER);
      });
    }

    public void setText(String text, Image image) {
      this.text = (text == null) ? "" : text.trim();
      this.image = image;
      redraw();
    }
  }
}
