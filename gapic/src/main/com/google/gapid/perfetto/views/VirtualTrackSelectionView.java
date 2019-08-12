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

import com.google.gapid.perfetto.models.VirtualTrack;
import java.util.Map;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

public class VirtualTrackSelectionView extends Composite {
  public VirtualTrackSelectionView(
      Composite parent, State state, VirtualTrack.Slice slice) {
    super(parent, SWT.NONE);
    setLayout(new GridLayout(2, false));

    withLayoutData(createBoldLabel(this, "Track slice:"), withSpans(new GridData(), 2, 1));

    createLabel(this, "Slice id:");
    createLabel(this, Long.toString(slice.sliceId));

    createLabel(this, "Time:");
    createLabel(this, timeToString(slice.ts - state.getTraceTime().start));

    createLabel(this, "Duration:");
    createLabel(this, timeToString(slice.dur));

    createLabel(this, "Slice name:");
    createLabel(this, slice.eventName);

    createBoldLabel(this, "\nArgs:");
    createLabel(this, "");

    for (Map.Entry<String, String> entry : slice.stringValues.entrySet()) {
      createLabel(this, entry.getKey());
      createLabel(this, entry.getValue());
    }

    for (Map.Entry<String, Long> entry : slice.intValues.entrySet()) {
      createLabel(this, entry.getKey());
      createLabel(this, Long.toString(entry.getValue()));
    }
  }
}
