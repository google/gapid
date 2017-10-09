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
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.Resource;
import org.eclipse.swt.widgets.Display;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import java.lang.reflect.InvocationHandler;
import java.lang.reflect.Method;
import java.lang.reflect.Proxy;
import java.util.Map;

/**
 * Contains themable resources that need to be loaded and disposed (such as {@link Image images},
 * {@link Color colors}, etc.).
 */
public interface Theme {
  @Icon("gapid/arrow.png") public Image arrow();
  @Icon("gapid/color_buffer0.png") public Image colorBuffer0();
  @Icon("gapid/color_buffer1.png") public Image colorBuffer1();
  @Icon("gapid/color_buffer2.png") public Image colorBuffer2();
  @Icon("gapid/color_buffer3.png") public Image colorBuffer3();
  @Icon("gapid/color_channels.png") public Image colorChannels();
  @Icon("gapid/culling_disabled.png") public Image cullingDisabled();
  @Icon("gapid/culling_enabled.png") public Image cullingEnabled();
  @Icon("gapid/depth_buffer.png") public Image depthBuffer();
  @Icon("gapid/error.png") public Image error();
  @Icon("gapid/faceted.png") public Image faceted();
  @Icon("gapid/flat.png") public Image flat();
  @Icon("gapid/flip_vertically.png") public Image flipVertically();
  @Icon("gapid/inject_spy.png") public Image injectSpy();
  @Icon("gapid/jump.png") public Image jump();
  @Icon("gapid/listen_for_trace.png") public Image listenForTrace();
  @Icon("gapid/lit.png") public Image lit();
  @Icon("gapid/logo.png") public Image logo();
  @Icon("gapid/logo_big.png") public Image logoBig();
  @Icon("gapid/normals.png") public Image normals();
  @Icon("gapid/opacity.png") public Image opacity();
  @Icon("gapid/point_cloud.png") public Image pointCloud();
  @Icon("gapid/refresh.png") public Image refresh();
  @Icon("gapid/save.png") public Image save();
  @Icon("gapid/smile.png") public Image smile();
  @Icon("gapid/smooth.png") public Image smooth();
  @Icon("gapid/trace_file.png") public Image traceFile();
  @Icon("gapid/transparency.png") public Image transparency();
  @Icon("gapid/winding_ccw.png") public Image windingCCW();
  @Icon("gapid/winding_cw.png") public Image windingCW();
  @Icon("gapid/wireframe_all.png") public Image wireframeAll();
  @Icon("gapid/wireframe_none.png") public Image wireframeNone();
  @Icon("gapid/wireframe_overlay.png") public Image wireframeOverlay();
  @Icon("gapid/yup.png") public Image yUp();
  @Icon("gapid/zup.png") public Image zUp();
  @Icon("gapid/loading_0_small.png") public Image loading0small();
  @Icon("gapid/loading_1_small.png") public Image loading1small();
  @Icon("gapid/loading_2_small.png") public Image loading2small();
  @Icon("gapid/loading_3_small.png") public Image loading3small();
  @Icon("gapid/loading_4_small.png") public Image loading4small();
  @Icon("gapid/loading_5_small.png") public Image loading5small();
  @Icon("gapid/loading_6_small.png") public Image loading6small();
  @Icon("gapid/loading_7_small.png") public Image loading7small();
  @Icon("gapid/loading_0_large.png") public Image loading0large();
  @Icon("gapid/loading_1_large.png") public Image loading1large();
  @Icon("gapid/loading_2_large.png") public Image loading2large();
  @Icon("gapid/loading_3_large.png") public Image loading3large();
  @Icon("gapid/loading_4_large.png") public Image loading4large();
  @Icon("gapid/loading_5_large.png") public Image loading5large();
  @Icon("gapid/loading_6_large.png") public Image loading6large();
  @Icon("gapid/loading_7_large.png") public Image loading7large();

  @Icon("android/zoom_actual.png") public Image zoomActual();
  @Icon("android/zoom_fit.png") public Image zoomFit();
  @Icon("android/zoom_in.png") public Image zoomIn();
  @Icon("android/zoom_out.png") public Image zoomOut();
  @Icon("android/android.png") public Image androidLogo();

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

  // About dialog text colors
  @RGB(argb = 0xff282828) public Color aboutBackground();
  @RGB(argb = 0xffc8c8c8) public Color aboutForeground();

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

  @TextStyle(foreground = 0xa9a9a9) public Styler structureStyler();
  @TextStyle(foreground = 0x0000ee) public Styler identifierStyler();
  @TextStyle(bold = true) public Styler labelStyler();
  @TextStyle(foreground = 0x0000ee, underline = true) public Styler linkStyler();
  @TextStyle(foreground = 0xee0000) public Styler errorStyler();
  @TextStyle(foreground = 0xffc800) public Styler warningStyler();

  @Text(Text.Mono) public Font getMonoSpaceFont();

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
    public String value();
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
    public static final int Mono = 1;

    public int value();
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
      return loadIcon(method) || loadColor(method) || loadTextStyle(method) || loadFont(method);
    }

    private boolean loadIcon(Method method) {
      Icon icon = method.getDeclaredAnnotation(Icon.class);
      if (icon != null) {
        resources.put(method.getName(), ImageDescriptor.createFromURL(
            Resources.getResource("icons/" + icon.value())).createImage(display));
        return true;
      }
      return false;
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
          case Text.Mono:
            Font font = FontDescriptor.createFrom(JFaceResources.getFont(JFaceResources.TEXT_FONT))
                .setHeight(JFaceResources.getDefaultFont().getFontData()[0].getHeight())
                .createFont(display);
            resources.put(method.getName(), font);
            return true;
        }
      }
      return false;
    }
  }
}
