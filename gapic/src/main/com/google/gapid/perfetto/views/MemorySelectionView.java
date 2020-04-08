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

import com.google.gapid.perfetto.models.MemorySummaryTrack;
import com.google.gapid.perfetto.models.ProcessMemoryTrack;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

import java.util.function.Consumer;

public class MemorySelectionView extends Composite {
  private MemorySelectionView(Composite parent, int numRows, Consumer<TableViewer> createColumns) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TableViewer viewer = createTableViewer(this, SWT.NONE);
    viewer.setContentProvider(new ArrayContentProvider());
    viewer.setLabelProvider(new LabelProvider());

    createColumns.accept(viewer);

    Integer[] rows = new Integer[numRows];
    for (int i = 0; i < rows.length; i++) {
      rows[i] = i;
    }
    viewer.setInput(rows);
    packColumns(viewer.getTable());
  }

  public MemorySelectionView(Composite parent, State state, MemorySummaryTrack.Values sel) {
    this(parent, sel.ts.length, viewer -> {
      createTableColumn(
          viewer, "Time", r -> timeToString(sel.ts[(Integer)r] - state.getTraceTime().start));
      createTableColumn(viewer, "Dur", r -> timeToString(sel.dur[(Integer)r]));
      createTableColumn(viewer, "Total", r -> String.valueOf(sel.total[(Integer)r]));
      createTableColumn(viewer, "Unused", r -> String.valueOf(sel.unused[(Integer)r]));
      createTableColumn(viewer, "BuffCache", r -> String.valueOf(sel.buffCache[(Integer)r]));
    });
  }

  public MemorySelectionView(Composite parent, State state, ProcessMemoryTrack.Values sel) {
    this(parent, sel.ts.length, viewer -> {
      createTableColumn(
          viewer, "Time", r -> timeToString(sel.ts[(Integer)r] - state.getTraceTime().start));
      createTableColumn(viewer, "Dur", r -> timeToString(sel.dur[(Integer)r]));
      createTableColumn(viewer, "File", r -> String.valueOf(sel.file[(Integer)r]));
      createTableColumn(viewer, "Anon", r -> String.valueOf(sel.anon[(Integer)r]));
      createTableColumn(viewer, "Shared", r -> String.valueOf(sel.shared[(Integer)r]));
      createTableColumn(viewer, "Swap", r -> String.valueOf(sel.swap[(Integer)r]));
    });
  }
}
