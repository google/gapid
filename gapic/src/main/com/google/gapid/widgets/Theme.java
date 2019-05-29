/*
 * Copyright (C) 2017 Google Inc.
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
package com.google.gapid.widgets;

import static com.google.gapid.util.Colors.fromARGB;

import com.google.common.collect.Maps;
import com.google.common.io.Resources;
import com.google.gapid.util.Colors;

import org.eclipse.jface.resource.FontDescriptor;
import org.eclipse.jface.resource.ImageDescriptor;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.viewers.StyledString.Styler;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.FontData;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Resource;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.widgets.Display;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import java.lang.reflect.InvocationHandler;
import java.lang.reflect.Method;
import java.lang.reflect.Proxy;
import java.net.URL;
import java.util.Map;

/**
 * Contains themable resources that need to be loaded and disposed (such as {@link Image images},
 * {@link Color colors}, etc.).
 */
public interface Theme {
  @Icon(file = "android.png", color = 0x335577) public Image androidLogo();
  @Icon(file = "arrow.png") public Image arrow();
  @Icon(file = "arrow_drop_down.png") public Image arrowDropDownLight();
  @Icon(file = "arrow_drop_right.png") public Image arrowDropRightLight();
  @Icon(file = "arrow_drop_down.png", color = 0xFFFFFF) public Image arrowDropDownDark();
  @Icon(file = "arrow_drop_right.png", color = 0xFFFFFF) public Image arrowDropRightDark();
  @Icon(file = "color_buffer0.png") public Image colorBuffer0();
  @Icon(file = "color_buffer1.png") public Image colorBuffer1();
  @Icon(file = "color_buffer2.png") public Image colorBuffer2();
  @Icon(file = "color_buffer3.png") public Image colorBuffer3();
  @Icon(file = "culling_disabled.png") public Image cullingDisabled();
  @Icon(file = "culling_enabled.png") public Image cullingEnabled();
  @Icon(file = "depth_buffer.png") public Image depthBuffer();
  @Icon(file = "error.png") public Image error();
  @Icon(file = "faceted.png") public Image faceted();
  @Icon(file = "flat.png") public Image flat();
  @Icon(file = "flip_vertically.png") public Image flipVertically();
  @Icon(file = "jump.png") public Image jump();
  @Icon(file = "histogram.png") public Image toggleHistogram();
  @Icon(file = "lit.png") public Image lit();
  @Icon(file = "logo_128.png") public Image dialogLogo();
  @Icon(file = "normals.png") public Image normals();
  @Icon(file = "overdraw.png") public Image overdraw();
  @Icon(file = "point_cloud.png") public Image pointCloud();
  @Icon(file = "range_start.png") public Image rangeStartLight();
  @Icon(file = "range_end.png") public Image rangeEndLight();
  @Icon(file = "range_start.png", color = 0xFFFFFF) public Image rangeStartDark();
  @Icon(file = "range_end.png", color = 0xFFFFFF) public Image rangeEndDark();
  @Icon(file = "refresh.png") public Image refresh();
  @Icon(file = "save.png") public Image save();
  @Icon(file = "settings.png") public Image settings();
  @Icon(file = "smile.png") public Image smile();
  @Icon(file = "smooth.png") public Image smooth();
  @Icon(file = "transparency.png") public Image transparency();
  @Icon(file = "unfold_less.png") public Image unfoldLessLight();
  @Icon(file = "unfold_more.png") public Image unfoldMoreLight();
  @Icon(file = "unfold_less.png", color = 0xFFFFFF) public Image unfoldLessDark();
  @Icon(file = "unfold_more.png", color = 0xFFFFFF) public Image unfoldMoreDark();
  @Icon(file = "winding_ccw.png") public Image windingCCW();
  @Icon(file = "winding_cw.png") public Image windingCW();
  @Icon(file = "wireframe_all.png") public Image wireframeAll();
  @Icon(file = "wireframe_none.png") public Image wireframeNone();
  @Icon(file = "wireframe_overlay.png") public Image wireframeOverlay();
  @Icon(file = "yup.png") public Image yUp();
  @Icon(file = "zup.png") public Image zUp();
  @Icon(file = "zoom_actual.png") public Image zoomActual();
  @Icon(file = "zoom_fit.png") public Image zoomFit();
  @Icon(file = "zoom_in.png") public Image zoomIn();
  @Icon(file = "zoom_out.png") public Image zoomOut();

