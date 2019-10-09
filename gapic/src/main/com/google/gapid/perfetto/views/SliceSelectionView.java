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

import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.SliceTrack;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a selected slice.
 */
public class SliceSelectionView extends Composite {
  public SliceSelectionView(Composite parent, State state, SliceTrack.Slice slice) {
    super(parent, SWT.NONE);
    setLayout(new GridLayout(2, false));

    withLayoutData(createBoldLabel(this, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(this, "Time:");
    createLabel(this, timeToString(slice.time - state.getTraceTime().start));

    createLabel(this, "Duration:");
    createLabel(this, timeToString(slice.dur));

    ThreadInfo thread = slice.getThread();
    if (thread != null) {
      ProcessInfo process = state.getProcessInfo(thread.upid);
      if (process != null) {
        createLabel(this, "Process:");
        createLabel(this, process.getDisplay());
      }

      createLabel(this, "Thread:");
      createLabel(this, thread.getDisplay());
    }

    if (!slice.category.isEmpty()) {
      createLabel(this, "Category:");
      createLabel(this, slice.category);
    }

    if (!slice.name.isEmpty()) {
      createLabel(this, "Name:");
      createLabel(this, slice.name);
    }
  }
}
