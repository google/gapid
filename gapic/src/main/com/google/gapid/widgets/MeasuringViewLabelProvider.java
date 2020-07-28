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

import com.google.gapid.views.Formatter.LinkableStyledString;
import com.google.gapid.views.Formatter.StylingString;

import org.eclipse.jface.viewers.ColumnViewer;
import org.eclipse.jface.viewers.StyledCellLabelProvider;
import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.ViewerCell;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.TextLayout;
import org.eclipse.swt.widgets.Widget;

import java.util.Arrays;

/**
 * A {@link StyledCellLabelProvider} that can be used to measure the size of the label to correctly
 * determine if the the mouse pointer is hovering over a link in the {@link StylingString}.
 */
public abstract class MeasuringViewLabelProvider extends StyledCellLabelProvider {
  private final ColumnViewer viewer;
  private final Theme theme;
  private final TextLayout layout;

  public MeasuringViewLabelProvider(ColumnViewer viewer, Theme theme) {
    this.viewer = viewer;
    this.theme = theme;
    this.layout = new TextLayout(viewer.getControl().getDisplay());
  }

  @Override
  public void dispose() {
    super.dispose();
    layout.dispose();
  }

  @Override
  public void update(ViewerCell cell) {
    // Adjusted from the DelegatingStyledCellLabelProvider implementation.

    StyledString styledString =
        format(cell.getItem(), cell.getElement(), LinkableStyledString.ignoring(theme)).getString();
    String newText = styledString.toString();

    StyleRange[] oldStyleRanges = cell.getStyleRanges();
    StyleRange[] newStyleRanges = styledString.getStyleRanges();

    if (!Arrays.equals(oldStyleRanges, newStyleRanges)) {
      cell.setStyleRanges(newStyleRanges);
      if (cell.getText().equals(newText)) {
        cell.setText("");
      }
    }
    Color bgcolor = getBackgroundColor(cell.getElement());
    if (bgcolor != null) {
      cell.setBackground(bgcolor);
    }
    cell.setImage(getImage(cell.getElement()));
    cell.setText(newText);
  }

  protected Image getImage(@SuppressWarnings("unused") Object element) {
    return null;
  }

  protected Color getBackgroundColor(@SuppressWarnings("unused") Object element) {
    return null;
  }

  protected abstract <S extends StylingString> S format(Widget item, Object element, S string);

  public Object getFollow(Point point) {
    ViewerCell cell = viewer.getCell(point);
    if (cell == null || !isFollowable(cell.getElement())) {
      return null;
    }

    LinkableStyledString string =
        format(cell.getItem(), cell.getElement(), LinkableStyledString.create(theme));
    string.endLink();
    string.append("placeholder", string.defaultStyle());
    updateLayout(cell, string.getString());

    Rectangle bounds = cell.getTextBounds();
    int offset = layout.getOffset(point.x - bounds.x, point.y - bounds.y, null);
    return string.getLinkTarget(offset);
  }

  protected abstract boolean isFollowable(Object element);

  private void updateLayout(ViewerCell cell, StyledString string) {
    // Adjusted from similar method from super class.
    layout.setStyle(null, 0, Integer.MAX_VALUE);
    layout.setText(string.toString());

    for (StyleRange range : cell.getStyleRanges()) {
      layout.setStyle(range, range.start, range.start + range.length - 1);
    }
  }
}
