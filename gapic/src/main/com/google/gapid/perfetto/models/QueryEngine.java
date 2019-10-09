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

import static com.google.common.util.concurrent.Futures.immediateFailedFuture;
import static com.google.common.util.concurrent.Futures.immediateFuture;
import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.proto.perfetto.Perfetto;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.FutureCache;
import com.google.gapid.util.Scheduler;
import com.google.gapid.views.StatusBar;

import java.util.AbstractMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.SortedMap;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Function;
import java.util.logging.Logger;
import java.util.stream.IntStream;
import java.util.stream.LongStream;
import java.util.stream.Stream;

/**
 * Interface to the trace processor query executor.
 */
public class QueryEngine {
  private static final Logger LOG = Logger.getLogger(QueryEngine.class.getName());

  private static final String TIMESPAN_QUERY = "select start_ts, end_ts from trace_bounds";
  private static final String NUM_CPUS_QUERY = "select count(distinct(cpu)) as c from sched";

  private final Client client;
  private final Path.Capture capture;
  private final StatusBar status;
  private final FutureCache<Long, ArgSet> argsCache;
  private final AtomicInteger scheduled = new AtomicInteger(0);
  private final AtomicInteger done = new AtomicInteger(0);
  private final AtomicBoolean updating = new AtomicBoolean(false);

  public QueryEngine(Client client, Path.Capture capture, StatusBar status) {
    this.client = client;
    this.capture = capture;
    this.status = status;
    this.argsCache = FutureCache.softCache(key -> ArgSet.get(this, key), Objects::nonNull);
  }

  public ListenableFuture<Perfetto.QueryResult> raw(String sql) {
    scheduled.incrementAndGet();
    updateStatus();
    return transform(client.perfettoQuery(capture, sql), r -> {
      done.incrementAndGet();
      updateStatus();
      return r;
    });
  }

  public ListenableFuture<Result> query(String sql) {
    return transformAsync(raw(sql), r -> {
      if (!r.getError().isEmpty()) {
        return immediateFailedFuture(new RpcException("Query failed: " + r.getError()));
      }
      return immediateFuture(new Result(r));
    });
  }

  public ListenableFuture<Result> queries(String... sql) {
    return query(sql, 0);
  }

  private ListenableFuture<Result> query(String[] queries, int idx) {
    return transformAsync(query(queries[idx]), result -> {
      if (idx + 1 >= queries.length) {
        return Futures.immediateFuture(result);
      }
      return query(queries, idx + 1);
    });
  }

  public ListenableFuture<ArgSet> getArgs(long id) {
    return argsCache.get(id);
  }

  public ListenableFuture<Map<Long, ArgSet>> getAllArgs(LongStream ids) {
    return transform(
        Futures.allAsList(ids
          .distinct()
          .mapToObj(id -> transform(
              getArgs(id), r -> new AbstractMap.SimpleImmutableEntry<Long, ArgSet>(id, r)))
          .collect(toList())),
        list -> ImmutableMap.copyOf(list));
  }

  public static ListenableFuture<Row> expectOneRow(ListenableFuture<Result> future) {
    return transformAsync(future, r -> {
      if (r.getNumRows() != 1) {
        return immediateFailedFuture(
            new RpcException("Expected a single result row, got " + r.getNumRows()));
      }
      return immediateFuture(r.getRow(0));
    });
  }

  public ListenableFuture<TimeSpan> getTraceTimeBounds() {
    return transform(expectOneRow(query(TIMESPAN_QUERY)),
        r -> new TimeSpan(r.getLong(0), r.getLong(1)));
  }

  public ListenableFuture<Integer> getNumberOfCpus() {
    return transform(expectOneRow(query(NUM_CPUS_QUERY)), r -> r.getInt(0));
  }

  public static String dropTable(String name) {
    return "drop table if exists " + name;
  }

  public static String dropView(String name) {
    return "drop view if exists " + name;
  }

  public static String createTable(String name, String using) {
    return "create virtual table " + name + " using " + using;
  }

  public static String createWindow(String name) {
    return createTable(name, "window");
  }

  public static String createSpan(String name, String params) {
    return createTable(name, "span_join(" + params + ")");
  }

