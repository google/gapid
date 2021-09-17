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
package com.google.gapid.glcanvas;

import static java.util.logging.Level.SEVERE;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.internal.gtk.GDK;
import org.eclipse.swt.internal.gtk.GTK;
import org.eclipse.swt.internal.gtk3.GTK3;
import org.eclipse.swt.internal.gtk3.GdkWindowAttr;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.lwjgl.PointerBuffer;
import org.lwjgl.opengl.GL;
import org.lwjgl.opengl.GLX;
import org.lwjgl.opengl.GLX13;
import org.lwjgl.opengl.GLX14;
import org.lwjgl.opengl.GLXCapabilities;
import org.lwjgl.system.Library;
import org.lwjgl.system.MemoryStack;
import org.lwjgl.system.linux.X11;
import org.lwjgl.system.linux.XVisualInfo;

import java.nio.Buffer;
import java.nio.IntBuffer;
import java.util.logging.Logger;

public abstract class GlCanvas extends Canvas {
  private static final Logger LOG = Logger.getLogger(GlCanvas.class.getName());

  private static final boolean jniLibraryLoaded;

  static {
    boolean loaded = false;
    try {
      Library.loadSystem("", "linux_glcanvas");
      loaded = true;
    } catch (UnsatisfiedLinkError e) {
      LOG.log(SEVERE, "Failed to load GlCanvas JNI library", e);
    }
    jniLibraryLoaded = loaded;
  }

  private long gdkWindow;
  private long xWindow;
  private long context;

  public GlCanvas(Composite parent, int style) {
    super(parent, style);
    if (jniLibraryLoaded && createContext()) {
      addListener(SWT.Resize, e -> {
        Rectangle clientArea = DPIUtil.autoScaleUp(getClientArea());
        GDK.gdk_window_move(gdkWindow, clientArea.x, clientArea.y);
        GDK.gdk_window_resize(gdkWindow, clientArea.width, clientArea.height);
      });
      addListener(SWT.Dispose, e -> {
        long display = getXDisplay();
        if (context != 0) {
          terminate();
          GLX.glXMakeCurrent(display, 0, 0);
          GLX.glXDestroyContext(display, context);
          context = 0;
        }
        if (gdkWindow != 0) {
          GDK.gdk_window_destroy(gdkWindow);
          gdkWindow = 0;
        }
      });
    }
  }

  private boolean createContext() {
    GTK.gtk_widget_realize(handle);
    long display = getXDisplay();
    int screen = X11.XDefaultScreen(display);

    GLXCapabilities glxCaps = GL.createCapabilitiesGLX(display, screen);
    if (!glxCaps.GLX13 || !glxCaps.GLX_ARB_create_context || !glxCaps.GLX_ARB_create_context_profile) {
      LOG.log(SEVERE, "Inssufficient GLX capabilities. GLX13: " + glxCaps.GLX13 +
          " GLX_ARB_create_context: " + glxCaps.GLX_ARB_create_context +
          " GLX_ARB_create_context_profile: " + glxCaps.GLX_ARB_create_context_profile);
      return false;
    }

    long config = chooseConfig(glxCaps, display);
    if (config == -1) {
      return false;
    }

    if (!createWindow(display, config)) {
      return false;
    }

    context = createContext0(display, config);
    if (context == 0) {
      LOG.log(SEVERE, "Failed to create an OpenGL 3.2 Core context");
      return false;
    }

    return true;
  }

  private static long chooseConfig(GLXCapabilities glxCaps, long display) {
    try (MemoryStack stack = MemoryStack.stackPush()) {
      IntBuffer attr = stack.mallocInt(60);
      set(attr, GLX13.GLX_X_RENDERABLE, 1);
      set(attr, GLX13.GLX_DRAWABLE_TYPE, GLX13.GLX_WINDOW_BIT);
      set(attr, GLX13.GLX_RENDER_TYPE, GLX13.GLX_RGBA_BIT);
      set(attr, GLX13.GLX_X_VISUAL_TYPE, GLX13.GLX_TRUE_COLOR);
      set(attr, GLX.GLX_DOUBLEBUFFER, 1);
      set(attr, GLX.GLX_RED_SIZE, 8);
      set(attr, GLX.GLX_GREEN_SIZE, 8);
      set(attr, GLX.GLX_BLUE_SIZE, 8);
      set(attr, GLX.GLX_DEPTH_SIZE, 24);
      if (glxCaps.GLX14 || glxCaps.GLX_ARB_multisample) {
        set(attr, GLX14.GLX_SAMPLE_BUFFERS, 1);
        set(attr, GLX14.GLX_SAMPLES, 4);
      }
      set(attr, X11.None, X11.None);

      ((Buffer)attr).flip(); // cast is there to work with JDK9.
      PointerBuffer configs = GLX13.glXChooseFBConfig(display, X11.XDefaultScreen(display), attr);
      if (configs == null || configs.capacity() < 1) {
        LOG.log(SEVERE, "glXChooseFBConfig returned no matching configs");
        return -1;
      }

      long config = configs.get(0);
      X11.XFree(configs);
      return config;
    }
  }

  private boolean createWindow(long display, long config) {
    try (XVisualInfo visual = GLX13.glXGetVisualFromFBConfig(display, config)) {
      if (visual == null) {
        LOG.log(SEVERE, "glXGetVisualFromFBConfig returned null");
        return false;
      }

      GdkWindowAttr attrs = new GdkWindowAttr();
      attrs.width = 1;
      attrs.height = 1;
      attrs.event_mask = GDK.GDK_KEY_PRESS_MASK | GDK.GDK_KEY_RELEASE_MASK |
          GDK.GDK_FOCUS_CHANGE_MASK | GDK.GDK_POINTER_MOTION_MASK |
          GDK.GDK_BUTTON_PRESS_MASK | GDK.GDK_BUTTON_RELEASE_MASK |
          GDK.GDK_ENTER_NOTIFY_MASK | GDK.GDK_LEAVE_NOTIFY_MASK |
          GDK.GDK_EXPOSURE_MASK | GDK.GDK_POINTER_MOTION_HINT_MASK;
      attrs.window_type = GDK.GDK_WINDOW_CHILD;
      attrs.visual =
          GDK.gdk_x11_screen_lookup_visual(GDK.gdk_screen_get_default(), (int)visual.visualid());
      gdkWindow = GTK3.gdk_window_new(GTK3.gtk_widget_get_window(handle), attrs, GDK.GDK_WA_VISUAL);
      if (gdkWindow == 0) {
        LOG.log(SEVERE, "Failed to create the GDK window");
        return false;
      }
      GDK.gdk_window_set_user_data(gdkWindow, handle);
    }

    xWindow = GDK.gdk_x11_window_get_xid(gdkWindow);
    GDK.gdk_window_show(gdkWindow);
    return true;
  }

  private static void set(IntBuffer buf, int name, int value) {
    buf.put(name).put(value);
  }

  private static native long createContext0(long display, long config);

  public boolean isOpenGL() {
    return context != 0;
  }

  public void setCurrent() {
    if (context == 0) {
      return;
    }

    checkWidget();
    GLX.glXMakeCurrent(getXDisplay(), xWindow, context);
  }

  public void swapBuffers () {
    if (context == 0) {
      return;
    }

    checkWidget();
    GLX.glXSwapBuffers(getXDisplay(), xWindow);
  }

  /**
   * Override to perform GL cleanup handling.
   */
  protected abstract void terminate();

  private long getXDisplay() {
    return GDK.gdk_x11_display_get_xdisplay(
        GDK.gdk_window_get_display(GTK3.gtk_widget_get_window(handle)));
  }
}
