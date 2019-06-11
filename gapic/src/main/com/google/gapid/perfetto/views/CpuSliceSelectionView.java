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
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a selected CPU slice.
 */
public class CpuSliceSelectionView extends Composite {
  public CpuSliceSelectionView(Composite parent, State state, CpuTrack.Slice slice) {
    super(parent, SWT.NONE);
    setLayout(new GridLayout(2, false));

    withLayoutData(createBoldLabel(this, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(this, "Time:");
    createLabel(this, timeToString(slice.time - state.getTraceTime().start));

    createLabel(this, "Duration:");
    createLabel(this, timeToString(slice.dur));

    ThreadInfo thread = state.getThreadInfo(slice.utid);
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
    createLabel(this, ThreadState.RUNNING.label);

    createLabel(this, "CPU:");
    createLabel(this, Integer.toString(slice.cpu + 1));

    createLabel(this, "End State:");
    createLabel(this, slice.endState.label);

    createLabel(this, "Priority:");
    createLabel(this, Integer.toString(slice.priority));
  }
}
