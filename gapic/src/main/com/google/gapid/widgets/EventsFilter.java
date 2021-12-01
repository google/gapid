/*
 * Copyright (C) 2021 Google Inc.
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

import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.gapid.util.Events;

import org.eclipse.swt.SWT;
import org.eclipse.swt.events.SelectionAdapter;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.ToolItem;

import java.util.regex.Pattern;
import java.util.regex.PatternSyntaxException;

/**
 * A filter panel for the events display.
 */
public class EventsFilter extends Composite {
  /**
   * @param parent the parent {@link Composite}
   */

  public EventsFilter(Composite parent) {
    super(parent, SWT.NONE);
    setLayout(new GridLayout(3, false));

    Button hideHostCommands = Widgets.createCheckbox(this, "Hide Host Commands", true);
    hideHostCommands.setLayoutData(new GridData(SWT.RIGHT, SWT.CENTER, false, false));

    Button hideBeginEnd = Widgets.createCheckbox(this, "Hide Begin/End", true);
    hideBeginEnd.setLayoutData(new GridData(SWT.RIGHT, SWT.CENTER, false, false));

    Button hideDeviceSync = Widgets.createCheckbox(this, "Hide Device Sync", true);
    hideDeviceSync.setLayoutData(new GridData(SWT.RIGHT, SWT.CENTER, false, false));

    hideHostCommands.addListener(SWT.Selection, e -> update(hideHostCommands.getSelection(), hideBeginEnd.getSelection(), hideDeviceSync.getSelection()));
    hideBeginEnd.addListener(SWT.Selection, e -> update(hideHostCommands.getSelection(), hideBeginEnd.getSelection(), hideDeviceSync.getSelection()));
    hideDeviceSync.addListener(SWT.Selection, e -> update(hideHostCommands.getSelection(), hideBeginEnd.getSelection(), hideDeviceSync.getSelection()));
  }

  private void update(Boolean hideHostCommands, Boolean hideBeginEnd, Boolean hideDeviceSync) {
    notifyListeners(Events.FilterEvents, Events.newFilterEventsEvent(EventsFilter.this, hideHostCommands, hideBeginEnd, hideDeviceSync));
  }
}
