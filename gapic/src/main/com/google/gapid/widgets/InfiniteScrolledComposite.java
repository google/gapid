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

import com.google.gapid.util.BigPoint;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.widgets.CopyPaste.CopySource;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.ScrolledComposite;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionAdapter;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.ScrollBar;

import java.math.BigInteger;
import java.util.function.IntConsumer;

/**
 * A {@link ScrolledComposite} where the contents can be too large in one or both dimensions to use
 * a standard scrollbar. The scrollbar thumb will jump back to the center position when let go,
 * while scrolling through the middle of the bulk of the scroll range.
 */
public class InfiniteScrolledComposite extends ScrolledComposite {
  private static final BigInteger MAX_NON_INFINTE =
      BigInteger.valueOf(DPIUtil.autoScaleDown((1 << 15)) - 1);

  private final Scrollable contents;
  private final Canvas canvas;
  private final ScrollHandler xHandler, yHandler;

  public InfiniteScrolledComposite(Composite parent, int style, Scrollable contents) {
    super(parent, style);
    this.contents = contents;
    canvas = new Canvas(this, SWT.DOUBLE_BUFFERED);
    this.xHandler = new ScrollHandler(getHorizontalBar(), x -> {
      canvas.setLocation(x, canvas.getLocation().y);
      canvas.redraw();
    });
    this.yHandler = new ScrollHandler(getVerticalBar(), y -> {
      canvas.setLocation(canvas.getLocation().x, y);
      canvas.redraw();
    });

    setContent(canvas);
    setExpandVertical(true);
    setExpandHorizontal(true);

    updateMinSize();
    addListener(SWT.Resize, e -> updateMinSize());
    canvas.addListener(SWT.Paint, e -> contents.paint(xHandler.offset, yHandler.offset, e.gc, e.getBounds()));
  }

  public BigPoint getLocation(Event e) {
    return new BigPoint(
        xHandler.offset.add(BigInteger.valueOf(e.x)),
        yHandler.offset.add(BigInteger.valueOf(e.y)));
  }

  public BigPoint getLocation(MouseEvent e) {
    return new BigPoint(
        xHandler.offset.add(BigInteger.valueOf(e.x)),
        yHandler.offset.add(BigInteger.valueOf(e.y)));
  }

  public BigPoint getMouseLocation() {
    Display disp = getDisplay();
    Point mouse = disp.map(null, canvas, disp.getCursorLocation());
    return new BigPoint(
        xHandler.offset.add(BigInteger.valueOf(mouse.x)),
        yHandler.offset.add(BigInteger.valueOf(mouse.y)));
  }

  public BigPoint getScrollLocation() {
    return new BigPoint(
        xHandler.offset.add(BigInteger.valueOf(xHandler.bar.getSelection())),
        yHandler.offset.add(BigInteger.valueOf(yHandler.bar.getSelection())));
  }

  public void scrollTo(BigInteger x, BigInteger y) {
    xHandler.scrollTo(x);
    yHandler.scrollTo(y);
    Widgets.scheduleIfNotDisposed(canvas, canvas::redraw);
  }

  @Override
  public void redraw() {
    super.redraw();
    canvas.redraw();
  }

  public void addContentListener(int type, Listener listener) {
    canvas.addListener(type, listener);
  }

  /**
   * Registers the given listener on the contents. Note that the selection events are not included.
   */
  public void addContentListener(MouseAdapter listener) {
    canvas.addMouseListener(listener);
    canvas.addMouseMoveListener(listener);
    canvas.addMouseWheelListener(listener);
    canvas.addMouseTrackListener(listener);
    xHandler.bar.addSelectionListener(listener);
    yHandler.bar.addSelectionListener(listener);
  }

  public void registerContentAsCopySource(CopyPaste copyPaste, CopySource source) {
    copyPaste.registerCopySource(canvas, source);
  }

