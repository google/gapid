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

import static com.google.gapid.util.Colors.hsl;
import static com.google.gapid.util.Colors.rgb;
import static com.google.gapid.util.Colors.rgba;

import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.RGBA;

/**
 * Constants governing the look of the UI.
 */
public class StyleConstants {
  public static final double TITLE_HEIGHT = 25;
  public static final double LABEL_OFFSET = 20;
  public static final double LABEL_ICON_SIZE = 16;
  public static final double LABEL_WIDTH = 280;
  public static final double LABEL_MARGIN = 4;
  public static final double LABEL_PIN_X = LABEL_WIDTH - LABEL_MARGIN - LABEL_ICON_SIZE;
  public static final double LABEL_TOGGLE_X = LABEL_PIN_X - LABEL_ICON_SIZE;
  public static final double TRACK_MARGIN = 4;
  public static final double DEFAULT_COUNTER_TRACK_HEIGHT = 45;
  public static final double PROCESS_COUNTER_TRACK_HEIGHT = 30;
  public static final double POWER_RAIL_COUNTER_TRACK_HEIGHT = 30;
  public static final double HIGHLIGHT_EDGE_NEARBY_WIDTH = 10;
  public static final double SELECTION_THRESHOLD = 0.333;
  public static final double ZOOM_FACTOR_SCALE = 0.05;
  public static final double ZOOM_FACTOR_SCALE_DRAG = 0.01;

  // Keyboard handling constants.
  public static final int KB_DELAY = 20;
  public static final int KB_PAN_SLOW = 30;
  public static final int KB_PAN_FAST = 60;
  public static final double KB_ZOOM_SLOW = 2 * ZOOM_FACTOR_SCALE;
  public static final double KB_ZOOM_FAST = 3 * ZOOM_FACTOR_SCALE;

  // Touchpad handling constants.
  public static final int TP_PAN_SLOW = 30;
  public static final int TP_PAN_FAST = 60;

  public static class Colors {
    public final RGBA background;
    public final RGBA titleBackground;
    public final RGBA gridline;
    public final RGBA panelBorder;
    public final RGBA hoverBackground;
    public final RGBA loadingBackground;
    public final RGBA loadingForeground;
    public final RGBA selectionBackground;
    public final RGBA timeHighlight;
    public final RGBA timeHighlightBorder;
    public final RGBA timeHighlightCover;
    public final RGBA timeHighlightEmphasize;
    public final RGBA cpuFreqIdle;
    public final RGBA timelineRuler;
    public final RGBA vsyncBackground;
    public final RGBA flagLine;
    public final RGBA flagHover;

    public final RGBA textMain;
    public final RGBA textAlt;

    public Colors(RGBA background,
        RGBA titleBackground,
        RGBA gridline,
        RGBA panelBorder,
        RGBA hoverBackground,
        RGBA loadingBackground,
        RGBA loadingForeground,
        RGBA selectionBackground,
        RGBA timeHighlight,
        RGBA timeHighlightBorder,
        RGBA timeHighlightCover,
        RGBA timeHighlightEmphasize,
        RGBA cpuFreqIdle,
        RGBA timelineRuler,
        RGBA vsyncBackground,
        RGBA flagLine,
        RGBA flagHover,
        RGBA textMain,
        RGBA textAlt) {
      this.background = background;
      this.titleBackground = titleBackground;
      this.gridline = gridline;
      this.panelBorder = panelBorder;
      this.hoverBackground = hoverBackground;
      this.loadingBackground = loadingBackground;
      this.loadingForeground = loadingForeground;
      this.selectionBackground = selectionBackground;
      this.timeHighlight = timeHighlight;
      this.timeHighlightBorder = timeHighlightBorder;
      this.timeHighlightCover = timeHighlightCover;
      this.timeHighlightEmphasize = timeHighlightEmphasize;
      this.cpuFreqIdle = cpuFreqIdle;
      this.timelineRuler = timelineRuler;
      this.vsyncBackground = vsyncBackground;
      this.flagLine = flagLine;
      this.flagHover = flagHover;
      this.textMain = textMain;
      this.textAlt = textAlt;
    }

