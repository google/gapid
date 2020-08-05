/*
 * Copyright (C) 2020 Google Inc.
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

package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.packColumns;

import com.google.gapid.models.Analytics;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.QueryViewer.ResultContentProvider;
import com.google.gapid.perfetto.QueryViewer.Row;
import com.google.gapid.proto.perfetto.Perfetto.QueryResult;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;
import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.TableColumn;

public class TraceMetadataDialog {

  private static final Logger LOG = Logger.getLogger(TraceMetadataDialog.class.getName());
  private static final String QUERY_STR =
      "SELECT name AS 'Name', CAST(COALESCE(int_value, str_value, 'NULL') as TEXT) as 'Value' FROM"
          + " metadata;";

  public static void showMetadata(
      Shell shell, Analytics analytics, Perfetto perfetto, Theme theme) {
    analytics.postInteraction(Analytics.View.QueryMetadata, ClientAction.Show);
    new DialogBase(shell, theme) {
      @Override
      public String getTitle() {
        return Messages.TRACE_METADATA_VIEW_TITLE;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite) super.createDialogArea(parent);
        TableViewer table = createTableViewer(area, SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL);
        table.getTable().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
        table.setContentProvider(new ResultContentProvider());
        table.setLabelProvider(new LabelProvider());
        Rpc.listen(
            perfetto.query(QUERY_STR),
            new UiCallback<QueryResult, QueryResult>(area, LOG) {
              @Override
              protected QueryResult onRpcThread(Result<QueryResult> result)
                  throws ExecutionException {
                try {
                  return result.get();
                } catch (RpcException e) {
                  LOG.log(Level.WARNING, "System Profile Query failure", e);
                  return QueryResult.newBuilder().setError(e.toString()).build();
                }
              }

              @Override
              protected void onUiThread(QueryResult result) {
                table.setInput(null);
                for (TableColumn col : table.getTable().getColumns()) {
                  col.dispose();
                }

                if (!result.getError().isEmpty()) {
                  Widgets.createLabel(area, "Error: " + result.getError());
                  area.requestLayout();
                } else if (result.getNumRecords() == 0) {
                  Widgets.createLabel(area, "Query returned no rows.");
                  area.requestLayout();
                } else {
                  for (int i = 0; i < result.getColumnDescriptorsCount(); i++) {
                    int col = i;
                    QueryResult.ColumnDesc desc = result.getColumnDescriptors(i);
                    Widgets.createTableColumn(
                        table, desc.getName(), row -> ((Row) row).getValue(col));
                    table.setInput(result);
                    packColumns(table.getTable());
                    table.getTable().requestLayout();
                  }
                }
              }
            });
        return area;
      }

      @Override
      protected void createButtonsForButtonBar(Composite parent) {
        createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      }
    }.open();
  }
}