  public void updateMinSize() {
    BigInteger w = contents.getWidth(), h = contents.getHeight();
    Rectangle size = getClientArea();
    xHandler.setEnabled(w.compareTo(MAX_NON_INFINTE) > 0);
    yHandler.setEnabled(h.compareTo(MAX_NON_INFINTE) > 0);
    xHandler.max = w.subtract(BigInteger.valueOf(size.width));
    yHandler.max = h.subtract(BigInteger.valueOf(size.height));
    int mw = (xHandler.enabled ? MAX_NON_INFINTE : w).intValueExact();
    int mh = (yHandler.enabled ? MAX_NON_INFINTE : h).intValueExact();
    setMinSize(mw, mh);
  }

  /**
   * Contents to be scrolled.
   */
  public static interface Scrollable {
    /**
     * @return the height of the scrolling contents in SWT points.
     */
    public BigInteger getHeight();

    /**
     * @return the width of the scrolling contents in SWT points.
     */
    public BigInteger getWidth();

    /**
     * Paints the contents at the given location using the given graphics context.
     */
    public void paint(BigInteger xOffset, BigInteger yOffset, GC gc, Rectangle area);
  }

  /**
   * Hanldes the scroll events of an infinite scrollbar.
   */
  private static class ScrollHandler extends SelectionAdapter {
    public final ScrollBar bar;
    private final IntConsumer updateLocation;
    public BigInteger offset = BigInteger.ZERO;
    public BigInteger max = BigInteger.ONE;
    public boolean enabled = false;

    public ScrollHandler(ScrollBar bar, IntConsumer updateLocation) {
      this.bar = bar;
      this.updateLocation = updateLocation;
      bar.addSelectionListener(this);
    }

    public void setEnabled(boolean enabled) {
      if (enabled != this.enabled) {
        this.enabled = enabled;
        this.offset = BigInteger.ZERO;
        this.max = BigInteger.ONE;
      }
    }

    public void scrollTo(BigInteger pos) {
      if (enabled) {
        int mid = getMidpoint(bar); BigInteger bigMid = BigInteger.valueOf(mid);
        if (pos.compareTo(bigMid) <= 0) {
          // Going to the top.
          offset = BigInteger.ZERO;
          bar.setSelection(pos.intValue());
          updateLocation.accept(-pos.intValue());
        } else if (pos.compareTo(max.subtract(bigMid)) >= 0) {
          // Going to the bottom.
          offset = max.subtract(bigMid).subtract(bigMid);
          int selection = pos.min(max).subtract(offset).intValue();
          bar.setSelection(selection);
          updateLocation.accept(-selection);
        } else {
          // Going somewhere in the middle.
          offset = pos.subtract(bigMid);
          bar.setSelection(mid);
          updateLocation.accept(-mid);
        }
      } else {
        if (pos.compareTo(BigInteger.valueOf(bar.getMaximum() - bar.getThumb())) <= 0) {
          int selection = pos.intValue();
          bar.setSelection(selection);
          updateLocation.accept(-selection);
        }
      }
    }

    @Override
    public void widgetSelected(SelectionEvent e) {
      if (enabled) {
        int selection = bar.getSelection();
        if ((e.stateMask & SWT.BUTTON1) != 0 &&
            (e.detail == SWT.NONE || e.detail == SWT.DRAG)) {
          // The user is scrolling by dragging the thumb thingie. Don't do anything.
        } else {
          // User has released the mouse or is scrolling in another fashion.
          updateLocation.accept(-update(selection));
        }
      }
    }

    private int update(int selection) {
      int mid = getMidpoint(bar);
      BigInteger maxOffset = max.subtract(BigInteger.valueOf(2 * mid));
      if (selection < mid && offset.equals(BigInteger.ZERO)) {
        // We're scrolling near the top. Don't do anything.
      } else if (selection > mid && offset.compareTo(maxOffset) >= 0) {
        // We're scrolling near the bottom. Don't do anything.
      } else if (selection == mid) {
        // Don't do anything. Since the bar hasn't moved.
      } else {
        offset = offset.add(BigInteger.valueOf(selection - mid))
            .max(BigInteger.ZERO).min(maxOffset);
        bar.setSelection(mid);
        selection = mid;
      }
      return selection;
    }

    private static int getMidpoint(ScrollBar bar) {
      int min = bar.getMinimum(), max = bar.getMaximum(), thumb = bar.getThumb();
      return (max - thumb - min) / 2 + min;
    }
  }
}
