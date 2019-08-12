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
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.VirtualTrackSelectionView;
import com.google.gapid.perfetto.views.State;
import java.util.Arrays;
import java.util.HashMap;
import org.eclipse.swt.widgets.Composite;

public class VirtualTrack extends Track<VirtualTrack.Data> {
  private static final String VIEW_SQL =
      "select slice_id, ref, ts, dur, name, arg_set_id from slices where ref = %d and ref_type='track'";
  private static final String SLICE_SQL =
      "select slice_id, ts, dur, name, arg_set_id from slices where slice_id = %d";
  private static final String INT_ARGS_SQL =
      "select flat_key, int_value from args where arg_set_id = %d and int_value is not null";
  private static final String STRING_ARGS_SQL =
      "select flat_key, string_value from args where arg_set_id = %d and string_value is not null";

  private final long id;
  private final String name;

  public VirtualTrack(long id, String name) {
    super("virtual_track_" + id);
    this.id = id;
    this.name = name;
  }

  public String getName() {
    return this.name;
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

  private static String sliceSql(long sliceId) {
    return format(SLICE_SQL, sliceId);
  }

  private static String intArgsSql(long argSetId) {
    return format(INT_ARGS_SQL, argSetId);
  }

  private static String stringArgsSql(long argSetId) {
    return format(STRING_ARGS_SQL, argSetId);
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

  public static ListenableFuture<Slice> getSliceAndArgs(QueryEngine qe, long id) {
    return transformAsync(expectOneRow(qe.query(sliceSql(id))), sliceRow -> {
      long argSetId = sliceRow.getLong(4);
      return transformAsync(qe.query(intArgsSql(argSetId)), intArgRes -> {
        return transform(qe.query(stringArgsSql(argSetId)),
            stringArgsRes -> { return new Slice(sliceRow, intArgRes, stringArgsRes); });
      });
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

  public static class Slice implements Selection {
    public final long sliceId;
    public final long ts;
    public final long dur;
    public final String eventName;
    public HashMap<String, Long> intValues = new HashMap<>();
    public HashMap<String, String> stringValues = new HashMap<>();

    public Slice(
        QueryEngine.Row sliceRow, QueryEngine.Result intArgs, QueryEngine.Result stringArgs) {
      this.sliceId = sliceRow.getLong(0);
      this.ts = sliceRow.getLong(1);
      this.dur = sliceRow.getLong(2);
      this.eventName = sliceRow.getString(3);

      intArgs.forEachRow((i, r) -> {
        String key = r.getString(0);
        long value = r.getLong(1);
        intValues.put(key, value);
      });

      stringArgs.forEachRow((i, r) -> {
        String key = r.getString(0);
        String stringValue = r.getString(1);
        stringValues.put(key, stringValue);
      });
    }

    @Override
    public String getTitle() {
      return "Track Slices";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new VirtualTrackSelectionView(parent, state, this);
    }

    @Override
    public String toString() {
      return "Slice{@" + ts + " +" + dur + " " + sliceId + "}";
    }
  }
}
