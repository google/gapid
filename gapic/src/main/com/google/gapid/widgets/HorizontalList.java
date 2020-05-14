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

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.ScrollBar;

import java.util.Arrays;

/**
 * A widget that displays custom drawn items in a horizontal row. Allows for an arbitrary number of
 * items with arbitrary sizes.
 *
 * Note: do not interact with the {@link Composite} interface, as it will break the functionality.
 */
public abstract class HorizontalList extends Composite {
  private static final int MARGIN = 5;

  private final ScrollBar hBar, vBar;
  private final Canvas canvas;
  private Item[] items = new Item[0];
  private int maxHeight = 0;

  public HorizontalList(Composite parent) {
    this(parent, 0);
  }

  public HorizontalList(Composite parent, int style) {
    super(parent, SWT.H_SCROLL | SWT.V_SCROLL | style);
    this.hBar = getHorizontalBar();
    this.vBar = getVerticalBar();

    setLayout(new FillLayout(SWT.HORIZONTAL));

    canvas = new Canvas(this, SWT.DOUBLE_BUFFERED);
    canvas.addListener(SWT.Paint, e -> {
      int offset = hBar.getSelection();
      int start = Arrays.binarySearch(items, null, (i, ignored) -> Integer.compare(i.x, offset));
      if (start < 0) {
        start = Math.max(0, -start - 2); // extra -1 to make sure partial items are drawn.
      }
      int y = MARGIN - vBar.getSelection();
      Rectangle size = getClientArea();

      for (int i = start; i < items.length && (items[i].x - offset) < size.width ; i++) {
        Item item = items[i];
        paint(e.gc, i, item.x - offset, y, item.width, item.height);
      }
    });
    hBar.addListener(SWT.Selection, e -> canvas.redraw());
    vBar.addListener(SWT.Selection, e -> canvas.redraw());

    addListener(SWT.Resize, e -> updateScrollbar());
    addListener(SWT.Show, e -> updateScrollbar());
  }

  protected abstract void paint(GC gc, int index, int x, int y, int w, int h);

  public void setItemCount(int count, int initWidth, int initHeight) {
    items = new Item[count];
    for (int i = 0, x = MARGIN; i < count; i++, x += initWidth + MARGIN) {
      items[i] = new Item(initWidth, initHeight, x);
    }
    maxHeight = initHeight;

    if (isVisible()) {
      updateScrollbar();
      repaint();
    }
  }

  public void setItemSize(int index, int width, int height) {
    int dx = width - items[index].width, oldHeight = items[index].height;
    items[index].width = width;
    items[index].height = height;

    if (dx != 0) {
      for (int i = index + 1; i < items.length; i++) {
        items[i].x += dx;
      }
    }

    if (height > oldHeight) {
      maxHeight = Math.max(maxHeight, height);
    } else if (height < oldHeight && oldHeight == maxHeight) {
      maxHeight = height;
      for (Item item : items) {
        maxHeight = Math.max(item.height, maxHeight);
      }
    }

    if (isVisible()) {
      updateScrollbar();
      repaint();
    }
  }

  public void repaint() {
    if (!isDisposed()) {
      canvas.redraw();
    }
  }

  public int getItemAt(int x) {
    int search = x + hBar.getSelection();
    int index = Arrays.binarySearch(items, null, (i, ignored) -> Integer.compare(i.x, search));
    return (index < 0) ? Math.max(0, -index - 2) : index;
  }

  public void addContentListener(int event, Listener listener) {
    canvas.addListener(event, listener);
  }

  public void scrollIntoView(int index) {
    if (index < 0 || index >= items.length) {
      return;
    }

    int min = items[index].x, max = min + items[index].width + MARGIN;
    int offset = hBar.getSelection(), size = getClientArea().width;
    if (min < offset) {
      hBar.setSelection(min - 5 * MARGIN);
    } else if (max > offset + size) {
      hBar.setSelection(max - size + 5 * MARGIN);
    }
  }

  private void updateScrollbar() {
    if (items.length == 0) {
      hBar.setVisible(false);
      vBar.setVisible(false);
      return;
    }

    Rectangle size = getClientArea();
    Item last = items[items.length - 1];
    int extend = last.x + last.width + 2 * MARGIN;
    if (extend < size.width) {
      hBar.setVisible(false);
      hBar.setValues(0, 0, 5, 5, 1, 2);
    } else {
      hBar.setVisible(true);
      int max = extend - size.width, thumb = Math.max(10, size.width * size.width / extend);
      hBar.setValues(Math.min(hBar.getSelection(), max), 0, max + thumb, thumb, 100, 1000);
    }

    extend = maxHeight + 2 * MARGIN;
    if (size.height > extend) {
      vBar.setVisible(false);
      vBar.setValues(0, 0, 5, 5, 1, 2);
    } else {
      vBar.setVisible(true);
      int max = extend - size.height, thumb = Math.max(10, size.height * size.height / extend);
      vBar.setValues(
          Math.min(vBar.getSelection(), max), 0, max + thumb, thumb, extend / 10, extend / 2);
    }
  }

  private static class Item {
    public int width, height;
    public int x;

    public Item(int width, int height, int x) {
      this.width = width;
      this.height = height;
      this.x = x;
    }
  }
}