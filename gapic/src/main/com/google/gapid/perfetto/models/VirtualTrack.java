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

import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import java.util.Arrays;

public class VirtualTrack extends Track<VirtualTrack.Data> {
  private static final String VIEW_SQL =
      "select slice_id, ref, ts, dur, name, arg_set_id from slices where ref = %d and ref_type='track'";
  private static final String RANGE_SQL = "select ts, dur from %s "
      + "where ts + dur >= %d and ts < %d";

  private final long id;
  private final String name;

  public VirtualTrack(long id, String name) {
    super("virtual_track_" + id);
    this.id = id;
    this.name = name;
  }

  @Override
  protected ListenableFuture<?> initialize(QueryEngine qe) {
    String slices = tableName("slices");
    String window = tableName("window");
    return qe.queries(
        dropTable(window), dropView(slices), createView(slices, viewSql()), createWindow(window));
  }

  private String viewSql() {
    return format(VIEW_SQL, id);
  }

  private String rangeSql(TimeSpan ts) {
    return format(RANGE_SQL, tableName("slices"), ts.start, ts.end);
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window win = Window.compute(req, 5);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(qe, req, win));
  }

  private ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req, Window win) {
    return transform(qe.query(viewSql()), res -> {
      int rows = res.getNumRows();
      if (rows == 0) {
        return Data.empty(req);
      }

      Data data = new Data(
          req, new long[rows], new long[rows], new long[rows], new long[rows], new String[rows]);
      res.forEachRow((i, r) -> {
        data.sliceId[i] = r.getLong(0);
        data.ref[i] = r.getLong(1);
        data.ts[i] = r.getLong(2);
        data.dur[i] = r.getLong(3);
        data.eventNames[i] = r.getString(4);
      });
      return data;
    });
  }

  public ListenableFuture<Data> getValues(QueryEngine qe, TimeSpan ts) {
    return transform(qe.query(rangeSql(ts)), res -> {
      int rows = res.getNumRows();
      Data data = new Data(
          null, new long[rows], new long[rows], new long[rows], new long[rows], new String[rows]);
      res.forEachRow((i, r) -> {
        data.sliceId[i] = r.getLong(0);
        data.ref[i] = r.getLong(1);
        data.ts[i] = r.getLong(2);
        data.dur[i] = r.getLong(3);
        data.eventNames[i] = r.getString(4);
      });
      return data;
    });
  }

  public static class Data extends Track.Data {
    public final String trackName = "unknown_track";
    public final long[] sliceId;
    public final long[] ref;
    public final long[] ts;
    public final long[] dur;
    public final String[] eventNames;

    public Data(DataRequest request, long[] sliceId, long[] ref, long[] ts, long[] dur,
        String[] eventNames) {
      super(request);
      this.sliceId = sliceId;
      this.ref = ref;
      this.ts = ts;
      this.dur = dur;
      this.eventNames = eventNames;
    }

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new long[0], new long[0], new long[0], new String[0]);
    }
  }
}
