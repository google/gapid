/*
 * Copyright (C) 2020 Google Inc.
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

import com.google.gapid.perfetto.models.BatterySummaryTrack;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

public class BatterySelectionView extends Composite {
  public BatterySelectionView(Composite parent, State state, BatterySummaryTrack.Values sel) {
    super(parent, SWT.None);
    setLayout(new FillLayout());

    TableViewer viewer = createTableViewer(this, SWT.NONE);
    viewer.setContentProvider(new ArrayContentProvider());
    viewer.setLabelProvider(new LabelProvider());

    createTableColumn(
        viewer, "Time", r -> timeToString(sel.ts[(Integer)r] - state.getTraceTime().start));
    createTableColumn(viewer, "Dur", r -> String.valueOf(sel.dur[(Integer)r]));
    createTableColumn(viewer, "Capacity", r -> String.valueOf(sel.capacity[(Integer)r]));
    createTableColumn(viewer, "Charge", r -> String.valueOf(sel.charge[(Integer)r]));
    createTableColumn(viewer, "Current", r -> String.valueOf(sel.current[(Integer)r]));

    Integer[] rows = new Integer[sel.ts.length];
    for (int i = 0; i < rows.length; i++) {
      rows[i] = i;
    }
    viewer.setInput(rows);
    packColumns(viewer.getTable());
  }
}