  public static String createSpanLeftJoin(String name, String params) {
    return createTable(name, "span_left_join(" + params + ")");
  }

  public static String createView(String name, String as) {
    return "create view " + name + " as " + as;
  }

  private void updateStatus() {
    if (updating.compareAndSet(false, true)) {
      scheduleIfNotDisposed(status, () -> {
        updating.set(false);
        int d = done.get(), s = scheduled.get();
        if (s == 0) {
          status.setServerStatusPrefix("");
        } else {
          status.setServerStatusPrefix("Queries: " + d + "/" + s);
        }

        if (s != 0 && d == s) {
          logFailure(LOG, Scheduler.EXECUTOR.schedule(() -> {
            int dd = done.get();
            if (scheduled.compareAndSet(dd, 0)) {
              done.updateAndGet(x -> x - dd);
              updateStatus();
            }
          }, 250, MILLISECONDS));
        }
      });
    }
  }

  public static class Result {
    private final Perfetto.QueryResult res;

    public Result(Perfetto.QueryResult res) {
      this.res = res;
    }

    public int getNumRows() {
      return (int)res.getNumRecords();
    }

    public Row getRow(int row) {
      return new Row() {
        @Override
        public boolean isNull(int column) {
          return Result.this.isNull(row, column);
        }

        @Override
        public long getLong(int column, long deflt) {
          return Result.this.getLong(row, column, deflt);
        }

        @Override
        public double getDouble(int column, double deflt) {
          return Result.this.getDouble(row, column, deflt);
        }

        @Override
        public String getString(int column, String deflt) {
          return Result.this.getString(row, column, deflt);
        }
      };
    }

    public void forEachRow(Row.Visitor visitor) {
      for (int i = 0; i < res.getNumRecords(); i++) {
        visitor.visit(i, getRow(i));
      }
    }

    public <K, V> Map<K, V> map(Function<Row, K> key, Function<Row, V> value) {
      Map<K, V> map = Maps.newHashMap();
      forEachRow(($, row) -> map.put(key.apply(row), value.apply(row)));
      return map;
    }

    public <K extends Comparable<K>, V> SortedMap<K, V> sortedMap(
        Function<Row, K> key, Function<Row, V> value) {
      SortedMap<K, V> map = Maps.newTreeMap();
      forEachRow(($, row) -> map.put(key.apply(row), value.apply(row)));
      return map;
    }

    public <V> List<V> list(Row.Callable<V> value) {
      List<V> list = Lists.newArrayListWithCapacity(getNumRows());
      forEachRow((i, row) -> list.add(value.call(i, row)));
      return list;
    }

    public Stream<Row> stream() {
      return IntStream.range(0, getNumRows()).mapToObj(this::getRow);
    }

    public boolean isNull(int row, int column) {
      return res.getColumns(column).getIsNulls(row);
    }

    public long getLong(int row, int column, long deflt) {
      Perfetto.QueryResult.ColumnValues c = res.getColumns(column);
      return (c.getIsNulls(row)) ? deflt : c.getLongValues(row);
    }

    public double getDouble(int row, int column, double deflt) {
      Perfetto.QueryResult.ColumnValues c = res.getColumns(column);
      return (c.getIsNulls(row)) ? deflt : c.getDoubleValues(row);
    }

    public String getString(int row, int column, String deflt) {
      Perfetto.QueryResult.ColumnValues c = res.getColumns(column);
      return (c.getIsNulls(row)) ? deflt : c.getStringValues(row);
    }
  }

  public static interface Row {
    public boolean isNull(int column);
    public default long getLong(int column) { return getLong(column, 0); }
    public long getLong(int column, long deflt);
    public default int getInt(int column) { return (int)getLong(column, 0); }
    public default int getInt(int column, int deflt)  { return (int)getLong(column, deflt); }
    public default double getDouble(int column) { return getDouble(column, 0); }
    public double getDouble(int column, double deflt);
    public default String getString(int column) { return getString(column, ""); }
    public String getString(int column, String deflt);

    public static interface Visitor {
      public void visit(int idx, Row row);
    }

    public static interface Callable<V> {
      public V call(int idx, Row row);
    }
  }
}
