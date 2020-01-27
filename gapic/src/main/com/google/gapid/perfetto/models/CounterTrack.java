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
package com.google.gapid.perfetto.models;

import static com.google.gapid.perfetto.models.QueryEngine.createSpan;
import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.CountersSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.Set;
import java.util.stream.LongStream;

public class CounterTrack extends Track.WithQueryEngine<CounterTrack.Data> {
  private static final String VIEW_SQL_DELTA =
      "select ts + 1 ts, lead(ts) over win - ts dur, lead(value) over win value " +
      "from counter where track_id = %d window win as (order by ts)";
  private static final String VIEW_SQL_EVENT =
      "select ts, lead(ts) over win - ts dur, value " +
      "from counter where track_id = %d window win as (order by ts)";
  private static final String SUMMARY_SQL =
      "select min(ts), max(ts + dur), avg(value) from %s group by quantum_ts";
  private static final String COUNTER_SQL = "select ts, ts + dur, value from %s";
  private static final String VALUE_SQL = "select ts, ts + dur, value from %s where ts = %d";
  private static final String RANGE_SQL =
      "select ts, ts + dur, value from %s " +
      "where ts + dur >= %d and ts <= %d order by ts";

  private final CounterInfo counter;

  public CounterTrack(QueryEngine qe, CounterInfo counter) {
    super(qe, "counter_" + counter.id);
    this.counter = counter;
  }

  public CounterInfo getCounter() {
    return counter;
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String vals = tableName("vals");
    String span = tableName("span");
    String window = tableName("window");
    return qe.queries(
        dropTable(span),
        dropTable(window),
        dropView(vals),
        createView(vals, viewSql()),
        createWindow(window),
        createSpan(span, vals + ", " + window));
  }

  private String viewSql() {
    switch (counter.interpolation) {
      case Delta: return format(VIEW_SQL_DELTA, counter.id);
      case Event: return format(VIEW_SQL_EVENT, counter.id);
      default: throw new AssertionError();
    }
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window win = (counter.count > Track.QUANTIZE_CUT_OFF) ? Window.compute(req, 5) :
        Window.compute(req);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(req, win));
  }

  private ListenableFuture<Data> computeData(DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql() : counterSQL()), res -> {
      int rows = res.getNumRows();
      if (rows == 0) {
        return Data.empty(req);
      }

      Data data = new Data(req, new long[rows + 1], new double[rows + 1]);
      res.forEachRow((i, r) -> {
        data.ts[i] = r.getLong(0);
        data.values[i] = r.getDouble(2);
      });
      data.ts[rows] = res.getLong(rows - 1, 1, 0);
      data.values[rows] = data.values[rows - 1];
      return data;
    });
  }

  private String summarySql() {
    return format(SUMMARY_SQL, tableName("span"));
  }

  private String counterSQL() {
    return format(COUNTER_SQL, tableName("span"));
  }

  public ListenableFuture<Data> getValue(long t) {
    return transform(expectOneRow(qe.query(valueSql(t))), row -> {
      Data data = new Data(null, new long[1], new double[1]);
      data.ts[0] = row.getLong(0);
      data.values[0] = row.getDouble(2);
      return data;
    });
  }

  public ListenableFuture<Data> getValues(TimeSpan ts) {
    return transform(qe.query(rangeSql(ts)), res -> {
      int rows = res.getNumRows();
      Data data = new Data(null, new long[rows], new double[rows]);
      res.forEachRow((i, r) -> {
        data.ts[i] = r.getLong(0);
        data.values[i] = r.getDouble(2);
      });
      return data;
    });
  }

  private String valueSql(long t) {
    return format(VALUE_SQL, tableName("vals"), t);
  }

  private String rangeSql(TimeSpan ts) {
    return format(RANGE_SQL, tableName("vals"), ts.start, ts.end);
  }

  public static class Data extends Track.Data {
    public final long[] ts;
    public final double[] values;

    public Data(DataRequest request, long[] ts, double[] values) {
      super(request);
      this.ts = ts;
      this.values = values;
    }

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new double[0]);
    }
  }

  public static class Values implements Selection<Values.Key>, Selection.Builder<Values> {
    public final long[] ts;
    public final String[] names;
    public final double[][] values;
    private final Set<Values.Key> valueKeys = Sets.newHashSet();

    public Values(String name, Data data) {
      this.ts = data.ts;
      this.names = new String[] { name };
      this.values = new double[][] { data.values };
      LongStream.of(ts).boxed().forEach(t -> valueKeys.add(new Values.Key(name, t)));
    }

    private Values(long[] ts, String[] names, double[][] values) {
      this.ts = ts;
      this.names = names;
      this.values = values;
      for (String name : names) {
        LongStream.of(ts).boxed().forEach(t -> valueKeys.add(new Values.Key(name, t)));
      }
    }

    @Override
    public String getTitle() {
      return "Counters";
    }

    @Override
    public boolean contains(Values.Key key) {
      return valueKeys.contains(key);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new CountersSelectionView(parent, state, this);
    }

    @Override
    public Selection.Builder getBuilder() {
      return this;
    }

    @Override
    public Values combine(Values other) {
      if (ts.length == 0) {
        return other;
      } else if (other.ts.length == 0) {
        return this;
      }

      long[] newTs = combineTs(ts, other.ts);

      double[][] newValues = new double[names.length + other.names.length][newTs.length];
      for (int i = 0, me = 0, them = 0; i < newTs.length; i++) {
        long rTs = newTs[i], meTs = ts[me], themTs = other.ts[them];
        if (rTs == meTs) {
          for (int n = 0; n < names.length; n++) {
            newValues[n][i] = values[n][me];
          }
          me = Math.min(me + 1, ts.length - 1);
        } else if (i > 0) {
          for (int n = 0; n < names.length; n++) {
            newValues[n][i] = newValues[n][i - 1];
          }
        }

        if (rTs == themTs) {
          for (int n = 0; n < other.names.length; n++) {
            newValues[n + names.length][i] = other.values[n][them];
          }
          them = Math.min(them + 1, other.ts.length - 1);
        } else if (i > 0) {
          for (int n = 0; n < other.names.length; n++) {
            newValues[names.length + n][i] = newValues[names.length + n][i - 1];
          }
        }
      }

      String[] newNames = Arrays.copyOf(names, names.length + other.names.length);
      System.arraycopy(other.names, 0, newNames, names.length, other.names.length);
      return new Values(newTs, newNames, newValues);
    }

    private static long[] combineTs(long[] a, long[] b) {
      long[] r = new long[a.length + b.length];
      int ai = 0, bi = 0, ri = 0;
      for (; ai < a.length && bi < b.length; ri++) {
        long av = a[ai], bv = b[bi];
        if (av == bv) {
          r[ri] = av;
          ai++;
          bi++;
        } else if (av < bv) {
          r[ri] = av;
          ai++;
        } else {
          r[ri] = bv;
          bi++;
        }
      }
      // One of these copies does nothing.
      System.arraycopy(a, ai, r, ri, a.length - ai);
      System.arraycopy(b, bi, r, ri, b.length - bi);
      return Arrays.copyOf(r, ri + a.length - ai + b.length - bi); // Truncate array.
    }

    @Override
    public Selection<Values.Key> build() {
      return this;
    }

    public static class Key {
      public final String name;
      public final long ts;

      public Key(String name, long ts) {
        this.name = name;
        this.ts = ts;
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Key)) {
          return false;
        }
        Key o = (Key)obj;
        return name.equals(o.name) && ts == o.ts;
      }

      @Override
      public int hashCode() {
        return name.hashCode() ^ Long.hashCode(ts);
      }
    }
  }
}