    private static final RGBA LIGHT_BACKGROUND = rgb(0xff, 0xff, 0xff);
    private static final RGBA LIGHT_TITLE_BACKGROUND = rgb(0xe9, 0xe9, 0xe9);
    private static final RGBA LIGHT_GRIDLINE = rgb(0xda, 0xda, 0xda);
    private static final RGBA LIGHT_PANEL_BORDER = LIGHT_GRIDLINE;
    private static final RGBA LIGHT_HOVER_BACKGROUND = rgba(0xf7, 0xf7, 0xf7, 0.95f);
    private static final RGBA LIGHT_LOADING_BACKGROUND = rgb(0xe0, 0xe6, 0xe8);
    private static final RGBA LIGHT_LOADING_FOREGROUND = rgb(0x66, 0x66, 0x66);
    private static final RGBA LIGHT_SELECTION_BACKGROUND = rgba(0, 0, 255, 0.3f);
    private static final RGBA LIGHT_TIME_HIGHLIGHT = rgb(0x32, 0x34, 0x35);
    private static final RGBA LIGHT_TIME_HIGHLIGHT_BORDER = LIGHT_GRIDLINE;
    private static final RGBA LIGHT_TIME_HIGHLIGHT_COVER = rgba(0, 0, 0, 0.2f);
    private static final RGBA LIGHT_TIME_HIGHLIGHT_EMPHASIZE = rgb(0xff, 0xde, 0x00);
    private static final RGBA LIGHT_CPU_FREQ_IDLE = rgb(0xf0, 0xf0, 0xf0);
    private static final RGBA LIGHT_TIMELINE_RULER = rgb(0x99, 0x99, 0x99);
    private static final RGBA LIGHT_VSYNC_BACKGROUND = rgb(0xf5, 0xf5, 0xf5);
    private static final RGBA LIGHT_FLAG_LINE = rgb(0, 0, 0);
    private static final RGBA LIGHT_FLAG_HOVER = rgb(0x80, 0x80, 0x80);

    private static final RGBA LIGHT_TEXT_MAIN = rgb(0x32, 0x34, 0x35);
    private static final RGBA LIGHT_TEXT_ALT = rgb(101, 102, 104);

    public static Colors light() {
      return new Colors(
            LIGHT_BACKGROUND,
            LIGHT_TITLE_BACKGROUND,
            LIGHT_GRIDLINE,
            LIGHT_PANEL_BORDER,
            LIGHT_HOVER_BACKGROUND,
            LIGHT_LOADING_BACKGROUND,
            LIGHT_LOADING_FOREGROUND,
            LIGHT_SELECTION_BACKGROUND,
            LIGHT_TIME_HIGHLIGHT,
            LIGHT_TIME_HIGHLIGHT_BORDER,
            LIGHT_TIME_HIGHLIGHT_COVER,
            LIGHT_TIME_HIGHLIGHT_EMPHASIZE,
            LIGHT_CPU_FREQ_IDLE,
            LIGHT_TIMELINE_RULER,
            LIGHT_VSYNC_BACKGROUND,
            LIGHT_FLAG_LINE,
            LIGHT_FLAG_HOVER,
            LIGHT_TEXT_MAIN,
            LIGHT_TEXT_ALT);
    }

    private static final RGBA DARK_BACKGROUND = rgb(0x1a, 0x1a, 0x1a);
    private static final RGBA DARK_TITLE_BACKGROUND = rgb(0x3b, 0x3b, 0x3b);
    private static final RGBA DARK_GRIDLINE = rgb(0x40, 0x40, 0x40);
    private static final RGBA DARK_PANEL_BORDER = DARK_GRIDLINE;
    private static final RGBA DARK_HOVER_BACKGROUND = rgba(0x17, 0x17, 0x17, 0.8f);
    private static final RGBA DARK_LOADING_BACKGROUND = rgb(0x4a, 0x4a, 0x4a);
    private static final RGBA DARK_LOADING_FOREGROUND = rgb(0xaa, 0xaa, 0xaa);
    private static final RGBA DARK_SELECTION_BACKGROUND = rgba(0, 0, 255, 0.5f);
    private static final RGBA DARK_TIME_HIGHLIGHT = rgb(0xff, 0xff, 0xff);
    private static final RGBA DARK_TIME_HIGHLIGHT_BORDER = DARK_GRIDLINE;
    private static final RGBA DARK_TIME_HIGHLIGHT_COVER = rgba(0xff, 0xff, 0xff, 0.2f);
    private static final RGBA DARK_TIME_HIGHLIGHT_EMPHASIZE = rgb(0xd2, 0xb6, 0x00);
    private static final RGBA DARK_CPU_FREQ_IDLE = rgb(0x55, 0x55, 0x55);
    private static final RGBA DARK_TIMELINE_RULER = rgb(0x99, 0x99, 0x99);
    private static final RGBA DARK_VSYNC_BACKGROUND = rgb(0x24, 0x24, 0x24);
    private static final RGBA DARK_FLAG_LINE = rgb(0xff, 0xff, 0xff);
    private static final RGBA DARK_FLAG_HOVER = rgb(0x80, 0x80, 0x80);

