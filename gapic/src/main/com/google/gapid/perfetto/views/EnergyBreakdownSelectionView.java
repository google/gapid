/*
 * Copyright (C) 2022 Google Inc.
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

import com.google.gapid.perfetto.models.EnergyBreakdownTrack;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

public class EnergyBreakdownSelectionView extends Composite {
  public EnergyBreakdownSelectionView(
      Composite parent, State state, EnergyBreakdownTrack.Values sel) {
    super(parent, SWT.None);
    setLayout(new FillLayout());

    TableViewer viewer = createTableViewer(this, SWT.NONE);
    viewer.setContentProvider(new ArrayContentProvider());
    viewer.setLabelProvider(new LabelProvider());

    createTableColumn(
        viewer, "Time", r -> timeToString(sel.ts[(Integer) r] - state.getTraceTime().start));

    for (String name : sel.values.keySet()) {
      createTableColumn(
          viewer,
          name + " (Total energy in uws)",
          r -> {
            if (sel.ids.get(name)[(Integer) r] == -1) {
              return "/";
            }
            return String.valueOf(sel.values.get(name)[(Integer) r]);
          });
    }

    for (Long uid : sel.uidValues.keySet()) {
      createTableColumn(
          viewer,
          "UID: " + String.valueOf(uid) + " (uws)",
          r -> {
            if (sel.uidValues.get(uid) == null) {
              return "";
            }
            return String.valueOf(sel.uidValues.get(uid)[(Integer) r]);
          });
    }

    Integer[] rows = new Integer[sel.ts.length - 1 >= 0 ? sel.ts.length - 1 : 0];
    for (int i = 0; i < rows.length; i++) {
      rows[i] = i;
    }
    viewer.setInput(rows);
    packColumns(viewer.getTable());
  }
}
