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

import com.google.gapid.util.Messages;

import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.DirectoryDialog;
import org.eclipse.swt.widgets.FileDialog;

/**
 * An {@link ActionTextbox} representing a file path where the button will open a file dialog.
 */
public abstract class FileTextbox extends ActionTextbox {
  public FileTextbox(Composite parent, String value) {
    super(parent, Messages.BROWSE, value);
  }

  /**
   * A {@link FileTextbox} limiting the selection to directories in the file dialog.
   */
  public static class File extends FileTextbox {
    public File(Composite parent) {
      super(parent, "");
    }

    public File(Composite parent, String value) {
      super(parent, value);
    }

    @Override
    protected String createAndShowDialog(String current) {
      FileDialog dialog = new FileDialog(getShell());
      if (!current.isEmpty()) {
        java.io.File file = new java.io.File(current);
        String parent = file.getParent();
        if (parent != null) {
          dialog.setFilterPath(parent);
        }
        dialog.setFileName(file.getName());
      }
      configureDialog(dialog);
      return dialog.open();
    }

    @SuppressWarnings("unused")
    protected void configureDialog(FileDialog dialog) {
      // Empty.
    }
  }

  /**
   * A {@link FileTextbox} limiting the selection to directories in the file dialog.
   */
  public static class Directory extends FileTextbox {
    public Directory(Composite parent) {
      super(parent, "");
    }

    public Directory(Composite parent, String value) {
      super(parent, value);
    }

    @Override
    protected String createAndShowDialog(String current) {
      DirectoryDialog dialog = new DirectoryDialog(getShell());
      dialog.setFilterPath(current);
      configureDialog(dialog);
      return dialog.open();
    }

    @SuppressWarnings("unused")
    protected void configureDialog(DirectoryDialog dialog) {
      // Empty.
    }

    @Override
    public String getText() {
      return super.getText().replaceFirst("^~", System.getProperty("user.home"));
    }
  }
}
