/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto;

import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTextarea;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.common.collect.Lists;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.proto.perfetto.Perfetto;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.IStructuredContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.TableColumn;
import org.eclipse.swt.widgets.Text;

import java.util.Comparator;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Allows the user to execute manual queries.
 */
public class QueryViewer extends Composite
    implements Capture.Listener, com.google.gapid.models.Perfetto.Listener {
  protected static final Logger LOG = Logger.getLogger(QueryViewer.class.getName());

  private final Models models;
  private final Button run;
  private final Text query;
  protected final TableViewer table;

  public QueryViewer(Composite parent, Models models) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    SashForm splitter = new SashForm(this, SWT.VERTICAL);

    Composite top = createComposite(splitter, new GridLayout(1, false));
    query = withLayoutData(createTextarea(top, "select * from perfetto_tables"),
        new GridData(SWT.FILL, SWT.FILL, true, true));
    run = withLayoutData(createButton(top, "Run", e -> exec()),
        new GridData(SWT.LEFT, SWT.BOTTOM, false, false));

    table = Widgets.createTableViewer(splitter, SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL);
    table.setContentProvider(new ResultContentProvider());
    table.setLabelProvider(new LabelProvider());

    splitter.setWeights(new int[] { 30, 70 });
    run.setEnabled(false);

    models.capture.addListener(this);
    models.perfetto.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.perfetto.removeListener(this);
    });
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    run.setEnabled(false);
  }

  @Override
  public void onPerfettoLoaded(Loadable.Message error) {
    run.setEnabled(error == null);
  }

  private void exec() {
    Rpc.listen(models.perfetto.query(query.getText()),
        new UiCallback<Perfetto.QueryResult, Perfetto.QueryResult>(this, LOG) {
      @Override
      protected Perfetto.QueryResult onRpcThread(
          Result<Perfetto.QueryResult> result) throws ExecutionException {
        try {
          return result.get();
        } catch (RpcException e) {
          LOG.log(Level.WARNING, "Perfetto Query failure", e);
          return Perfetto.QueryResult.newBuilder()
              .setError(e.toString())
              .build();
        }
      }

      @Override
      protected void onUiThread(Perfetto.QueryResult result) {
        table.setInput(null);
        for (TableColumn col : table.getTable().getColumns()) {
          col.dispose();
        }

        if (!result.getError().isEmpty()) {
          Widgets.createTableColumn(table, "Error", $ -> result.getError());
        } else if (result.getNumRecords() == 0) {
          Widgets.createTableColumn(table, "Result", $ -> "Query returned no rows.");
        } else {
          List<Widgets.ColumnAndComparator<Row>> columns = Lists.newArrayList();
          for (int i = 0; i < result.getColumnDescriptorsCount(); i++) {
            int col = i;
            Perfetto.QueryResult.ColumnDesc desc = result.getColumnDescriptors(i);
            columns.add(createTableColumn(
                table, desc.getName(), row -> row.getValue(col), comparator(result, col)));
          }
          Widgets.sorting(table, columns);
        }

        table.setInput(result);
        packColumns(table.getTable());
        table.getTable().requestLayout();
      }
    });
  }

  protected static Comparator<Row> comparator(Perfetto.QueryResult res, int col) {
    Perfetto.QueryResult.ColumnValues vals = res.getColumns(col);
    switch (res.getColumnDescriptors(col).getType()) {
      case DOUBLE: return (r1, r2) -> {
        if (vals.getIsNulls(r1.row)) {
          return vals.getIsNulls(r2.row) ? 0 : -1;
        } else if (vals.getIsNulls(r2.row)) {
          return 1;
        } else {
          return Double.compare(vals.getDoubleValues(r1.row), vals.getDoubleValues(r2.row));
        }
      };
      case LONG: return (r1, r2) -> {
        if (vals.getIsNulls(r1.row)) {
          return vals.getIsNulls(r2.row) ? 0 : -1;
        } else if (vals.getIsNulls(r2.row)) {
          return 1;
        } else {
          return Long.compare(vals.getLongValues(r1.row), vals.getLongValues(r2.row));
        }
      };
      case STRING: return (r1, r2) -> {
        if (vals.getIsNulls(r1.row)) {
          return vals.getIsNulls(r2.row) ? 0 : -1;
        } else if (vals.getIsNulls(r2.row)) {
          return 1;
        } else {
          return vals.getStringValues(r1.row).compareTo(vals.getStringValues(r2.row));
        }
      };
      default: return (r1, r2) -> 0;
    }
  }

  private static class ResultContentProvider implements IStructuredContentProvider {
    public ResultContentProvider() {
    }

    @Override
    public Object[] getElements(Object inputElement) {
      Perfetto.QueryResult result = (Perfetto.QueryResult)inputElement;
      if (result == null) {
        return new Object[0];
      } else if (!result.getError().isEmpty() || result.getNumRecords() == 0) {
        return new Row[] { new Row(result, 0) };
      } else {
        Row[] r = new Row[(int)result.getNumRecords()];
        for (int i = 0; i < r.length; i++) {
          r[i] = new Row(result, i);
        }
        return r;
      }
    }
  }

  private static class Row {
    public final Perfetto.QueryResult result;
    public final int row;

    public Row(Perfetto.QueryResult result, int row) {
      this.result = result;
      this.row = row;
    }

    public String getValue(int column) {
      Perfetto.QueryResult.ColumnValues vals = result.getColumns(column);
      if (vals.getIsNulls(row)) {
        return "NULL";
      }

      switch (result.getColumnDescriptors(column).getType()) {
        case DOUBLE: return Double.toString(vals.getDoubleValues(row));
        case LONG: return Long.toString(vals.getLongValues(row));
        case STRING: return vals.getStringValues(row);
        default: return "???";
      }
    }
  }
}
