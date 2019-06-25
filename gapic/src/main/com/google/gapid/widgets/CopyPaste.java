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
package com.google.gapid.widgets;

import com.google.gapid.util.Events;

import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyledText;
import org.eclipse.swt.dnd.Clipboard;
import org.eclipse.swt.dnd.TextTransfer;
import org.eclipse.swt.dnd.Transfer;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Text;

/**
 * Clipboard helper to handle copy-paste operations on various widgets.
 */
public class CopyPaste {
  private static final String SOURCE_DATA_KEY = CopyPaste.class.getName() + ".source";

  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private final Display display;
  private final Clipboard clipboard;
  private final org.eclipse.swt.widgets.Listener focusListener;
  private boolean copyState = false;

  public CopyPaste(Display display) {
    this.display = display;
    this.clipboard = new Clipboard(display);
    this.focusListener = e -> updateCopyState();

    display.addFilter(SWT.FocusIn, focusListener);
    display.addFilter(SWT.FocusOut, focusListener);
  }

  public void dispose() {
    if (!display.isDisposed()) {
      display.removeFilter(SWT.FocusIn, focusListener);
      display.removeFilter(SWT.FocusOut, focusListener);
      clipboard.dispose();
    }
  }

  public void registerCopySource(Control focusReceiver, CopySource source) {
    focusReceiver.setData(SOURCE_DATA_KEY, source);
  }

  public void updateCopyState() {
    boolean newState = getCurrentCopier().hasCopyData();
    if (newState != copyState) {
      copyState = newState;
      listeners.fire().onCopyEnabled(copyState);
    }
  }

  public void doCopy() {
    getCurrentCopier().copy();
  }

  private Copier getCurrentCopier() {
    Control focus = display.getFocusControl();
    if (focus == null) {
      return Copier.NULL_COPIER;
    }

    CopySource source = (CopySource)focus.getData(SOURCE_DATA_KEY);
    if (source == null) {
      if (focus instanceof Text || focus instanceof StyledText) {
        return new Copier() {
          @Override
          public boolean hasCopyData() {
            return true;
          }

          @Override
          public void copy() {
            if (focus instanceof Text) {
              ((Text)focus).copy();
            } else if (focus instanceof StyledText) {
              ((StyledText)focus).copy();
            }
          }
        };
      }
      return Copier.NULL_COPIER;
    }

    return new Copier() {
      @Override
      public boolean hasCopyData() {
        return source.hasCopyData();
      }

      @Override
      public void copy() {
        doCopy(source);
      }
    };
  }

  protected void doCopy(CopySource source) {
    if (!source.hasCopyData()) {
      return;
    }

    CopyData[] data = source.getCopyData();
    if (data.length == 0) {
      return;
    }

    Object[] objs = new Object[data.length];
    Transfer[] transfers = new Transfer[data.length];
    for (int i = 0; i < data.length; i++) {
      objs[i] = data[i].data;
      transfers[i] = data[i].transfer;
    }
    clipboard.setContents(objs, transfers);
  }

  public void setContents(String s) {
    clipboard.setContents(new Object[]{s}, new Transfer[]{TextTransfer.getInstance()});
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  /**
   * A source from which data can be copied to the clipboard.
   */
  public static interface CopySource {
    /**
     * @return whether the source currently contains any copiable (selected) data.
     */
    public boolean hasCopyData();

    /**
     * @return the current copiable data.
     */
    public CopyData[] getCopyData();
  }

  /**
   * Data to be copied to the clipboard.
   */
  public static class CopyData {
    public final Object data;
    public final Transfer transfer;

    public CopyData(Object data, Transfer transfer) {
      this.data = data;
      this.transfer = transfer;
    }

    public static CopyData text(String text) {
      return new CopyData(text, TextTransfer.getInstance());
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event that indicates whether the copy action (such as in the menu) should be enabled.
     */
    public default void onCopyEnabled(boolean enabled) { /* empty */ }
  }

  /**
   * Handles copying to the clipboard.
   */
  private static interface Copier {
    public static final Copier NULL_COPIER = new Copier() {
      @Override
      public boolean hasCopyData() {
        return false;
      }

      @Override
      public void copy() {
        // Do nothing.
      }
    };

    /**
     * @return whether the copier currently contains any copiable data.
     */
    public boolean hasCopyData();

    /**
     * Performs the copy operation.
     */
    public void copy();
  }
}
