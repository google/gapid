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

import static java.util.logging.Level.INFO;

import org.eclipse.swt.SWT;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Menu;
import org.eclipse.swt.widgets.MenuItem;

import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * Special handling for OSX application menus.
 */
public class MacApplication {
  private static final Logger LOG = Logger.getLogger(MacApplication.class.getName());

  private static Consumer<String> onOpen = null;
  private static String documentToOpen;

  private MacApplication() {
  }

  public static void listenForOpenDocument(Display display) {
    display.addListener(SWT.OpenDocument, e -> {
      LOG.log(INFO, "OpenDocument Event: " + e);
      if (onOpen != null) {
        onOpen.accept(e.text);
      } else {
        documentToOpen = e.text;
      }
    });
  }

  /**
   * Initializes the OSX application menus.
   */
  public static void init(
      Display display, Runnable onAbout, Runnable onSettings, Consumer<String> newOnOpen) {
    Menu menu = display.getSystemMenu();
    if (menu == null) {
      return;
    }

    for (MenuItem item : menu.getItems()) {
      switch (item.getID()) {
        case SWT.ID_ABOUT:
          item.addListener(SWT.Selection, e -> onAbout.run());
          break;
        case SWT.ID_PREFERENCES:
          item.addListener(SWT.Selection, e -> onSettings.run());
          break;
      }
    }

    onOpen = newOnOpen;
    if (documentToOpen != null) {
      onOpen.accept(documentToOpen);
    }
  }
}
