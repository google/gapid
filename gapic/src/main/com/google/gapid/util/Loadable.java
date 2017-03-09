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

/**
 * Widget mixin for widgets that show loadable data. Typical implementations should show a loading
 * animation to the user while the data is loading.
 */
public interface Loadable {
  public static final Loadable NULL_LOADABLE = new Loadable() { /* empty */ };

  /**
   * Indicates that the data is being loaded.
   */
  public default void startLoading() { /* empty */ }

  /**
   * Indicates that the data has finished loading.
   */
  public default void stopLoading() { /* empty */ }

  /**
   * Shows a message to the user instead of the loading animation.
   */
  @SuppressWarnings("unused")
  public default void showMessage(MessageType type, String text) { /* empty */ }

  public static enum MessageType {
    Info, Error;
  }
}
