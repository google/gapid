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

import static com.google.gapid.perfetto.views.StyleConstants.colors;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Fonts;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.Track;

/**
 * Utility functions for drawing the loading indicators.
 */
public class Loading {
  private static final double MARGIN = 3;

  public static void drawLoading(RenderContext ctx, Track.Data data, State state, double h) {
    if (h < 3 * MARGIN) {
      return;
    }

    TimeSpan visible = state.getVisibleTime();
    TimeSpan available = (data == null) ? TimeSpan.ZERO : data.request.range;
    if (available.end <= visible.start || available.start >= visible.end) {
      drawLoading(ctx, state.timeToPx(visible.start), state.timeToPx(visible.end), h);
      return;
    }

    if (available.start > visible.start) {
      drawLoading(ctx, state.timeToPx(visible.start), state.timeToPx(available.start), h);
    }

    if (available.end < visible.end) {
      drawLoading(ctx, state.timeToPx(available.end), state.timeToPx(visible.end), h);
    }
  }

  public static void drawLoading(RenderContext ctx, double x1, double x2, double h) {
    double w = x2 - x1;
    ctx.setBackgroundColor(colors().loadingBackground);
    ctx.fillRect(x1, MARGIN, w, h - 2 * MARGIN);
    ctx.setForegroundColor(colors().loadingForeground);
    ctx.drawTextIfFits(
        Fonts.Style.Normal, "Loading...", x1 + MARGIN, MARGIN, w - 2 * MARGIN, h - 2 * MARGIN);
  }
}
