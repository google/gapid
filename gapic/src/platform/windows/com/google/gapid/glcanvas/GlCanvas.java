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
// Modified from SWT's org.eclipse.swt.opengl.GLCanvas with license:
/*******************************************************************************
 * Copyright (c) 2000, 2016 IBM Corporation and others.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/
package com.google.gapid.glcanvas;

import static org.lwjgl.system.MemoryUtil.memAddress;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.lwjgl.opengl.WGL;
import org.lwjgl.opengl.WGLARBMultisample;
import org.lwjgl.opengl.WGLARBPixelFormat;
import org.lwjgl.system.JNI;
import org.lwjgl.system.MemoryStack;
import org.lwjgl.system.MemoryUtil;
import org.lwjgl.system.windows.GDI32;
import org.lwjgl.system.windows.PIXELFORMATDESCRIPTOR;
import org.lwjgl.system.windows.User32;

import java.nio.IntBuffer;

public abstract class GlCanvas extends Canvas {
  private static final String USE_OWNDC_KEY = "org.eclipse.swt.internal.win32.useOwnDC";

  private final long context;
  private final boolean initialized;

  public GlCanvas(Composite parent, int style) {
    super(parent, checkStyle(parent, style));
    Canvas initCanvas = new Canvas(parent, style);
    parent.getDisplay().setData(USE_OWNDC_KEY, Boolean.FALSE);

    Context initContext = new Context(initCanvas.handle).createSimplePixelFormat();
    if (initContext == null) {
      context = 0;
      initialized = false;
      return;
    }

    Context actualContext;
    if (initContext.createPixelFormat()) {
      actualContext = new Context(handle).setPixelFormat(initContext.pixelFormat);
    } else {
      actualContext = new Context(handle).createSimplePixelFormat();
    }

    if (actualContext == null) {
      context = 0;
      initialized = false;
    } else {
      context = actualContext.context;
      initialized = true;
    }

    initContext.release();
    initCanvas.dispose();

    if (initialized) {
      addListener(SWT.Dispose, e -> {
        terminate();
        WGL.wglDeleteContext(context);
      });
    }
  }

  private static int checkStyle(Composite parent, int style) {
    if (parent != null) {
      parent.getDisplay().setData(USE_OWNDC_KEY, Boolean.TRUE);
    }
    return style;
  }

  public boolean isOpenGL() {
    return initialized;
  }

  public void setCurrent () {
    checkWidget();
    if (!initialized) {
      return;
    }

    long dc = User32.GetDC(handle);
    WGL.wglMakeCurrent(dc, context);
    User32.ReleaseDC(handle, dc);
  }

  public void swapBuffers () {
    checkWidget();
    if (!initialized) {
      return;
    }

    long dc = User32.GetDC(handle);
    GDI32.SwapBuffers(dc);
    User32.ReleaseDC(handle, dc);
  }

  /** Override to perform GL cleanup handling. */
  protected abstract void terminate();

  private static class Context {
    public final long handle;
    public final long dc;
    public int pixelFormat;
    public long context;

    public Context(long handle) {
      this.handle = handle;
      this.dc = User32.GetDC(handle);
    }

    public Context createSimplePixelFormat() {
      PIXELFORMATDESCRIPTOR pfd = PIXELFORMATDESCRIPTOR.calloc();
      pfd.nSize((short)PIXELFORMATDESCRIPTOR.SIZEOF);
      pfd.nVersion((short)1);
      pfd.dwFlags(GDI32.PFD_DRAW_TO_WINDOW | GDI32.PFD_SUPPORT_OPENGL | GDI32.PFD_DOUBLEBUFFER);
      pfd.dwLayerMask(GDI32.PFD_MAIN_PLANE);
      pfd.iPixelType(GDI32.PFD_TYPE_RGBA);
      pfd.cRedBits((byte)8);
      pfd.cGreenBits((byte)8);
      pfd.cBlueBits((byte)8);
      pfd.cDepthBits((byte)24);

      pixelFormat = GDI32.ChoosePixelFormat(dc, pfd);
      if (pixelFormat == 0 || !GDI32.SetPixelFormat(dc, pixelFormat, pfd)) {
        release();
        return null;
      }
      return createContext();
    }

    public Context setPixelFormat(int format) {
      this.pixelFormat = format;
      PIXELFORMATDESCRIPTOR pfd = PIXELFORMATDESCRIPTOR.calloc();
      pfd.nSize((short)PIXELFORMATDESCRIPTOR.SIZEOF);
      pfd.nVersion((short)1);
      pfd.dwFlags(GDI32.PFD_DRAW_TO_WINDOW | GDI32.PFD_SUPPORT_OPENGL | GDI32.PFD_DOUBLEBUFFER);
      pfd.dwLayerMask(GDI32.PFD_MAIN_PLANE);
      pfd.iPixelType(GDI32.PFD_TYPE_RGBA);
      pfd.cRedBits((byte)8);
      pfd.cGreenBits((byte)8);
      pfd.cBlueBits((byte)8);
      pfd.cDepthBits((byte)24);
      if (!GDI32.SetPixelFormat(dc, pixelFormat, pfd)) {
        release();
        return null;
      }
      return createContext();
    }

    private Context createContext() {
      context = WGL.wglCreateContext(dc);
      if (context == 0 || !WGL.wglMakeCurrent(dc, context)) {
        User32.ReleaseDC(handle, dc);
        return null;
      }
      return this;
    }

    public boolean createPixelFormat() {
      long chooseFormat = WGL.wglGetProcAddress("wglChoosePixelFormatARB");
      if (chooseFormat == 0) {
        chooseFormat = WGL.wglGetProcAddress("wglChoosePixelFormatEXT");
        if (chooseFormat == 0) {
          return false;
        }
      }

      IntBuffer buf = MemoryStack.stackCallocInt(20);
      long bufAddr = memAddress(buf);
      set(buf, WGLARBPixelFormat.WGL_DRAW_TO_WINDOW_ARB, 1);
      set(buf, WGLARBPixelFormat.WGL_SUPPORT_OPENGL_ARB, 1);
      set(buf, WGLARBPixelFormat.WGL_ACCELERATION_ARB, WGLARBPixelFormat.WGL_FULL_ACCELERATION_ARB);
      set(buf, WGLARBPixelFormat.WGL_PIXEL_TYPE_ARB, WGLARBPixelFormat.WGL_TYPE_RGBA_ARB);
      set(buf, WGLARBPixelFormat.WGL_COLOR_BITS_ARB, 24);
      set(buf, WGLARBPixelFormat.WGL_DEPTH_BITS_ARB, 24);
      set(buf, WGLARBPixelFormat.WGL_DOUBLE_BUFFER_ARB, 1);
      set(buf, WGLARBMultisample.WGL_SAMPLE_BUFFERS_ARB, 1);
      set(buf, WGLARBMultisample.WGL_SAMPLES_ARB, 4);
      set(buf, 0, 0);
      long result = MemoryStack.nstackMalloc(4, 4 * 2);
      if (JNI.callPPPPPI(dc, bufAddr, 0L, 1, result + 4, result, chooseFormat) != 1 ||
          MemoryUtil.memGetInt(result) == 0) {
        return false;
      }

      pixelFormat = MemoryUtil.memGetInt(result + 4);
      return true;
    }

    private static void set(IntBuffer attrib, int name, int value) {
      attrib.put(name).put(value);
    }

    public void release() {
      User32.ReleaseDC(handle, dc);
      if (context != 0) {
        WGL.wglDeleteContext(context);
      }
    }
  }
}
