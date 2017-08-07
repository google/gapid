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

import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;

/**
 * Displays status information at the bottom of the main window.
 */
public class StatusBar extends Composite {
  private final Label status;
  private final Link notification;
  private Runnable onNotificationClick;

  public StatusBar(Composite parent) {
    super(parent, SWT.NONE);

    setLayout(withMargin(new GridLayout(2, false), 0, 0));

    status = Widgets.createLabel(this, "");
    status.setLayoutData(new GridData(SWT.LEFT, SWT.FILL, true, false));

    notification = Widgets.createLink(this, "", (e) -> {
      if (onNotificationClick != null) {
        onNotificationClick.run();
      }
    });
    notification.setLayoutData(new GridData(SWT.RIGHT, SWT.FILL, false, false));
  }

  /**
   * Updates the notification to the given text.
   *
   * @param text the notification text.
   * @param onClick the optional notifiction click handler.
   */
  public void setNotification(String text, Runnable onClick) {
    notification.setText((onClick != null) ? "<a>" + text + "</a>" : text);
    onNotificationClick = onClick;
    layout();
  }
}