    private static final RGBA DARK_TEXT_MAIN = rgb(0xf1, 0xf1, 0xf8);
    private static final RGBA DARK_TEXT_ALT = rgb(0xdd, 0xdd, 0xdd);

    public static Colors dark() {
      return new Colors(
            DARK_BACKGROUND,
            DARK_TITLE_BACKGROUND,
            DARK_GRIDLINE,
            DARK_PANEL_BORDER,
            DARK_HOVER_BACKGROUND,
            DARK_LOADING_BACKGROUND,
            DARK_LOADING_FOREGROUND,
            DARK_SELECTION_BACKGROUND,
            DARK_TIME_HIGHLIGHT,
            DARK_TIME_HIGHLIGHT_BORDER,
            DARK_TIME_HIGHLIGHT_COVER,
            DARK_TIME_HIGHLIGHT_EMPHASIZE,
            DARK_CPU_FREQ_IDLE,
            DARK_TIMELINE_RULER,
            DARK_VSYNC_BACKGROUND,
            DARK_FLAG_LINE,
            DARK_FLAG_HOVER,
            DARK_TEXT_MAIN,
            DARK_TEXT_ALT);
    }
  }

  private static Colors colors = Colors.light();
  private static boolean isDark = false;

  private StyleConstants() {
  }

  public static Colors colors() {
    return colors;
  }

  public static Gradient gradient(int seed) {
    // See Gradients.Colors for explanation of magic constants.
    int idx = ((seed + 8) & 0x7fffffff) % Gradients.COUNT;
    return (isDark ? Gradients.DARK : Gradients.LIGHT)[idx];
  }

