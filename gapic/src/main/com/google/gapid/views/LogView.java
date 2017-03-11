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
package com.google.gapid.views;

import com.google.gapid.util.Logging;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.text.Document;
import org.eclipse.jface.text.TextViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.concurrent.atomic.AtomicBoolean;

/**
 * A view that shows log messages.
 */
public class LogView extends Composite implements Tab {
  private final TextViewer text;
  private final AtomicBoolean dirty = new AtomicBoolean(false);

  public LogView(Composite parent) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout(SWT.VERTICAL));

    text = new TextViewer(this, SWT.MULTI | SWT.H_SCROLL | SWT.V_SCROLL);
    text.setEditable(false);
    text.getTextWidget().setFont(JFaceResources.getFont(JFaceResources.TEXT_FONT));
    text.setDocument(new Document());
    updateText();
    Logging.setListener(() -> {
      if (!dirty.getAndSet(true)) {
        Widgets.scheduleIfNotDisposed(this, this::updateText);
      }
    });
    addListener(SWT.Dispose, e -> Logging.setListener(null));
  }

  private void updateText() {
    String message = Logging.getLogMessages();
    text.getDocument().set(message);
    text.setTopIndex(message.length() - 1);
    dirty.set(false);
  }

  @Override
  public Control getControl() {
    return this;
  }
}