  @IconSequence(names = {
      "logo_128.png", "logo_64.png", "logo_48.png", "logo_32.png", "logo_16.png",
  }) public Image[] windowLogo();
  @IconSequence(pattern = "color_channels_%02d.png", count = 16) public Image[] colorChannels();
  @IconSequence(pattern = "loading_%d_small.png", count = 8) public Image[] loadingSmall();
  @IconSequence(pattern = "loading_%d_large.png", count = 8) public Image[] loadingLarge();

  // Shader source highlight colors.
  @RGB(argb = 0xff808080) public Color commentColor();
  @RGB(argb = 0xff7f0055) public Color keywordColor();
  @RGB(argb = 0xff000080) public Color identifierColor();
  @RGB(argb = 0xff0000ff) public Color numericConstantColor();
  @RGB(argb = 0xff808000) public Color preprocessorColor();

  // Memory highlighting (background) colors.
  @RGB(argb = 0xffdcfadc) public Color memoryReadHighlight();
  @RGB(argb = 0xfffadcdc) public Color memoryWriteHighlight();
  @RGB(argb = 0xffdcdcfa) public Color memorySelectionHighlight();

  // About & Welcome dialog text colors
  @RGB(argb = 0xffa9a9a9) public Color welcomeVersionColor();

  // Logging view colors by log level.
  @RGB(argb = 0xbb000000) public Color logVerboseForeground();
  @RGB(argb = 0xffcecece) public Color logVerboseBackground();
  @RGB(argb = 0xdd000000) public Color logDebugForeground();
  @RGB(argb = 0xffe5e5e5) public Color logDebugBackground();
  @RGB(argb = 0xff000000) public Color logInfoForeground();
  @RGB(argb = 0xfff2f2f2) public Color logInfoBackground();
  @RGB(argb = 0xff4c4a00) public Color logWarningForeground();
  @RGB(argb = 0xfffffa82) public Color logWarningBackground();
  @RGB(argb = 0xff4c0100) public Color logErrorForeground();
  @RGB(argb = 0xffff8484) public Color logErrorBackground();
  @RGB(argb = 0xff1b004c) public Color logFatalForeground();
  @RGB(argb = 0xffcd83ff) public Color logFatalBackground();

  // Image panel colors.
  @RGB(argb = 0xffc0c0c0) public Color imageCheckerDark();
  @RGB(argb = 0xffffffff) public Color imageCheckerLight();
  @RGB(argb = 0xff000000) public Color imageCursorDark();
  @RGB(argb = 0xffffffff) public Color imageCursorLight();
  @RGB(argb = 0xffff9900) public Color imageWarning();

  @RGB(argb = 0xc0404040) public Color histogramBackgroundDark();
  @RGB(argb = 0xc0555555) public Color histogramBackgroundLight();
  @RGB(argb = 0x80000000) public Color histogramCurtain();
  @RGB(argb = 0xc0202020) public Color histogramArrow();

  @RGB(argb = 0xffcccccc) public Color statusBarMemoryBar();

  @TextStyle(foreground = 0xa9a9a9) public Styler structureStyler();
  @TextStyle(foreground = 0x0000ee) public Styler identifierStyler();
  @TextStyle(bold = true) public Styler labelStyler();
  @TextStyle(foreground = 0x0000ee, underline = true) public Styler linkStyler();
  @TextStyle(foreground = 0xee0000) public Styler errorStyler();
  @TextStyle(foreground = 0xffc800) public Styler warningStyler();

