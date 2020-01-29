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
  public static final double PROCESS_COUNTER_TRACK_HIGHT = 30;
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

    public final RGBA textMain;
    public final RGBA textAlt;
    public final RGBA textInvertedMain;
    public final RGBA textInvertedAlt;

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
        RGBA textMain,
        RGBA textAlt,
        RGBA textInvertedMain,
        RGBA textInvertedAlt) {
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
      this.textMain = textMain;
      this.textAlt = textAlt;
      this.textInvertedMain = textInvertedMain;
      this.textInvertedAlt = textInvertedAlt;
    }

    private static final RGBA LIGHT_BACKGROUND = rgb(0xff, 0xff, 0xff);
    private static final RGBA LIGHT_TITLE_BACKGROUND = rgb(0xf1, 0xf1, 0xf1);
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
    private static final RGBA LIGHT_VSYNC_BACKGROUND = rgb(0xf9, 0xf9, 0xf9);

    private static final RGBA LIGHT_TEXT_MAIN = rgb(0x32, 0x34, 0x35);
    private static final RGBA LIGHT_TEXT_ALT = rgb(101, 102, 104);
    private static final RGBA LIGHT_TEXT_INVERTED_MAIN = rgb(0xff, 0xff, 0xff);
    private static final RGBA LIGHT_TEXT_INVERTED_ALT = rgb(0xdd, 0xdd, 0xdd);

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
            LIGHT_TEXT_MAIN,
            LIGHT_TEXT_ALT,
            LIGHT_TEXT_INVERTED_MAIN,
            LIGHT_TEXT_INVERTED_ALT);
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

    private static final RGBA DARK_TEXT_MAIN = rgb(0xf1, 0xf1, 0xf8);
    private static final RGBA DARK_TEXT_ALT = rgb(0xdd, 0xdd, 0xdd);
    private static final RGBA DARK_TEXT_INVERTED_MAIN = rgb(0x19, 0x1A, 0x19);
    private static final RGBA DARK_TEXT_INVERTED_ALT = rgb(50, 51, 52);

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
            DARK_TEXT_MAIN,
            DARK_TEXT_ALT,
            DARK_TEXT_INVERTED_MAIN,
            DARK_TEXT_INVERTED_ALT);
    }
  }

  private static final HSL GRAY_COLOR = new HSL(0, 0, 62);

  /*
   * Definition and API for colors in GAPID.
   * */
  public static class Palette {
    public static enum BaseColor {
      GREY(new HSL(202, 25, 67)),
      LIGHT_BLUE(new HSL(200, 98, 82)),
      ORANGE(new HSL(20, 90, 70)),
      GREEN(new HSL(131, 65, 67)),
      LIGHT_PURPLE(new HSL(262, 51, 70)),
      PINK(new HSL(338, 70, 67)),
      TURQUOISE(new HSL(171, 74, 61)),
      TAN(new HSL(42, 50, 75)),
      PACIFIC_BLUE(new HSL(198, 100, 42)),
      TEAL(new HSL(172, 74, 29)),
      PURPLE(new HSL(262, 70, 58)),
      DARK_ORANGE(new HSL(22, 76, 43)),
      INDIGO(new HSL(217, 50, 40)),
      LIME(new HSL(59, 51, 45)),
      MAGENTA(new HSL(320, 53, 47)),
      VIVID_BLUE(new HSL(215, 100, 55));

      public final HSL hsl;
      public final RGBA rgb;

      private BaseColor(HSL hsl) {
        this.hsl = hsl;
        this.rgb = hsl.rgb();
      }
    }

    private static final int COLOR_COUNT = BaseColor.values().length;
    private static final int[] LIGHT_OFFSETS = new int[] {
        5, 2, 4, 5, 4, 5, 6, 3, 10, 12, 6, 9, 10, 9, 9, 7,
    };
    private static final int[] DARK_OFFSETS = new int[] {
        -3, -6, -4, -3, -4, -3, -2, -5, -1, -1, -2, -1, -1, -1, -1, -1,
    };
    private static final int LIGHT_SHADE_COUNT = 5;
    private static final int DARK_SHADE_COUNT = 5;
    private static final HSL[][] LIGHT_COLORS = createLightThemeColor();
    private static final HSL[][] DARK_COLORS = createDarkThemeColor();

    /**
     * Retrieve the color from the basic palette.
     */
    public static RGBA getColor(int hueIdx) {
      return BaseColor.values()[hueIdx % COLOR_COUNT].rgb;
    }

    /**
     * Retrieve the color with an adjusted brightness.
     *
     * @param hueIdx decides what color you get, e.g. it's green or purple.
     * @param shadeIdx decides how pale the color you get. If it's positive ,the color is retrieved
     * from Light theme, otherwise Dark theme. Within a limit, the larger the number, the lighter.
     */
    public static RGBA getColor(int hueIdx, int shadeIdx) {
      if (shadeIdx == 0) {
        return BaseColor.values()[hueIdx % COLOR_COUNT].rgb;
      } else if (shadeIdx > 0) {
        shadeIdx = shadeIdx > LIGHT_SHADE_COUNT ? (LIGHT_SHADE_COUNT - 1) : (shadeIdx - 1);
        return LIGHT_COLORS[hueIdx % COLOR_COUNT][shadeIdx].rgb();
      } else {
        shadeIdx = shadeIdx < -DARK_SHADE_COUNT ? (DARK_SHADE_COUNT - 1) : (-shadeIdx - 1);
        return DARK_COLORS[hueIdx % COLOR_COUNT][shadeIdx].rgb();
      }
    }

    private static HSL[][] createLightThemeColor() {
      HSL[][] light = new HSL[COLOR_COUNT][LIGHT_SHADE_COUNT];
      for (int hueIdx = 0; hueIdx < COLOR_COUNT; hueIdx++) {
        HSL base = BaseColor.values()[hueIdx].hsl;
        int offset = LIGHT_OFFSETS[hueIdx];
        for (int shade = 0; shade < LIGHT_SHADE_COUNT; shade++) {
          light[hueIdx][shade] = new HSL(base.h, base.s, base.l + (shade + 1) * offset);
        }
      }
      return light;
    }

    private static HSL[][] createDarkThemeColor() {
      HSL[][] dark = new HSL[COLOR_COUNT][DARK_SHADE_COUNT];
      for (int hueIdx = 0; hueIdx < COLOR_COUNT; hueIdx++) {
        HSL base = BaseColor.values()[hueIdx].hsl;
        int offset = DARK_OFFSETS[hueIdx];
        for (int shade = 0; shade < DARK_SHADE_COUNT; shade++) {
          dark[hueIdx][shade] = new HSL(base.h, base.s, base.l + (shade + 1) * offset);
        }
      }
      return dark;
    }
  }

  private static Colors colors = Colors.light();
  private static boolean isDark = false;

  private StyleConstants() {
  }

  public static Colors colors() {
    return colors;
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

  public static HSL getGrayColor() {
    return GRAY_COLOR;
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