  public static Gradient mainGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[14] : Gradients.LIGHT[14];
  }

  public static Gradient threadStateSleeping() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[15] : Gradients.LIGHT[15];
  }

  public static Gradient threadStateRunnable() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[13] : Gradients.LIGHT[13];
  }

  public static Gradient threadStateRunning() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[10] : Gradients.LIGHT[10];
  }

  public static Gradient threadStateBlockedOk() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[1] : Gradients.LIGHT[1];
  }

  public static Gradient threadStateBlockedWarn() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[2] : Gradients.LIGHT[2];
  }

  public static Gradient batteryInGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[10] : Gradients.LIGHT[10];
  }

  public static Gradient batteryOutGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[10] : Gradients.LIGHT[1];
  }

  public static Gradient memoryUsedGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[13] : Gradients.LIGHT[13];
  }

  public static Gradient memoryBuffersGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[14] : Gradients.LIGHT[14];
  }

  public static Gradient memoryRssFileGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[13] : Gradients.LIGHT[16];
  }

  public static Gradient memoryRssAnonGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[14] : Gradients.LIGHT[13];
  }

  public static Gradient memoryRssSharedGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[16] : Gradients.LIGHT[14];
  }

  public static Gradient memorySwapGradient() {
    // See Gradients.Colors for explanation of magic constants.
    return isDark ? Gradients.DARK[2] : Gradients.LIGHT[2];
  }

  public static boolean isLight() {
    return !isDark;
  }

  public static boolean isDark() {
    return isDark;
  }

  public static void setDark(boolean dark) {
    isDark = dark;
    colors = isDark ? Colors.dark() : Colors.light();
  }

  public static void toggleDark() {
    setDark(!isDark);
  }

  public static Image arrowDown(Theme theme) {
    return isDark ? theme.arrowDropDownDark() : theme.arrowDropDownLight();
  }

  public static Image arrowRight(Theme theme) {
    return isDark ? theme.arrowDropRightDark() : theme.arrowDropRightLight();
  }

  public static Image unfoldMore(Theme theme) {
    return isDark ? theme.unfoldMoreDark() : theme.unfoldMoreLight();
  }

  public static Image unfoldLess(Theme theme) {
    return isDark ? theme.unfoldLessDark() : theme.unfoldLessLight();
  }

  public static Image rangeStart(Theme theme) {
    return isDark ? theme.rangeStartDark() : theme.rangeStartLight();
  }

  public static Image rangeEnd(Theme theme) {
    return isDark ? theme.rangeEndDark() : theme.rangeEndLight();
  }

  public static Image pinActive(Theme theme) {
    return isDark ? theme.pinActiveDark() : theme.pinActiveLight();
  }

  public static Image pinInactive(Theme theme) {
    return isDark ? theme.pinInactiveDark() : theme.pinInactiveLight();
  }

  public static Image flag(Theme theme) {
    return isDark ? theme.flagDark() : theme.flagLight();
  }

  public static Image flagFilled(Theme theme) {
    return isDark ? theme.flagFilledDark() : theme.flagFilledLight();
  }

  public static Image flagGreyed(Theme theme) {
    return theme.flagGreyed();
  }

  public static class Gradient {
    private static final float HIGH_TARGET = 0.9f;
    private static final float LOW_TARGET = 0.2f;

    private static final float LIGHT_BASE = 0.3f;
    private static final float LIGHT_BORDER = -0.1f;
    private static final float LIGHT_HIGHLIHGT = -0.5f;
    private static final float DARK_BASE = 0.1f;
    private static final float DARK_BORDER = -0.4f;
    private static final float DARK_HIGHLIGHT = 0.7f;

    public final RGBA base;
    public final RGBA border;
    public final RGBA highlight;
    public final RGBA alternate;
    public final RGBA disabled = hsl(0, 0, 0.62f);

    private final float h, s, l;
    private final float high, low;

    public Gradient(float h, float s, float l, boolean light) {
      this.h = h;
      this.s = s;
      this.l = l;
      this.high = light ? HIGH_TARGET : LOW_TARGET;
      this.low = light ? LOW_TARGET : HIGH_TARGET;

      this.base = lerp(light ? LIGHT_BASE : DARK_BASE);
      this.border = lerp(light ? LIGHT_BORDER : DARK_BORDER);
      this.highlight = lerp(light ? LIGHT_HIGHLIHGT : DARK_HIGHLIGHT);
      this.alternate = lerp(1);
    }

    /**
     * @param x interpolation multiplier in the range [-1, 1].
     */
    public RGBA lerp(float x) {
      if (x < 0) {
        return hsl(h, s, l - x * (low - l));
      } else {
        return hsl(h, s, l + x * (high - l));
      }
    }

    /** Sets fill to base. **/
    public void applyBase(RenderContext ctx) {
      ctx.setBackgroundColor(base);
    }

    /** Sets fill to base, and stroke to border. **/
    public void applyBaseAndBorder(RenderContext ctx) {
      ctx.setForegroundColor(border);
      ctx.setBackgroundColor(base);
    }

    /** Sets stroke to border. **/
    public void applyBorder(RenderContext ctx) {
      ctx.setForegroundColor(border);
    }
  }

  private static class Gradients {
    // Order here matters, when changing, adjust the indices above.
    private static final float[][] COLORS = {
        {  15.38f, 0.2633f, 0.6000f }, // Brown
        {  20.15f, 0.8954f, 0.7000f }, // Orange         // blocked OK, battery out
        {  21.92f, 0.7626f, 0.4294f }, // Dark Orange    // blocked warn, swap
        {  36.22f, 0.7312f, 0.5000f }, // Light Brown
        {  42.19f, 0.4412f, 0.7490f }, // Tan
        {  50.00f, 0.5822f, 0.5844f }, // Gold
        {  59.49f, 0.5109f, 0.4490f }, // Lime
        {  66.00f, 0.8233f, 0.5400f }, // Apple Green
        {  88.00f, 0.5000f, 0.5300f }, // Chartreuse
        { 122.00f, 0.3900f, 0.4900f }, // Dark Green
        { 130.91f, 0.6548f, 0.6706f }, // Green          // running, battery in
        { 171.02f, 0.5787f, 0.5598f }, // Turquoise
        { 172.36f, 0.7432f, 0.2902f }, // Teal
        { 198.40f, 1.0000f, 0.4157f }, // Pacific Blue   // runnable, mem used, rss anon
        { 200.22f, 0.9787f, 0.8157f }, // Light Blue     // main, mem buf/cache, rss shared
        { 201.95f, 0.2455f, 0.6725f }, // Grey           // sleeping
        { 214.85f, 1.0000f, 0.5510f }, // Vivid Blue     // rss file
        { 217.06f, 0.5000f, 0.4000f }, // Indigo
        { 261.54f, 0.5065f, 0.6980f }, // Light Purple
        { 262.30f, 0.6981f, 0.5843f }, // Purple
        { 298.56f, 0.5540f, 0.6845f }, // Light Magenta
        { 319.69f, 0.5333f, 0.4706f }, // Magenta
        { 338.32f, 0.7041f, 0.6686f }, // Pink
    };

    public static final Gradient[] LIGHT;
    public static final Gradient[] DARK;
    public static final int COUNT = COLORS.length;

    static {
      LIGHT = new Gradient[COUNT];
      DARK = new Gradient[COUNT];

      for (int i = 0; i < COUNT; i++) {
        float h = COLORS[i][0], s = COLORS[i][1], l = COLORS[i][2];
        LIGHT[i] = new Gradient(h, s, l, true);
        DARK[i] = new Gradient(h, s, l, false);
      }
    }
  }
}
