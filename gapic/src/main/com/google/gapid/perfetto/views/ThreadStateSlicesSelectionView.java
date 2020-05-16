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
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.perfetto.models.ThreadTrack;

import org.eclipse.jface.viewers.IStructuredContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected thread state slices.
 */
public class ThreadStateSlicesSelectionView extends Composite {
  public ThreadStateSlicesSelectionView(
      Composite parent, State state, ThreadTrack.StateSlices slices) {
    super(parent, SWT.NONE);
    if (slices.getCount() == 1) {
      setSingleSliceView(state, slices);
    } else if (slices.getCount() > 1) {
      setMultiSlicesView(slices);
    }
  }

  private void setSingleSliceView(State state, ThreadTrack.StateSlices slice) {
    setLayout(new GridLayout(2, false));
    withLayoutData(createBoldLabel(this, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(this, "Time:");
    createLabel(this, timeToString(slice.times.get(0) - state.getTraceTime().start));

    createLabel(this, "Duration:");
    createLabel(this, timeToString(slice.durs.get(0)));

    ThreadInfo thread = state.getThreadInfo(slice.utids.get(0));
    if (thread != null) {
      ProcessInfo process = state.getProcessInfo(thread.upid);
      if (process != null) {
        createLabel(this, "Process:");
        createLabel(this, process.getDisplay());
      }

      createLabel(this, "Thread:");
      createLabel(this, thread.getDisplay());
    }

    createLabel(this, "State:");
    createLabel(this, slice.states.get(0).label);
  }

  private void setMultiSlicesView(ThreadTrack.StateSlices slices) {
    setLayout(new FillLayout());

    ThreadTrack.Entry[] entries = ThreadTrack.organizeSlicesToEntry(slices);

    TableViewer viewer = createTableViewer(this, SWT.NONE);
    viewer.setContentProvider(new IStructuredContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return entries;
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTableColumn(viewer, "State",
        e -> ((ThreadTrack.Entry)e).state.label);
    createTableColumn(viewer, "Duration",
        e -> timeToString(((ThreadTrack.Entry)e).totalDur));
    viewer.setInput(entries);
    packColumns(viewer.getTable());
  }
}
