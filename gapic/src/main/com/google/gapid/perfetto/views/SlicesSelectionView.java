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
package com.google.gapid.perfetto.views;

import static com.google.gapid.perfetto.TimeSpan.timeToString;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;

import com.google.gapid.perfetto.models.SliceTrack;

import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected slices.
 */
public class SlicesSelectionView extends Composite {
  public SlicesSelectionView(Composite parent, SliceTrack.Slices sel) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TreeViewer viewer = createTreeViewer(this, SWT.NONE);
    viewer.getTree().setHeaderVisible(true);
    viewer.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return sel.nodes.toArray();
      }

      @Override
      public boolean hasChildren(Object element) {
        return !n(element).children.isEmpty();
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }

      @Override
      public Object[] getChildren(Object element) {
        return n(element).children.toArray();
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTreeColumn(viewer, "Name", e -> n(e).name);
    createTreeColumn(viewer, "Wall Time", e -> timeToString(n(e).dur));
    createTreeColumn(viewer, "Self Time", e -> timeToString(n(e).self));
    createTreeColumn(viewer, "Count", e -> Integer.toString(n(e).count));
    createTreeColumn(viewer, "Avg Wall Time", e -> timeToString(n(e).dur / n(e).count));
    createTreeColumn(viewer, "Avg Self TIme", e -> timeToString(n(e).self / n(e).count));
    viewer.setInput(sel);
    packColumns(viewer.getTree());
  }

  protected static SliceTrack.Node n(Object o) {
    return (SliceTrack.Node)o;
  }
}
