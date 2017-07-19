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

import org.eclipse.swt.SWT;
import org.eclipse.swt.internal.cocoa.NSNotificationCenter;
import org.eclipse.swt.internal.cocoa.NSOpenGLContext;
import org.eclipse.swt.internal.cocoa.NSOpenGLPixelFormat;
import org.eclipse.swt.internal.cocoa.OS;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Listener;

public abstract class GlCanvas extends Canvas {
  private static final String GLCONTEXT_KEY = "org.eclipse.swt.internal.cocoa.glcontext";

  private static final int NSOpenGLPFAOpenGLProfile = 99;
  private static final int NSOpenGLProfileVersion3_2Core = 0x3200;
  private static final int NSOpenGLPFAAccelerated = 73;
  private static final int NSOpenGLPFANoRecovery = 72;

  private NSOpenGLContext context;
  private NSOpenGLPixelFormat pixelFormat;

  public GlCanvas(Composite parent, int style) {
    super(parent, style);
    OS.objc_msgSend(view.id, OS.sel_registerName("setWantsBestResolutionOpenGLSurface:"), true);

    int attrib [] = new int[] {
        NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core,
        OS.NSOpenGLPFAColorSize, 24,
        OS.NSOpenGLPFADepthSize, 24,
        OS.NSOpenGLPFASampleBuffers, 1,
        OS.NSOpenGLPFASamples, 4,
        OS.NSOpenGLPFADoubleBuffer,
        NSOpenGLPFANoRecovery,
        NSOpenGLPFAAccelerated,
        0
    };

    pixelFormat = (NSOpenGLPixelFormat)new NSOpenGLPixelFormat().alloc();

    if (pixelFormat == null) {
      dispose();
      SWT.error(SWT.ERROR_UNSUPPORTED_DEPTH);
    }
    pixelFormat.initWithAttributes(attrib);

    context = (NSOpenGLContext)new NSOpenGLContext().alloc();
    if (context == null) {
      dispose ();
      SWT.error (SWT.ERROR_UNSUPPORTED_DEPTH);
    }
    context = context.initWithFormat(pixelFormat, null);
    context.setValues(new int[] { -1 }, OS.NSOpenGLCPSurfaceOrder);
    setData(GLCONTEXT_KEY, context);
    NSNotificationCenter.defaultCenter().addObserver(view,
        OS.sel_updateOpenGLContext_, OS.NSViewGlobalFrameDidChangeNotification, view);

    Listener listener = event -> {
      switch (event.type) {
        case SWT.Dispose:
          terminate();
          setData(GLCONTEXT_KEY, null);
          NSNotificationCenter.defaultCenter().removeObserver(view);

          if (context != null) {
            context.clearDrawable();
            context.release();
            context = null;
          }
          if (pixelFormat != null) {
            pixelFormat.release();
            pixelFormat = null;
          }
          break;
      }
    };
    addListener(SWT.Dispose, listener);
  }

  public void setCurrent () {
    checkWidget();
    context.makeCurrentContext();
  }

  public void swapBuffers () {
    checkWidget();
    context.flushBuffer();
  }

  protected abstract void terminate();
}
