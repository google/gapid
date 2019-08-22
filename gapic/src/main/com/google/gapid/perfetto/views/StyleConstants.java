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

import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.widgets.Theme;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.RGBA;

/**
 * Constants governing the look of the UI.
 */
public class StyleConstants {
  public static final double TITLE_HEIGHT = 25;
  public static final double LABEL_OFFSET = 20;
  public static final double ICON_SIZE = 24;
  public static final double TOGGLE_ICON_OFFSET = 30;
  public static final double LABEL_WIDTH = 250;
  public static final double TRACK_MARGIN = 4;
  public static final double SELECTION_THRESHOLD = 0.333;
  public static final double ZOOM_FACTOR_SCALE = 0.05;

  // Keyboard handling constants.
  public static final int KB_DELAY = 20;
  public static final int KB_PAN_SLOW = 30;
  public static final int KB_PAN_FAST = 60;
  public static final double KB_ZOOM_SLOW = 2 * ZOOM_FACTOR_SCALE;
  public static final double KB_ZOOM_FAST = 3 * ZOOM_FACTOR_SCALE;

  public static class Colors {
    public final int background;
    public final RGBA titleBackground;
    public final RGBA gridline;
    public final RGBA panelBorder;
    public final RGBA hoverBackground;
    public final RGBA loadingBackground;
    public final RGBA loadingForeground;
    public final RGBA selectionBackground;
    public final RGBA cpuUsageFill;
    public final RGBA cpuUsageStroke;
    public final RGBA cpuFreqIdle;
    public final RGBA timelineRuler;
    public final RGBA counterFill;
    public final RGBA counterStroke;

    public final RGBA textMain;
    public final RGBA textAlt;
    public final RGBA textInvertedMain;
    public final RGBA textInvertedAlt;

    public final RGBA threadStateRunning;
    public final RGBA threadStateRunnable;
    public final RGBA threadStateIowait;
    public final RGBA threadStateUninterruptile;
    public final RGBA threadStateSleep;
    public final RGBA threadStateUnknown;

    public final RGBA memoryBufferedCached;
    public final RGBA memoryUsed;

    public Colors(int background,
        RGBA titleBackground,
        RGBA gridline,
        RGBA panelBorder,
        RGBA hoverBackground,
        RGBA loadingBackground,
        RGBA loadingForeground,
        RGBA selectionBackground,
        RGBA cpuUsageFill,
        RGBA cpuUsageStroke,
        RGBA cpuFreqIdle,
        RGBA timelineRuler,
        RGBA counterFill,
        RGBA counterStroke,
        RGBA textMain,
        RGBA textAlt,
        RGBA textInvertedMain,
        RGBA textInvertedAlt,
        RGBA threadStateRunning,
        RGBA threadStateRunnable,
        RGBA threadStateIowait,
        RGBA threadStateUninterruptile,
        RGBA threadStateSleep,
        RGBA threadStateUnknown,
        RGBA memoryBufferedCached,
        RGBA memoryUsed) {
      this.background = background;
      this.titleBackground = titleBackground;
      this.gridline = gridline;
      this.panelBorder = panelBorder;
      this.hoverBackground = hoverBackground;
      this.loadingBackground = loadingBackground;
      this.loadingForeground = loadingForeground;
      this.selectionBackground = selectionBackground;
      this.cpuUsageFill = cpuUsageFill;
      this.cpuUsageStroke = cpuUsageStroke;
      this.cpuFreqIdle = cpuFreqIdle;
      this.timelineRuler = timelineRuler;
      this.counterFill = counterFill;
      this.counterStroke = counterStroke;
      this.textMain = textMain;
      this.textAlt = textAlt;
      this.textInvertedMain = textInvertedMain;
      this.textInvertedAlt = textInvertedAlt;
      this.threadStateRunning = threadStateRunning;
      this.threadStateRunnable = threadStateRunnable;
      this.threadStateIowait = threadStateIowait;
      this.threadStateUninterruptile = threadStateUninterruptile;
      this.threadStateSleep = threadStateSleep;
      this.threadStateUnknown = threadStateUnknown;
      this.memoryBufferedCached = memoryBufferedCached;
      this.memoryUsed = memoryUsed;
    }