  @Text(Text.Mono) public Font monoSpaceFont();
  @Text(Text.Big) public Font bigBoldFont();

  public void dispose();

  public static Theme load(Display display) {
    Map<String, Object> resources = new Loader(display).load();
    return (Theme)Proxy.newProxyInstance(
        Theme.class.getClassLoader(), new Class<?>[] { Theme.class }, new InvocationHandler() {
      @Override
      public Object invoke(Object proxy, Method method, Object[] args) throws Throwable {
        Object result = null;
        if ("dispose".equals(method.getName())) {
          for (Object resource : resources.values()) {
            if (resource instanceof Resource) {
              ((Resource)resource).dispose();
            } else if (resource instanceof DisposableStyler) {
              ((DisposableStyler)resource).dispose();
            } else if (resource instanceof Image[]) {
              for (Image image : (Image[])resource) {
                image.dispose();
              }
            }
          }
          resources.clear();
        } else {
          result = resources.get(method.getName());
          if (result == null) {
            throw new RuntimeException("Resource not found: " + method.getName());
          }
        }
        return result;
      }
    });
  }

  /**
   * Annotation for an icon image resource.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface Icon {
    public String file();
    public int color() default -1;
  }

  /**
   * Annotation for an icon sheet image resource.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface IconSequence {
    /**
     * @return the list of image file names in the sequence. If this is provided, the pattern and
     * count are ignored.
     */
    public String[] names() default { };

    /**
     * @return a {@link String#format(String, Object...)} pattern given a sequence number, i, to
     *     format the file name of the ith image in the sequence.
     */
    public String pattern() default "";

