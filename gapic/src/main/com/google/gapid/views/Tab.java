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

import org.eclipse.swt.widgets.Control;

/**
 * A tab in the main ui.
 */
public interface Tab {
  /**
   * Reinitializes this tab from the current state of the models. Called if the tab was created
   * after the UI has already been visible for some time.
   */
  public default void reinitialize() { /* do nothing */ }

  /**
   * @return the {@link Control} for this tab.
   */
  public Control getControl();
}
