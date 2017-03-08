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
package com.google.gapid.util;

import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.MouseListener;
import org.eclipse.swt.events.MouseMoveListener;
import org.eclipse.swt.events.MouseTrackListener;
import org.eclipse.swt.events.MouseWheelListener;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.events.SelectionListener;

public class MouseAdapter implements MouseListener, MouseMoveListener, MouseWheelListener,
    MouseTrackListener, SelectionListener {
  @Override
  public void mouseEnter(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseExit(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseHover(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseScrolled(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseMove(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseDoubleClick(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseDown(MouseEvent e) {
    // Empty.
  }

  @Override
  public void mouseUp(MouseEvent e) {
    // Empty.
  }

  @Override
  public void widgetDefaultSelected(SelectionEvent e) {
    // Empty.
  }

  @Override
  public void widgetSelected(SelectionEvent e) {
    // Empty.
  }
}