    private static final int LIGHT_BACKGROUND = SWT.COLOR_WHITE;
    private static final RGBA LIGHT_TITLE_BACKGROUND = rgb(0xf7, 0xf7, 0xf7);
    private static final RGBA LIGHT_GRIDLINE = rgb(0xda, 0xda, 0xda);
    private static final RGBA LIGHT_PANEL_BORDER = LIGHT_GRIDLINE;
    private static final RGBA LIGHT_HOVER_BACKGROUND = rgba(0xf7, 0xf7, 0xf7, 0.9f);
    private static final RGBA LIGHT_LOADING_BACKGROUND = rgb(0xee, 0xee, 0xee);
    private static final RGBA LIGHT_LOADING_FOREGROUND = rgb(0x66, 0x66, 0x66);
    private static final RGBA LIGHT_SELECTION_BACKGROUND = rgba(0, 0, 255, 0.3f);
    private static final RGBA LIGHT_CPU_USAGE_FILL = rgb(0x00, 0xB8, 0xD4);
    private static final RGBA LIGHT_CPU_USAGE_STROKE = rgb(0x0D, 0x9A, 0xA8);
    private static final RGBA LIGHT_CPU_FREQ_IDLE = rgb(240, 240, 240);
    private static final RGBA LIGHT_TIMELINE_RULER = rgb(0x99, 0x99, 0x99);
    private static final RGBA LIGHT_COUNTER_FILL = LIGHT_CPU_USAGE_FILL;
    private static final RGBA LIGHT_COUNTER_STROKE = LIGHT_CPU_USAGE_STROKE;

    private static final RGBA LIGHT_TEXT_MAIN = rgb(0x32, 0x34, 0x35);
    private static final RGBA LIGHT_TEXT_ALT = rgb(101, 102, 104);
    private static final RGBA LIGHT_TEXT_INVERTED_MAIN = rgb(0xff, 0xff, 0xff);
    private static final RGBA LIGHT_TEXT_INVERTED_ALT = rgb(0xdd, 0xdd, 0xdd);

    private static final RGBA LIGHT_THREAD_STATE_RUNNING = LIGHT_CPU_USAGE_FILL;
    private static final RGBA LIGHT_THREAD_STATE_RUNNABLE = rgb(126, 200, 148);
    private static final RGBA LIGHT_THREAD_STATE_IOWAIT = rgb(255, 140, 0);
    private static final RGBA LIGHT_THREAD_STATE_UNINTERRUPTILE = rgb(182, 125, 143);
    private static final RGBA LIGHT_THREAD_STATE_SLEEP = rgb(240, 240, 240);
    private static final RGBA LIGHT_THREAD_STATE_UNKNOWN = rgb(199, 155, 125);

