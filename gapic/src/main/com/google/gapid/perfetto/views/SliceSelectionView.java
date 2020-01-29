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
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Iterables;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.SliceTrack;
import com.google.gapid.perfetto.models.SliceTrack.RenderStageInfo;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a selected slice.
 */
public class SliceSelectionView extends Composite {
  private static final int PROPERTIES_PER_PANEL = 8;
  private static final int PANEL_INDENT = 25;

  public SliceSelectionView(Composite parent, State state, SliceTrack.Slice slice) {
    super(parent, SWT.NONE);
    setLayout(withMargin(new GridLayout(2, false), 0, 0));

    Composite main = withLayoutData(createComposite(this, new GridLayout(2, false)),
        new GridData(SWT.LEFT, SWT.TOP, false, false));
    withLayoutData(createBoldLabel(main, "Slice:"), withSpans(new GridData(), 2, 1));

    createLabel(main, "Time:");
    createLabel(main, timeToString(slice.time - state.getTraceTime().start));

    createLabel(main, "Duration:");
    createLabel(main, timeToString(slice.dur));

    ThreadInfo thread = slice.getThread();
    if (thread != null) {
      ProcessInfo process = state.getProcessInfo(thread.upid);
      if (process != null) {
        createLabel(main, "Process:");
        createLabel(main, process.getDisplay());
      }

      createLabel(main, "Thread:");
      createLabel(main, thread.getDisplay());
    }

    if (!slice.category.isEmpty()) {
      createLabel(main, "Category:");
      createLabel(main, slice.category);
    }

    if (!slice.name.isEmpty()) {
      createLabel(main, "Name:");
      createLabel(main, slice.name);
    }

    RenderStageInfo renderStageInfo = slice.getRenderStageInfo();
    if (renderStageInfo != null) {
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

      ImmutableMap<String, String> props = propsBuilder.build();
      if (!props.isEmpty()) {
        withLayoutData(createBoldLabel(main, "Vulkan Info"),
            withSpans(new GridData(), 2, 1));
        props.forEach((key, value) -> {
          createLabel(main, key);
          createLabel(main, value);
        });
      }
    }

    if (!slice.args.isEmpty()) {
      String[] keys = Iterables.toArray(slice.args.keys(), String.class);
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
          createLabel(props, String.valueOf(slice.args.get(keys[i + c * PROPERTIES_PER_PANEL])));
        }
        if (cols != panels) {
          withLayoutData(createLabel(props, ""), withSpans(new GridData(), 2 * (panels - cols), 1));
        }
      }
    }
  }
}
