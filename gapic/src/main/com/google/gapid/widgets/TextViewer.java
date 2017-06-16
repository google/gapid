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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.util.Loadable.MessageType;
import com.google.gapid.util.Messages;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.MessageDialog;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;

import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Popup displaying text to the user.
 */
public class TextViewer {
  private static final int INITIAL_MIN_HEIGHT = 600;
  protected static final Logger LOG = Logger.getLogger(TextViewer.class.getName());

  private TextViewer() {
  }

  public static void showViewTextPopup(
      Shell shell, Widgets widgets, String title, ListenableFuture<String> text) {
    new MessageDialog(shell, Messages.VIEW_DETAILS, null, title, MessageDialog.INFORMATION, 0,
        IDialogConstants.OK_LABEL) {
      protected LoadablePanel<Text> loadable;

      @Override
      protected boolean isResizable() {
        return true;
      }

      @Override
      protected Control createCustomArea(Composite parent) {
        loadable = new LoadablePanel<Text>(parent, widgets, panel -> new Text(
            panel, SWT.MULTI | SWT.READ_ONLY | SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL));
        loadable.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

        loadable.startLoading();
        Rpc.listen(text, new UiErrorCallback<String, String, String>(parent, LOG) {
          @Override
          protected ResultOrError<String, String> onRpcThread(Rpc.Result<String> result)  {
            try {
              return success(result.get());
            } catch (RpcException e) {
              return error(e.getMessage());
            } catch (ExecutionException e) {
              return error(e.getCause().toString());
            }
          }

          @Override
          protected void onUiThreadSuccess(String result) {
            loadable.getContents().setText(result);
            loadable.stopLoading();
          }

          @Override
          protected void onUiThreadError(String error) {
            loadable.showMessage(MessageType.Error, error);
          }
        });
        return loadable;
      }

      @Override
      protected Point getInitialSize() {
        Point size = super.getInitialSize();
        size.y = Math.max(size.y, INITIAL_MIN_HEIGHT);
        return size;
      }
    }.open();
  }
}
