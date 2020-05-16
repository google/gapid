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
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.collect.Iterables;
import com.google.gapid.perfetto.models.FrameEventsTrack;

import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.stream.Collectors;

/**
 * Displays information about a selected Frame event.
 */
public class FrameEventsSelectionView extends Composite {
  private static final int PROPERTIES_PER_PANEL = 8;
  private static final int PANEL_INDENT = 25;

  public FrameEventsSelectionView(Composite parent, State state, FrameEventsTrack.Slices slices) {
    super(parent, SWT.NONE);
    if (slices.getCount() == 1) {
      setSingleSliceView(state, slices);
    } else if (slices.getCount() > 1) {
      setMultiSlicesView(slices);
    }
  }

  private void setSingleSliceView(State state, FrameEventsTrack.Slices slice) {
    setLayout(withMargin(new GridLayout(3, false), 0, 0));

    Composite main = withLayoutData(createComposite(this, new GridLayout(2, false)),
        new GridData(SWT.LEFT, SWT.TOP, false, false));
    withLayoutData(createBoldLabel(main, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(main, "Name:");
    createLabel(main, slice.names.get(0));

    createLabel(main, "Time:");
    createLabel(main, timeToString(slice.times.get(0) - state.getTraceTime().start));

    createLabel(main, "Duration:");
    createLabel(main, timeToString(slice.durs.get(0)));

    if (slice.frameStats.get(0) != null) {
      // If the selected event is a displayed frame slice, show the frame stats
      Composite stats = withLayoutData(createComposite(this, new GridLayout(2, false)),
          withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), PANEL_INDENT, 0));
      withLayoutData(createBoldLabel(stats, "Frame Stats:"),
          withSpans(new GridData(), 2, 1));

      slice.frameStats.get(0).forEach((k, v) -> {
        withLayoutData(createBoldLabel(stats, k.toString()),
            withSpans(new GridData(), 2, 1));

        createLabel(stats, "Frame number: ");
        createLabel(stats, Long.toString(v.frameNumber));

        createLabel(stats, "Queue to Acquire: ");
        createLabel(stats, timeToString(v.queueToAcquireTime));

        createLabel(stats, "Acquire to Latch: ");
        createLabel(stats, timeToString(v.acquireToLatchTime));

        createLabel(stats, "Latch to Present: ");
        createLabel(stats, timeToString(v.latchToPresentTime));
      });
    } else {
      // Show the frame number associated with the event
      createLabel(main, "Frame Number:");
      createLabel(main, Arrays.stream(slice.frameNumbers.get(0))
          .map(l -> Long.toString(l))
          .collect(Collectors.joining(", ")));
    }

    if (!slice.argsets.get(0).isEmpty()) {
      String[] keys = Iterables.toArray(slice.argsets.get(0).keys(), String.class);
      int panels = (keys.length + PROPERTIES_PER_PANEL - 1) / PROPERTIES_PER_PANEL;
      Composite props = withLayoutData(createComposite(this, new GridLayout(2 * panels, false)),
          withIndents(new GridData(SWT.LEFT, SWT.TOP, false, false), PANEL_INDENT, 0));
      withLayoutData(createBoldLabel(props, "Properties:"),
          withSpans(new GridData(), 2 * panels, 1));

      for (int i = 0; i < keys.length && i < PROPERTIES_PER_PANEL; i++) {
        int cols = (keys.length - i + PROPERTIES_PER_PANEL - 1) / PROPERTIES_PER_PANEL;
        for (int c = 0; c < cols; c++) {
          withLayoutData(createLabel(props, keys[i + c * PROPERTIES_PER_PANEL] + ":"),
              withIndents(new GridData(), (c == 0) ? 0 : PANEL_INDENT, 0));
          createLabel(props, String.valueOf(slice.argsets.get(0).get(keys[i + c * PROPERTIES_PER_PANEL])));
        }
        if (cols != panels) {
          withLayoutData(createLabel(props, ""), withSpans(new GridData(), 2 * (panels - cols), 1));
        }
      }
    }
  }

  private void setMultiSlicesView(FrameEventsTrack.Slices slices) {
    setLayout(new FillLayout());

    FrameEventsTrack.Node[] nodes = FrameEventsTrack.organizeSlicesToNodes(slices);

    TreeViewer viewer = createTreeViewer(this, SWT.NONE);
    viewer.getTree().setHeaderVisible(true);
    viewer.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return nodes;
      }

      @Override
      public boolean hasChildren(Object element) {
        return false;
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }

      @Override
      public Object[] getChildren(Object element) {
        return null;
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTreeColumn(viewer, "Name", e -> n(e).name);
    createTreeColumn(viewer, "Self Time", e -> timeToString(n(e).self));
    createTreeColumn(viewer, "Layers", e -> String.join(", ", n(e).layers));
    viewer.setInput(slices);
    packColumns(viewer.getTree());
  }

  private static FrameEventsTrack.Node n(Object o) {
    return (FrameEventsTrack.Node)o;
  }
}
