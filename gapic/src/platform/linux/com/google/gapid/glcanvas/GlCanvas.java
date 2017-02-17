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

import org.eclipse.swt.opengl.GLCanvas;
import org.eclipse.swt.opengl.GLData;
import org.eclipse.swt.widgets.Composite;

// On linux, simply use the SWT GLCanvas as it works fine out of the box.
public class GlCanvas extends GLCanvas {
  public GlCanvas(Composite parent, int style) {
    super(parent, style, getGlData());
  }

  private static GLData getGlData() {
    GLData result = new GLData();
    result.doubleBuffer = true;
    result.sampleBuffers = 1;
    result.samples = 4;
    return result;
  }
}
