/*
 * Copyright (C) 2019 Google Inc.
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

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createHorizontalSash;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Layout;
import org.eclipse.swt.widgets.Sash;

public class DrawerComposite extends Composite {
  private static final int SASH_HEIGHT = 5;
  private static final int MIN_SIZE = 50;
  private static final int TITLE_MARGIN = 5;
  private static final int BORDER_HEIGHT = 3;

  private final int titleHeight;
  private final Composite main;
  private final Sash sash;
  private final Label border;
  private final Composite drawer;
  private String title = "";
  private boolean expanded = false;
  private int drawerHeight;

  public DrawerComposite(Composite parent, int style, int initDrawerHeight, Theme theme) {
    super(parent, style);
    setLayout(new Layout() {
      @Override
      protected Point computeSize(Composite composite, int wHint, int hHint, boolean flushCache) {
        return layoutComputeSize(wHint, hHint, flushCache);
      }

      @Override
      protected void layout(Composite composite, boolean flushCache) {
        layoutLayout();
      }

      @Override
      protected boolean flushCache(Control control) {
        return true;
      }
    });

    this.drawerHeight = initDrawerHeight;
    this.titleHeight = computeTitleHeight(theme);

    main = createComposite(this, new FillLayout());
    sash = createHorizontalSash(this, e -> {
      Rectangle size = getClientArea();
      e.y = Math.max(MIN_SIZE,
          Math.min(size.height - SASH_HEIGHT - MIN_SIZE - titleHeight - BORDER_HEIGHT, e.y));
      if (e.y != ((Sash)e.widget).getBounds().y) {
        drawerHeight = size.height - e.y - SASH_HEIGHT - titleHeight - BORDER_HEIGHT;
        layout();
        redraw();
      }
    });
    border = new Label(this, SWT.SEPARATOR | SWT.HORIZONTAL);
    drawer = createComposite(this, new FillLayout());

    addListener(SWT.Paint, e -> {
      Rectangle size = getClientArea();
      int y = size.height - (expanded ? drawerHeight : 0) - titleHeight;

      e.gc.setFont(theme.selectedTabTitleFont());
      e.gc.drawText(title, TITLE_MARGIN, y + TITLE_MARGIN);
      Image img = expanded ? theme.expandMore() : theme.expandLess();
      Rectangle imgSize = img.getBounds();
      e.gc.drawImage(
          img, size.width - TITLE_MARGIN - imgSize.width, y + (titleHeight - imgSize.height) / 2);

    });
    addListener(SWT.MouseMove, e -> {
      setCursor(isOnDrawerBar(e) ? getDisplay().getSystemCursor(SWT.CURSOR_HAND) : null);
    });
    addListener(SWT.MouseExit, e -> {
      setCursor(null);
    });
    addListener(SWT.MouseUp, e -> {
      if (isOnDrawerBar(e)) {
        toggle();
      }
    });
  }

  private boolean isOnDrawerBar(Event e) {
    Rectangle size = getClientArea();
    int h = size.height - (expanded ? drawerHeight : 0) - titleHeight;

    // When collapsed, treat the entire title-bar as the click/hover target, otherwise, just the
    // titleHeight-by-titleHeight square on the right edge, where the icon is drawn.
    if (e.y < h || e.y >= h + titleHeight) {
      return false;
    } else if (!expanded) {
      return true;
    }
    return e.x >= size.width - TITLE_MARGIN - titleHeight && e.x < size.width - TITLE_MARGIN;
  }

  private int computeTitleHeight(Theme theme) {
    GC gc = new GC(this);
    try {
      gc.setFont(theme.selectedTabTitleFont());
      return gc.getFontMetrics().getHeight() + 2 * TITLE_MARGIN;
    } finally {
      gc.dispose();
    }
  }

  public void setText(String text) {
    this.title = text;
  }

  public void setDrawerHeight(int drawerHeight) {
    this.drawerHeight = drawerHeight;
    layout();
    redraw();
  }

  public int getDrawerHeight() {
    return drawerHeight;
  }

  public Composite getMain() {
    return main;
  }

  public Composite getDrawer() {
    return drawer;
  }

  public void toggle() {
    expanded = !expanded;
    layout();
    redraw();
  }

  public void setExpanded(boolean expanded) {
    if (this.expanded != expanded) {
      toggle();
    }
  }

  protected Point layoutComputeSize(int wHint, int hHint, boolean flushCache) {
    int height = (hHint == SWT.DEFAULT) ? SWT.DEFAULT :
      hHint - (expanded ? drawerHeight + SASH_HEIGHT : 0) - titleHeight - BORDER_HEIGHT;
    Point mainSize = main.computeSize(wHint, height, flushCache);
    Point drawerSize = drawer.computeSize(wHint, drawerHeight, flushCache);
    return new Point(Math.max(mainSize.x, drawerSize.x),
        mainSize.y + BORDER_HEIGHT + titleHeight + (expanded ? SASH_HEIGHT + drawerHeight : 0));
  }

  protected void layoutLayout() {
    Rectangle size = getClientArea();
    if (expanded) {
      int y = size.height - SASH_HEIGHT - BORDER_HEIGHT - titleHeight - drawerHeight;
      main.setBounds(0, 0, size.width, y);
      sash.setBounds(0, y, size.width, SASH_HEIGHT);
      sash.setVisible(true);
      border.setBounds(0, y + SASH_HEIGHT, size.width, BORDER_HEIGHT);
      drawer.setBounds(0, y + SASH_HEIGHT + BORDER_HEIGHT + titleHeight, size.width, drawerHeight);
      drawer.setVisible(true);
    } else {
      int y = size.height - BORDER_HEIGHT - titleHeight;
      main.setBounds(0, 0, size.width, y);
      sash.setBounds(0, 0, 0, 0);
      sash.setVisible(false);
      border.setBounds(0, y, size.width, BORDER_HEIGHT);
      drawer.setBounds(0, 0, 0, 0);
      drawer.setVisible(false);
    }
  }
}
