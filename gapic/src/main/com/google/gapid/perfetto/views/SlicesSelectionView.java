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
import static com.google.gapid.widgets.Widgets.createSelectableLabel;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Iterables;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.SliceTrack;

import com.google.gapid.perfetto.models.SliceTrack.GpuSlices;
import com.google.gapid.perfetto.models.SliceTrack.RenderStageInfo;
import com.google.gapid.perfetto.models.SliceTrack.ThreadSlices;
import com.google.gapid.perfetto.models.ThreadInfo;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected slices.
 */
public class SlicesSelectionView extends Composite {
  private static final int PROPERTIES_PER_PANEL = 8;
  private static final int PANEL_INDENT = 25;

  public SlicesSelectionView(Composite parent, State state, SliceTrack.Slices slices) {
    super(parent, SWT.NONE);
    if (slices.getCount() == 1) {
      setSingleSliceView(state, slices);
    } else if (slices.getCount() > 1) {
      setMultiSlicesView(slices);
    }
  }

  private void setSingleSliceView(State state, SliceTrack.Slices slice) {
    setLayout(withMargin(new GridLayout(2, false), 0, 0));

    Composite main = withLayoutData(createComposite(this, new GridLayout(2, false)),
        new GridData(SWT.LEFT, SWT.TOP, false, false));
    withLayoutData(createBoldLabel(main, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(main, "Time:");
    createLabel(main, timeToString(slice.times.get(0) - state.getTraceTime().start));

    createLabel(main, "Duration:");
    createLabel(main, timeToString(slice.durs.get(0)));

    ThreadInfo thread = ThreadInfo.EMPTY;
    if (slice instanceof ThreadSlices) {
      thread = ((ThreadSlices)slice).getThreadAt(0);
    }
    if (thread != null && thread != ThreadInfo.EMPTY) {
      ProcessInfo process = state.getProcessInfo(thread.upid);
      if (process != null) {
        createLabel(main, "Process:");
        createLabel(main, process.getDisplay());
      }

      createLabel(main, "Thread:");
      createLabel(main, thread.getDisplay());
    }

    if (!slice.categories.get(0).isEmpty()) {
      createLabel(main, "Category:");
      createLabel(main, slice.categories.get(0));
    }

    if (!slice.names.get(0).isEmpty()) {
      createLabel(main, "Name:");
      createLabel(main, slice.names.get(0));
    }

    RenderStageInfo renderStageInfo = RenderStageInfo.EMPTY;
    if (slice instanceof GpuSlices) {
      renderStageInfo = ((GpuSlices)slice).getRenderStageInfoAt(0);
    }
    if (renderStageInfo != null && renderStageInfo != RenderStageInfo.EMPTY) {
      ImmutableMap.Builder<String, String> propsBuilder = ImmutableMap.builder();
      if (renderStageInfo.frameBufferHandle != 0) {
        if (renderStageInfo.frameBufferName.isEmpty()) {
          propsBuilder.put("VkFrameBuffer:", String.format("0x%08X", renderStageInfo.frameBufferHandle));
        } else {
          propsBuilder.put("VkFrameBuffer:", renderStageInfo.frameBufferName + " <" + String.format("0x%08X", renderStageInfo.frameBufferHandle) + ">");
        }
      }
      if (renderStageInfo.renderPassHandle != 0) {
        if (renderStageInfo.renderPassName.isEmpty()) {
          propsBuilder.put("VkRenderPass:", String.format("0x%08X", renderStageInfo.renderPassHandle));
        } else {
          propsBuilder.put("VkRenderPass:", renderStageInfo.renderPassName + " <" + String.format("0x%08X", renderStageInfo.renderPassHandle) + ">");
        }
      }
      if (renderStageInfo.commandBufferHandle != 0) {
        if (renderStageInfo.commandBufferName.isEmpty()) {
          propsBuilder.put("VkCommandBuffer:", String.format("0x%08X", renderStageInfo.commandBufferHandle));
        } else {
          propsBuilder.put("VkCommandBuffer:", renderStageInfo.commandBufferName + " <" + String.format("0x%08X", renderStageInfo.commandBufferHandle) + ">");
        }
      }

      if (renderStageInfo.submissionId != 0) {
        propsBuilder.put("SubmissionId:", Long.toString(renderStageInfo.submissionId));
      }

      ImmutableMap<String, String> props = propsBuilder.build();
      if (!props.isEmpty()) {
        withLayoutData(createBoldLabel(main, "Vulkan Info:"),
            withSpans(new GridData(), 2, 1));
        props.forEach((key, value) -> {
          createSelectableLabel(main, key);
          createSelectableLabel(main, value);
        });
      }
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
          withLayoutData(createSelectableLabel(props, keys[i + c * PROPERTIES_PER_PANEL] + ":"),
              withIndents(new GridData(), (c == 0) ? 0 : PANEL_INDENT, 0));
          createSelectableLabel(props, String.valueOf(slice.argsets.get(0).get(keys[i + c * PROPERTIES_PER_PANEL])));
        }
        if (cols != panels) {
          withLayoutData(createLabel(props, ""), withSpans(new GridData(), 2 * (panels - cols), 1));
        }
      }
    }
  }

  private void setMultiSlicesView(SliceTrack.Slices slices) {
    setLayout(new FillLayout());

    SliceTrack.Node[] nodes = SliceTrack.organizeSlicesToNodes(slices);

    TreeViewer viewer = createTreeViewer(this, SWT.NONE);
    viewer.getTree().setHeaderVisible(true);
    viewer.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return nodes;
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
    viewer.setInput(nodes);
    packColumns(viewer.getTree());
  }

  private static SliceTrack.Node n(Object o) {
    return (SliceTrack.Node)o;
  }
}