    private static final RGBA LIGHT_MEMORY_BUFFERED_CACHED = rgb(0x76, 0xD2, 0xff);
    private static final RGBA LIGHT_MEMORY_USED = rgb(0x34, 0x65, 0xA4);

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
            LIGHT_CPU_USAGE_FILL,
            LIGHT_CPU_USAGE_STROKE,
            LIGHT_CPU_FREQ_IDLE,
            LIGHT_TIMELINE_RULER,
            LIGHT_COUNTER_FILL,
            LIGHT_COUNTER_STROKE,
            LIGHT_TEXT_MAIN,
            LIGHT_TEXT_ALT,
            LIGHT_TEXT_INVERTED_MAIN,
            LIGHT_TEXT_INVERTED_ALT,
            LIGHT_THREAD_STATE_RUNNING,
            LIGHT_THREAD_STATE_RUNNABLE,
            LIGHT_THREAD_STATE_IOWAIT,
            LIGHT_THREAD_STATE_UNINTERRUPTILE,
            LIGHT_THREAD_STATE_SLEEP,
            LIGHT_THREAD_STATE_UNKNOWN,
            LIGHT_MEMORY_BUFFERED_CACHED,
            LIGHT_MEMORY_USED);
    }

    private static final int DARK_BACKGROUND = SWT.COLOR_BLACK;
    private static final RGBA DARK_TITLE_BACKGROUND = rgb(0x25, 0x25, 0x25);
    private static final RGBA DARK_GRIDLINE = rgb(0x40, 0x40, 0x40);
    private static final RGBA DARK_PANEL_BORDER = DARK_GRIDLINE;
    private static final RGBA DARK_HOVER_BACKGROUND = rgba(0x17, 0x17, 0x17, 0.7f);
    private static final RGBA DARK_LOADING_BACKGROUND = rgb(0x25, 0x25, 0x25);
    private static final RGBA DARK_LOADING_FOREGROUND = rgb(0xAA, 0xAA, 0xAA);
    private static final RGBA DARK_SELECTION_BACKGROUND = rgba(0, 0, 255, 0.5f);
    private static final RGBA DARK_CPU_USAGE_FILL = rgb(0x00, 0xB8, 0xD4);
    private static final RGBA DARK_CPU_USAGE_STROKE = rgb(0x0D, 0x9A, 0xA8);
    private static final RGBA DARK_CPU_FREQ_IDLE = rgb(240, 240, 240);
    private static final RGBA DARK_TIMELINE_RULER = rgb(0x99, 0x99, 0x99);
    private static final RGBA DARK_COUNTER_FILL = DARK_CPU_USAGE_FILL;
    private static final RGBA DARK_COUNTER_STROKE = DARK_CPU_USAGE_STROKE;

    private static final RGBA DARK_TEXT_MAIN = rgb(0xff, 0xff, 0xff);
    private static final RGBA DARK_TEXT_ALT = rgb(0xdd, 0xdd, 0xdd);
    private static final RGBA DARK_TEXT_INVERTED_MAIN = rgb(0x19, 0x1A, 0x19);
    private static final RGBA DARK_TEXT_INVERTED_ALT = rgb(50, 51, 52);

    private static final RGBA DARK_THREAD_STATE_RUNNING = DARK_CPU_USAGE_FILL;
    private static final RGBA DARK_THREAD_STATE_RUNNABLE = rgb(126, 200, 148);
    private static final RGBA DARK_THREAD_STATE_IOWAIT = rgb(255, 140, 0);
    private static final RGBA DARK_THREAD_STATE_UNINTERRUPTILE = rgb(182, 125, 143);
    private static final RGBA DARK_THREAD_STATE_SLEEP = rgb(240, 240, 240);
    private static final RGBA DARK_THREAD_STATE_UNKNOWN = rgb(199, 155, 125);

    private static final RGBA DARK_MEMORY_BUFFERED_CACHED = rgb(0x76, 0xD2, 0xff);
    private static final RGBA DARK_MEMORY_USED = rgb(0x34, 0x65, 0xA4);

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
            DARK_CPU_USAGE_FILL,
            DARK_CPU_USAGE_STROKE,
            DARK_CPU_FREQ_IDLE,
            DARK_TIMELINE_RULER,
            DARK_COUNTER_FILL,
            DARK_COUNTER_STROKE,
            DARK_TEXT_MAIN,
            DARK_TEXT_ALT,
            DARK_TEXT_INVERTED_MAIN,
            DARK_TEXT_INVERTED_ALT,
            DARK_THREAD_STATE_RUNNING,
            DARK_THREAD_STATE_RUNNABLE,
            DARK_THREAD_STATE_IOWAIT,
            DARK_THREAD_STATE_UNINTERRUPTILE,
            DARK_THREAD_STATE_SLEEP,
            DARK_THREAD_STATE_UNKNOWN,
            DARK_MEMORY_BUFFERED_CACHED,
            DARK_MEMORY_USED);
    }
  }

  private static final HSL[] MD_PALETTE = new HSL[] {
      new HSL(4, 90, 58),
      new HSL(340, 82, 52),
      new HSL(291, 64, 42),
      new HSL( 262, 52, 47),
      new HSL(231, 48, 48),
      new HSL(207, 90, 54),
      new HSL(199, 98, 48),
      new HSL(187, 100, 42),
      new HSL(174, 100, 29),
      new HSL(122, 39, 49),
      new HSL(88, 50, 53),
      new HSL(66, 70, 54),
      new HSL(45, 100, 51),
      new HSL(36, 100, 50),
      new HSL(14, 100, 57),
      new HSL(16, 25, 38),
      new HSL(200, 18, 46),
      new HSL(54, 100, 62),
  };
  private static final HSL GRAY_COLOR = new HSL(0, 0, 62);

  private static Colors colors = Colors.light();
  private static boolean isDark = false;

  private StyleConstants() {
  }

  public static Colors colors() {
    return colors;
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

  public static float hueForCpu(int cpu) {
    return (128 + (32 * cpu)) % 256;
  }

  public static HSL colorForThread(ThreadInfo thread) {
    if (thread == null) {
      return GRAY_COLOR;
    }
    return MD_PALETTE[(int)((thread.upid != 0 ? thread.upid : thread.utid) % MD_PALETTE.length)];
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

  public static class HSL {
    public final int h, s, l;

    public HSL(int h, int s, int l) {
      this.h = h;
      this.s = s;
      this.l = l;
    }

    public RGBA rgb() {
      return hsl(h, s / 100f, l / 100f);
    }

    public HSL adjusted(int newH, int newS, int newL) {
      return new HSL(clamp(newH, 0, 360), clamp(newS, 0, 100), clamp(newL, 0, 100));
    }

    private static int clamp(int x, int min, int max) {
      return Math.min(Math.max(x, min), max);
    }
  }
}