    /**
     * @return the number of images in the sequence.
     */
    public int count() default 0;
  }

  /**
   * Annotation for a color resource.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface RGB {
    public int argb();
  }

  /**
   * Annotation for a text style resource.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface TextStyle {
    public int foreground() default -1;
    public int background() default -1;
    public boolean underline() default false;
    public boolean bold() default false;
  }

  /**
   * Annotation for a font resource.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface Text {
    public static final int Mono = 1, Big = 2;

    public int value();
  }

  /**
   * Annotation for a font resource loaded from a TTF file.
   */
  @Target(ElementType.METHOD)
  @Retention(RetentionPolicy.RUNTIME)
  public static @interface TTF {
    public String ttf();
    public String name();
    public int size();
  }

  /**
   * A {@link Styler} that will dispose its resources.
   */
  public static class DisposableStyler extends Styler {
    private final Color foreground, background;
    private final boolean underline, bold;

    public DisposableStyler(Color foreground, Color background, boolean underline, boolean bold) {
      this.foreground = foreground;
      this.background = background;
      this.underline = underline;
      this.bold = bold;
    }

    public DisposableStyler(Display display, TextStyle style) {
      this(style.foreground() >= 0 ? new Color(display, Colors.fromRGB(style.foreground())) : null,
          style.background() >= 0 ? new Color(display, Colors.fromRGB(style.background())) : null,
          style.underline(), style.bold());
    }

    public void dispose() {
      if (foreground != null) {
        foreground.dispose();
      }
      if (background != null) {
        background.dispose();
      }
    }

    @Override
    public void applyStyles(org.eclipse.swt.graphics.TextStyle textStyle) {
      textStyle.foreground = foreground;
      textStyle.background = background;
      textStyle.underline = underline;
      if (bold) {
        textStyle.font = JFaceResources.getFontRegistry().getBold(JFaceResources.DEFAULT_FONT);
      }
    }
  }

  /**
   * Resource loader for the theme.
   */
  public static class Loader {
    private final Display display;
    private final Map<String, Object> resources = Maps.newHashMap();

    public Loader(Display display) {
      this.display = display;
    }

    public Map<String, Object> load() {
      for (Method method : Theme.class.getDeclaredMethods()) {
        loadResource(method);
      }
      return resources;
    }

    private boolean loadResource(Method method) {
      return loadIcon(method) || loadIconSequence(method) || loadColor(method) ||
          loadTextStyle(method) || loadFont(method);
    }

    private boolean loadIcon(Method method) {
      Icon icon = method.getDeclaredAnnotation(Icon.class);
      if (icon != null) {
        resources.put(method.getName(), loadImage(icon.file(), icon.color()));
        return true;
      }
      return false;
    }

    private boolean loadIconSequence(Method method) {
      IconSequence seq = method.getDeclaredAnnotation(IconSequence.class);
      if (seq != null) {
        String[] names = seq.names();
        Image[] icons = new Image[names.length == 0 ? seq.count() : names.length];
        for (int i = 0; i < icons.length; i++) {
          icons[i] = loadImage(names.length == 0 ? String.format(seq.pattern(), i) : names[i], -1);
        }
        resources.put(method.getName(), icons);
        return true;
      }
      return false;
    }

    private Image loadImage(String img, int color) {
      ImageData data = ImageDescriptor.createFromURL(Resources.getResource("icons/" + img))
          .getImageData(DPIUtil.getDeviceZoom());
      if (color >= 0) {
        for (int y = 0, o = 0; y < data.height; y++, o += data.bytesPerLine) {
          for (int x = 0, i = o; x < data.width; x++) {
            data.data[i++] = (byte)((color >> 16) & 0xFF);
            data.data[i++] = (byte)((color >> 8) & 0xFF);
            data.data[i++] = (byte)(color & 0xFF);
          }
        }
      }
      return new Image(display, data);
    }

    private boolean loadColor(Method method) {
      RGB rgb = method.getDeclaredAnnotation(RGB.class);
      if (rgb != null) {
        resources.put(method.getName(), new Color(display, fromARGB(rgb.argb())));
        return true;
      }
      return false;
    }

    private boolean loadTextStyle(Method method) {
      TextStyle style = method.getDeclaredAnnotation(TextStyle.class);
      if (style != null) {
        resources.put(method.getName(), new DisposableStyler(display, style));
        return true;
      }
      return false;
    }

    public boolean loadFont(Method method) {
      Text text = method.getDeclaredAnnotation(Text.class);
      if (text != null) {
        switch (text.value()) {
          case Text.Mono: {
            Font font = FontDescriptor.createFrom(JFaceResources.getFont(JFaceResources.TEXT_FONT))
                .setHeight(JFaceResources.getDefaultFont().getFontData()[0].getHeight())
                .createFont(display);
            resources.put(method.getName(), font);
            return true;
          }
          case Text.Big: {
            Font dflt = JFaceResources.getDefaultFont();
            Font font = FontDescriptor.createFrom(dflt)
                .setHeight(dflt.getFontData()[0].getHeight() * 3 / 2)
                .setStyle(SWT.BOLD)
                .createFont(display);
            resources.put(method.getName(), font);
            return true;
          }
        }
      }

      TTF ttf = method.getDeclaredAnnotation(TTF.class);
      if (ttf != null) {
        URL url = Resources.getResource("fonts/" + ttf.ttf());
        if (url == null) {
          return false;
        }
        try {
          File fontFile = File.createTempFile(method.getName(), ".ttf");
          fontFile.deleteOnExit();
          try (OutputStream out = new FileOutputStream(fontFile)) {
            Resources.copy(url, out);
          }
          if (!display.loadFont(fontFile.getAbsolutePath())) {
            return false;
          }
          resources.put(method.getName(),
              new Font(display, new FontData(ttf.name(), ttf.size(), SWT.NORMAL)));
          return true;
        } catch (IOException e) {
          e.printStackTrace();
          return false;
        }
      }
      return false;
    }
  }
}
