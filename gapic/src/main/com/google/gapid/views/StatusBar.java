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
package com.google.gapid.views;

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.filling;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpacing;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowData;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;

/**
 * Displays status information at the bottom of the main window.
 */
public class StatusBar extends Composite {
  private final Composite memoryStatus;
  private final Composite replayStatus;
  private final Composite serverStatus;
  private final HeapStatus heap;
  private final Label serverPrefix;
  private final Label server;
  private final Label replay;
  private final Link notification;
  private Runnable onNotificationClick = null;

  public StatusBar(Composite parent, Theme theme) {
    super(parent, SWT.NONE);

    setLayout(withSpacing(withMargin(new GridLayout(5, false), 0, 0), 5, 0));
    withLayoutData(new Label(this, SWT.SEPARATOR | SWT.HORIZONTAL),
        withSpans(new GridData(SWT.FILL, SWT.TOP, true, false), 5, 1));

    memoryStatus = withLayoutData(
        createComposite(this, filling(new RowLayout(SWT.HORIZONTAL), true, false)),
        new GridData(SWT.LEFT, SWT.FILL, false, false));
    replayStatus = withLayoutData(
        createComposite(this, filling(new RowLayout(SWT.HORIZONTAL), true, false)),
        new GridData(SWT.LEFT, SWT.FILL, false, false));
    serverStatus = withLayoutData(
        createComposite(this, filling(new RowLayout(SWT.HORIZONTAL), true, false)),
        new GridData(SWT.LEFT, SWT.FILL, true, false));
    notification = withLayoutData(createLink(this, "", $ -> {
      if (onNotificationClick != null) {
        onNotificationClick.run();
      }
    }), new GridData(SWT.RIGHT, SWT.FILL, false, false));

    createLabel(memoryStatus, "Server:");
    heap = new HeapStatus(memoryStatus, theme);
    withLayoutData(new Label(memoryStatus, SWT.SEPARATOR | SWT.VERTICAL), new RowData(SWT.DEFAULT, 1));

    createLabel(replayStatus, "Replay:");
    replay = createLabel(replayStatus, "");
    withLayoutData(new Label(replayStatus, SWT.SEPARATOR | SWT.VERTICAL), new RowData(SWT.DEFAULT, 1));
    replayStatus.setVisible(false);

    serverPrefix = createLabel(serverStatus, "");
    server = createLabel(serverStatus, "");
    serverStatus.setVisible(false);
  }

  /**
   * Updates the notification to the given text.
   *
   * @param text the notification text.
   * @param onClick the optional notifiction click handler.
   */
  public void setNotification(String text, Runnable onClick) {
    notification.setText((onClick != null) ? "<a>" + text + "</a>" : text);
    onNotificationClick = onClick;
    layout();
  }

  public void setServerStatusPrefix(String text) {
    serverStatus.setVisible(true);
    serverPrefix.setText(text);
    layout();
  }

  public void setServerStatus(String text) {
    serverStatus.setVisible(true);
    server.setText(text);
    layout();
  }

  public void setServerHeapSize(long heapSize) {
    serverStatus.setVisible(true);
    heap.setHeap(heapSize);
    layout();
  }

  public void setReplayStatus(String text) {
    replayStatus.setVisible(true);
    replay.setText(text);
    layout();
  }

  private static class HeapStatus extends Canvas {
    private static final int PADDING = 2;

    private long heap = 0;
    private long max = 1;
    private String label = "";
    private int maxMeasuredWidth;

    public HeapStatus(Composite parent, Theme theme) {
      super(parent, SWT.NONE);

      addListener(SWT.Paint, e -> {
        Rectangle ca = getClientArea();
        e.gc.setBackground(theme.statusBarMemoryBar());
        e.gc.fillRectangle(0, 0, (int)(ca.width * heap / max), ca.height);

        Point ts = e.gc.stringExtent(label);
        e.gc.drawText(
            label, ca.width - PADDING - ts.x, (ca.height - ts.y) / 2, SWT.DRAW_TRANSPARENT);
      });
    }

    public void setHeap(long newHeap) {
      heap = newHeap;
      max = Math.max(max, newHeap);
      label = bytesToHuman(newHeap) + " of " + bytesToHuman(max);
      redraw();
    }

    @Override
    public Point computeSize(int wHint, int hHint, boolean changed) {
      Point result;
      if (label.isEmpty()) {
        result = new Point(0, 0);
      } else {
        GC gc = new GC(this);
        result = gc.stringExtent(label);
        gc.dispose();
        maxMeasuredWidth = result.x = Math.max(maxMeasuredWidth, result.x);
      }

      if (wHint != SWT.DEFAULT) {
        result.x = wHint;
      } else {
        result.x += 2 * PADDING;
      }
      if (hHint != SWT.DEFAULT) {
        result.y = hHint;
      }
      return result;
    }

    private static String bytesToHuman(long bytes) {
      long mb = bytes >> 20; // The heap is never smaller than 4MB.
      if (mb > 1024) {
        // Show GBs with a decimal.
        return String.format("%.1fGB", mb / 1024.0);
      } else {
        return mb + "MB";
      }
    }
  }
}
