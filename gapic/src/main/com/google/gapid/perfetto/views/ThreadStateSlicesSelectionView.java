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
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.packColumns;

import com.google.gapid.perfetto.models.ThreadTrack;

import org.eclipse.jface.viewers.IStructuredContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected thread state slices.
 */
public class ThreadStateSlicesSelectionView extends Composite {
  public ThreadStateSlicesSelectionView(
      Composite parent, ThreadTrack.StateSlices slices) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TableViewer viewer = createTableViewer(this, SWT.NONE);
    viewer.setContentProvider(new IStructuredContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return slices.entries.toArray();
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTableColumn(viewer, "State",
        e -> ((ThreadTrack.StateSlices.Entry)e).state.label);
    createTableColumn(viewer, "Duration",
        e -> timeToString(((ThreadTrack.StateSlices.Entry)e).totalDur));
    viewer.setInput(slices);
    packColumns(viewer.getTable());
  }
}
