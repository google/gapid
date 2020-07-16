/*
 * Copyright (C) 2018 Google Inc.
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
package com.google.gapid.models;

import static com.google.gapid.glviewer.Geometry.isPolygon;
import static com.google.gapid.models.DeviceDependentModel.Source.withSource;
import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.Paths.meshAfter;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Objects;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.glviewer.geo.Model;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.server.Client.InvalidArgumentException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Streams;
import com.google.protobuf.ByteString;

import org.eclipse.swt.widgets.Shell;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.nio.FloatBuffer;
import java.util.Arrays;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * Model responsible for loading draw call geometry data.
 */
public class Geometries
    extends DeviceDependentModel<Geometries.Data, Geometries.Source, Loadable.Message, Geometries.Listener> {
  private static final Logger LOG = Logger.getLogger(Geometries.class.getName());

  private static final Stringtable.Msg NO_MESH_ERR = Strings.create("ERR_MESH_NOT_AVAILABLE");
  protected static final Vertex.BufferFormat POS_NORM_XYZ_F32 = Vertex.BufferFormat.newBuilder()
      .addStreams(Vertex.StreamFormat.newBuilder()
          .setSemantic(Vertex.Semantic.newBuilder()
              .setType(Vertex.Semantic.Type.Position)
              .setIndex(0))
          .setFormat(Streams.FMT_XYZ_F32))
      .addStreams(Vertex.StreamFormat.newBuilder()
          .setSemantic(Vertex.Semantic.newBuilder()
              .setType(Vertex.Semantic.Type.Normal)
              .setIndex(0))
          .setFormat(Streams.FMT_XYZ_F32))
      .build();

  public Geometries(
      Shell shell, Analytics analytics, Client client, Devices devices, CommandStream commands) {
    super(LOG, shell, analytics, client, Listener.class, devices);

    commands.addListener(new CommandStream.Listener() {
      @Override
      public void onCommandsSelected(CommandIndex selection) {
        load(withSource(getSource(), new Source(selection, null)), false);
      }
    });
  }

  /**
   * Reloads the models using the provided semantics. Has no effect if no command is currently
   * selected or if the semantics haven't changed.
   */
  public void updateSemantics(VertexSemantics semantics) {
    load(Source.withSemantics(getSource(), semantics), false);
  }

  public VertexSemantics getSemantics() {
    if (isLoaded()) {
      return getData().semantics;
    }
    DeviceDependentModel.Source<Source> src = getSource();
    return (src == null || src.source == null) ? null : src.source.semantics;
  }

  @Override
  protected boolean isSourceComplete(DeviceDependentModel.Source<Source> source) {
    return super.isSourceComplete(source) && source.source.command != null;
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source s, Path.Device device) {
    return MoreFutures.transformAsync(fetchMeshMetadata(device, s.command, s.semantics), semantics -> {
      ListenableFuture<Model> originalFuture = fetchModel(device, meshAfter(
          s.command, semantics.getOptions().build(), POS_NORM_XYZ_F32));
      ListenableFuture<Model> facetedFuture = fetchModel(device, meshAfter(
          s.command, semantics.getOptions().setFaceted(true).build(), POS_NORM_XYZ_F32));
      return MoreFutures.combine(Arrays.asList(originalFuture, facetedFuture), models -> {
        MoreFutures.Result<Model> original = models.get(0);
        MoreFutures.Result<Model> faceted = models.get(1);
        if (original.hasFailed() && faceted.hasFailed()) {
          // Both failed, so get the error from the original model's call.
          throw original.error;
        } else if (original.succeeded() && original.result.getNormals() == null) {
          // The original model has no normals, but the geometry doesn't support faceted normals.
          return new Data(device, semantics, null, faceted.result);
        } else if (original.succeeded() && !isPolygon(original.result.getPrimitive())) {
          // TODO: if gapis returns an error for the faceted request, this is not needed.
          return new Data(device, semantics, null, original.result);
        } else {
          return new Data(device, semantics, original.result, faceted.result);
        }
      });
    });
  }

  @Override
  protected ResultOrError<Data, Loadable.Message> processResult(Rpc.Result<Data> result) {
    try {
      return success(result.get());
    } catch (DataUnavailableException e) {
      DeviceDependentModel.Source<Source> s = getSource();
      // TODO: don't assume that it's because of not selecting a draw call.
      return success(new Data(s.device, s.source.semantics, null, null));
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load the geometry", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(LOG, "Failed to load the geometry", e);
      }
      return error(Loadable.Message.error("Failed to load the geometry"));
    }
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onGeometryLoaded(error);
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onGeometryLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onGeometryLoaded(null);
  }

  private ListenableFuture<VertexSemantics> fetchMeshMetadata(
      Path.Device device, CommandIndex command, VertexSemantics currentSemantics) {
    return MoreFutures.transform(client.get(meshAfter(command, Paths.NODATA_MESH_OPTIONS), device),
        value -> new VertexSemantics(value.getMesh(), currentSemantics));
  }

  private ListenableFuture<Model> fetchModel(Path.Device device, Path.Any path) {
    return MoreFutures.transformAsync(client.get(path, device), value -> fetchModel(value.getMesh()));
  }

  private static ListenableFuture<Model> fetchModel(API.Mesh mesh) {
    Vertex.Buffer vb = mesh.getVertexBuffer();
    float[] positions = null;
    float[] normals = null;

    for (Vertex.Stream stream : vb.getStreamsList()) {
      switch (stream.getSemantic().getType()) {
        case Position:
          positions = byteStringToFloatArray(stream.getData());
          break;
        case Normal:
          normals = byteStringToFloatArray(stream.getData());
          break;
        default:
          // Ignore.
      }
    }

    API.DrawPrimitive primitive = mesh.getDrawPrimitive();
    if (positions == null || (normals == null && isPolygon(primitive))) {
      return Futures.immediateFailedFuture(
          new InvalidArgumentException(NO_MESH_ERR, new Client.Stack(() -> "")));
    }

    int[] indices = mesh.getIndexBuffer().getIndicesList().stream().mapToInt(x -> x).toArray();
    Model model = new Model(primitive, mesh.getStats(), positions, normals, indices);
    return Futures.immediateFuture(model);
  }

  private static float[] byteStringToFloatArray(ByteString bytes) {
    byte[] data = bytes.toByteArray();
    FloatBuffer buffer = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN).asFloatBuffer();
    float[] out = new float[data.length / 4];
    buffer.get(out);
    return out;
  }

  public static class Source {
    public final CommandIndex command;
    public final VertexSemantics semantics;

    public Source(CommandIndex command, VertexSemantics semantics) {
      this.command = command;
      this.semantics = semantics;
    }

    public static DeviceDependentModel.Source<Source> withSemantics(
        DeviceDependentModel.Source<Source> src, VertexSemantics newSemantics) {
      Source me = (src == null) ? null : src.source;
      return new DeviceDependentModel.Source<Source>((src == null) ? null : src.device,
          new Source((me == null) ? null : me.command, newSemantics));
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      Source s = (Source)obj;
      return Objects.equal(command, s.command) && Objects.equal(semantics, s.semantics);
    }

    @Override
    public int hashCode() {
      return 31 * command.hashCode() + (semantics == null ? 0 : semantics.hashCode());
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    public final VertexSemantics semantics;
    public final Model original;
    public final Model faceted;

    public Data(Path.Device device, VertexSemantics semantics, Model original, Model faceted) {
      super(device);
      this.semantics = semantics;
      this.original = original;
      this.faceted = faceted;
    }

    public boolean hasFaceted() {
      return faceted != null;
    }

    public boolean hasOriginal() {
      return original != null;
    }
  }

  public static class VertexSemantics {
    public final Element[] elements;

    private VertexSemantics(Element[] elements) {
      this.elements = elements;
    }

    public VertexSemantics(API.Mesh mesh, VertexSemantics current) {
      Map<String, Vertex.Semantic.Type> assigned = Maps.newHashMap();
      if (current != null) {
        for (Element e : current.elements) {
          assigned.put(e.name, e.semantic);
        }
      }

      this.elements = new Element[mesh.getVertexBuffer().getStreamsCount()];
      for (int i = 0; i < elements.length; i++) {
        elements[i] = Element.get(mesh.getVertexBuffer().getStreams(i), assigned);
      }
      Arrays.sort(elements, (e1, e2) -> e1.name.compareTo(e2.name));
    }

    public VertexSemantics copy() {
      return new VertexSemantics(elements.clone());
    }

    // TODO: this breaks immutability. Call .copy() before calling assign().
    public void assign(int idx, Vertex.Semantic.Type semantic) {
      Element e = elements[idx];
      elements[idx] = new Element(e.name, e.type, semantic);
    }

    public Path.MeshOptions.Builder getOptions() {
      Path.MeshOptions.Builder r = Path.MeshOptions.newBuilder();
      Arrays.stream(elements)
          .forEach(e -> r.addVertexSemantics(Path.MeshOptions.SemanticHint.newBuilder()
              .setName(e.name)
              .setType(e.semantic)));
      return r;
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof VertexSemantics)) {
        return false;
      }
      return Arrays.equals(elements, ((VertexSemantics)obj).elements);
    }

    @Override
    public int hashCode() {
      return Arrays.hashCode(elements);
    }

    public static class Element {
      public final String name;
      public final String type;
      public final Vertex.Semantic.Type semantic;

      public Element(String name, String type, Vertex.Semantic.Type semantic) {
        this.name = name;
        this.type = type;
        this.semantic = semantic;
      }

      public static Element get(Vertex.Stream s, Map<String, Vertex.Semantic.Type> assigned) {
        Vertex.Semantic.Type type = assigned.get(s.getName());
        if (type == null) {
          type = s.getSemantic().getType();
        }
        return new Element(s.getName(), Streams.toString(s.getFormat()), type);
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Element)) {
          return false;
        }
        Element e = (Element)obj;
        return name.equals(e.name) && type.equals(e.type) && semantic == e.semantic;
      }

      @Override
      public int hashCode() {
        int h = name.hashCode();
        h = 31 * h + type.hashCode();
        h = 31 * h + semantic.hashCode();
        return h;
      }
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that geometry data is being loaded.
     */
    public default void onGeometryLoadingStart() { /* empty */ }

    /**
     * Event indicating that the geometry data has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onGeometryLoaded(Loadable.Message error) { /* empty */ }
  }
}
