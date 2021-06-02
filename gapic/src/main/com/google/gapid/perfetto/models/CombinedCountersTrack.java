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
import static java.util.Arrays.stream;
import static java.util.stream.Collectors.joining;
import static java.util.stream.IntStream.range;

import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;

import java.util.Set;
import java.util.function.BinaryOperator;
import java.util.function.Consumer;

/**
 * {@link Track} that combines multiple counters into a single graph.
 */
public abstract class CombinedCountersTrack
    <D extends CombinedCountersTrack.Data, V extends CombinedCountersTrack.Values<V>>
    extends Track.WithQueryEngine<D> {
  private final boolean needQuantize;
  private final String viewSql;
  private final String summarySql;
  private final String dataSql;
  private final String valueSql;
  private final String rangeSql;

  public CombinedCountersTrack(
      QueryEngine qe, String trackId, Column[] columns, boolean needQuantize) {
    super(qe, trackId);
    if (stream(columns).allMatch(Column::isNil)) {
      throw new IllegalArgumentException("At least one counter needs to be non-nil");
    }

    this.needQuantize = needQuantize;
    this.viewSql = buildViewSql(columns);
    this.summarySql = buildSummarySql(columns, tableName("span"));
    this.dataSql = buildDataSql(columns, tableName("span"));
    this.valueSql = buildValueSql(columns, tableName("vals"));
    this.rangeSql = buildRangeSql(columns, tableName("vals"));
  }

  private static String buildViewSql(Column[] columns) {
    // select id, ts, lead(ts, 1, <end>) over win - ts dur, [last_non_null(v_<#>) over win v_<#>]+
    // from (
    //   select min(id) id, ts, [max(case when track_id = <id> then value else null end) v_<#>]+
    //   from counter where track_id in ([<id>]+)
    //   group by ts)
    // window win as (order by ts)

    StringBuilder sb = new StringBuilder()
        .append("select id, ts, ")
        .append("lead(ts, 1, (select end_ts from trace_bounds)) over win - ts dur");
    for (int i = 0; i < columns.length; i++) {
      if (columns[i].isNil()) {
        sb.append(", ").append(columns[i].sqlDefault).append(" v_").append(i);
      } else {
        sb.append(", last_non_null(v_").append(i).append(") over win v_").append(i);
      }
    }
    sb.append(" from (select min(id) id, ts");
    for (int i = 0; i < columns.length; i++) {
      Column c = columns[i];
      if (!c.isNil()) {
        sb.append(", max(case when track_id = ").append(c.trackId).append(" then ")
            .append("cast(value as ").append(c.sqlType).append(") else null end) v_").append(i);
      }
    }
    sb.append(" from counter where track_id in ");
    sb.append(stream(columns)
        .filter(c -> !c.isNil())
        .map(c -> Long.toString(c.trackId))
        .collect(joining(", ", "(", ")")));
    sb.append(" group by ts) window win as (order by ts)");
    return sb.toString();
  }

  private static String buildSummarySql(Column[] columns, String tableName) {
    // select id, min(start), max(end), [v_<#>]+
    // from (
    //   select min(id) id, min(ts) start, max(ts + dur) end, [v_<#>]+
    //   from <tableName>
    //   group by quantum_ts)
    // group by id

    StringBuilder sb = new StringBuilder();
    sb.append("select id, min(start), max(end)");
    for (int i = 0; i < columns.length; i++) {
      Column c = columns[i];
      sb.append(", cast(")
          .append(c.sqlAggregate).append("(")
          .append("v_").append(i).append(") ")
          .append("as ").append(c.sqlType).append(")");
    }
    sb.append(" from (");
    sb.append("select min(id) id, min(ts) start, max(ts + dur) end");
    for (int i = 0; i < columns.length; i++) {
      Column c = columns[i];
      sb.append(", cast(")
          .append(c.sqlAggregate).append("(")
          .append("v_").append(i).append(") ")
          .append("as ").append(c.sqlType).append(") v_").append(i);
    }
    sb.append(" from ").append(tableName).append(" group by quantum_ts) group by id");
    return sb.toString();
  }

  private static String buildDataSql(Column[] columns, String span) {
    // select id, ts, ts + dur, [coalesce(v_<#>, <default>)]+ from <span>

    StringBuilder sb = new StringBuilder()
        .append("select id, ts, ts + dur");
    for (int i = 0; i < columns.length; i++) {
      sb.append(", coalesce(v_").append(i).append(", ").append(columns[i].sqlDefault).append(")");
    }
    sb.append(" from ").append(span);
    return sb.toString();
  }

  private static String buildValueSql(Column[] columns, String tableName) {
    // select id, ts, dur, [v_<#>]+ from <table> where id = ?

    StringBuilder sb = new StringBuilder("select id, ts, dur");
    for (int i = 0; i < columns.length; i++) {
      sb.append(", v_").append(i);
    }
    sb.append(" from ").append(tableName).append(" where id = %d");
    return sb.toString();
  }

  private static String buildRangeSql(Column[] columns, String tableName) {
    // select id, ts, dur, [v_<#>]+ from <table> where ts + dur >= ? and ts <= ? order by ts

    StringBuilder sb = new StringBuilder("select id, ts, dur");
    for (int i = 0; i < columns.length; i++) {
      sb.append(", v_").append(i);
    }
    sb.append(" from ").append(tableName)
        .append(" where ts + dur >= %d and ts <= %d order by ts");
    return sb.toString();
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
        createView(vals, viewSql),
        createWindow(window),
        createSpan(span, vals + ", " + window));
  }

  @Override
  protected ListenableFuture<D> computeData(DataRequest req) {
    Window win = needQuantize ? Window.compute(req, 5) : Window.compute(req);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(req, win));
  }

  private ListenableFuture<D> computeData(DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql : dataSql), res -> {
      int rows = res.getNumRows();
      if (rows == 0) {
        return createData(req, 0);
      }

      D data = createData(req, rows + 1);
      res.forEachRow(data::set);
      data.copyRow(res.getLong(rows - 1, 2, 0), rows - 1, rows);
      return data;
    });
  }

  protected abstract D createData(DataRequest request, int numRows);

  public ListenableFuture<V> getValue(long id) {
    return transform(expectOneRow(qe.query(valueSql(id))), row -> {
      V v = createValues(1, this::combine);
      v.set(0, row);
      return v;
    });
  }

  private String valueSql(long id) {
    return format(valueSql, id);
  }

  public ListenableFuture<V> getValues(TimeSpan ts) {
    return transform(qe.query(rangeSql(ts)), res -> {
      V v = createValues(res.getNumRows(), this::combine);
      res.forEachRow(v::set);
      return v;
    });
  }

  private String rangeSql(TimeSpan ts) {
    return format(rangeSql, ts.start, ts.end);
  }

  protected abstract V createValues(int numRows, BinaryOperator<V> combiner);

  private V combine(V a, V b) {
    if (a.ts.length == 0) {
      return b;
    } else if (b.ts.length == 0) {
      return a;
    }

    long[] ts = new long[a.ts.length + b.ts.length];
    int count = combineTs(a.ts, b.ts, ts);
    V v = createValues(count, this::combine);
    System.arraycopy(ts, 0, v.ts, 0, count);

    int ai = 0, bi = 0, ri = 0;
    for (; ai < a.ts.length && bi < b.ts.length; ri++) {
      long ats = a.ts[ai], bts = b.ts[bi], rts = v.ts[ri];
      if (rts == ats) {
        v.copyFrom(a, ai, ri);
        ai++;
        if (rts == bts) {
          bi++;
        }
      } else {
        v.copyFrom(b, bi, ri);
        bi++;
      }
    }

    for (; ai < a.ts.length; ai++, ri++) {
      v.copyFrom(a, ai, ri);
    }
    for (; bi < b.ts.length; bi++, ri++) {
      v.copyFrom(b, bi, ri);
    }

    for (int i = 1; i < count; i++) {
      v.dur[i - 1] = v.ts[i] - v.ts[i - 1];
    }
    v.dur[count - 1] = Math.max(
        a.ts[a.ts.length - 1] + a.dur[a.dur.length - 1],
        b.ts[b.ts.length - 1] + b.dur[b.dur.length - 1]) - v.ts[count - 1];

    v.combineIds(a);
    v.combineIds(b);
    return v;
  }

  private static int combineTs(long[] a, long[] b, long[] r) {
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
    return ri + (a.length - ai) + (b.length - bi);
  }


  public static class Data extends Track.Data {
    protected static final int FIRST_DATA_COLUMN = 3;

    public final long[] id;
    public final long[] ts;

    public Data(DataRequest request, int numRows) {
      super(request);
      this.id = new long[numRows];
      this.ts = new long[numRows];
    }

    public void set(int idx, QueryEngine.Row row) {
      id[idx] = row.getLong(0);
      ts[idx] = row.getLong(1);
    }

    public void copyRow(long time, int src, int dst) {
      id[dst] = id[src];
      ts[dst] = time;
    }
  }

  public abstract static class Values<T extends Values<T>> implements Selection<T> {
    protected static final int FIRST_DATA_COLUMN = 3;

    public final long[] ts;
    public final long[] dur;
    private final Set<Long> valueKeys;
    private final BinaryOperator<T> combiner;

    public Values(int numRows, BinaryOperator<T> combiner) {
      this.ts = new long[numRows];
      this.dur = new long[numRows];
      this.valueKeys = Sets.newHashSet();
      this.combiner = combiner;
    }

    public void combineIds(Values<?> other) {
      valueKeys.addAll(other.valueKeys);
    }

    public void set(int idx, QueryEngine.Row row) {
      valueKeys.add(row.getLong(0));
      ts[idx] = row.getLong(1);
      dur[idx] = row.getLong(2);
    }

    public abstract void copyFrom(T other, int src, int dst);

    @Override
    public boolean contains(Long key) {
      return valueKeys.contains(key);
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      long min = stream(ts).min().getAsLong();
      long max = range(0, ts.length).mapToLong(i -> ts[i] + dur[i]).max().getAsLong();
      span.accept(new TimeSpan(min, max));
    }

    @Override
    @SuppressWarnings("unchecked")
    public T combine(T that) {
      return combiner.apply((T)this, that);
    }
  }

  protected static class Column {
    public final long trackId;
    public final String sqlType;
    public final String sqlDefault;
    public final String sqlAggregate;

    public Column(long trackId, String sqlType, String sqlDefault, String sqlAggregate) {
      this.trackId = trackId;
      this.sqlType = sqlType;
      this.sqlDefault = sqlDefault;
      this.sqlAggregate = sqlAggregate;
    }

    public static Column nil(String sqlDefault) {
      return new Column(-1, "", sqlDefault, "max");
    }

    public boolean isNil() {
      return trackId < 0;
    }
  }
}
