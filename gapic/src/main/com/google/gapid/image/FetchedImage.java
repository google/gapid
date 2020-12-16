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
package com.google.gapid.image;

import static com.google.common.util.concurrent.Futures.immediateFailedFuture;
import static com.google.common.util.concurrent.Futures.immediateFuture;
import static com.google.gapid.util.Paths.blob;
import static com.google.gapid.util.Paths.imageData;
import static com.google.gapid.util.Paths.imageInfo;
import static com.google.gapid.util.Paths.resourceInfo;
import static com.google.gapid.util.Paths.thumbnail;

import com.google.common.base.Function;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.Image.Key;
import com.google.gapid.proto.image.Image.Info;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Values;

import org.eclipse.swt.graphics.ImageData;

import java.util.List;
import java.util.function.Consumer;

/**
 * A {@link MultiLayerAndLevelImage} fetched from the RPC server.
 */
public class FetchedImage implements MultiLayerAndLevelImage {
  private final Layer[] layers;

  public static ListenableFuture<FetchedImage> load(
      Client client, Path.Device device, ListenableFuture<Path.ImageInfo> imageInfo) {
    return MoreFutures.transformAsync(imageInfo, imageInfoPath -> load(client, device, imageInfoPath));
  }

  public static ListenableFuture<FetchedImage> load(
      Client client, Path.Device device, Path.ImageInfo imagePath) {
    return MoreFutures.transformAsync(client.get(imageInfo(imagePath), device), value -> {
      Images.Format format = getFormat(value.getImageInfo());
      return MoreFutures.transform(client.get(imageData(imagePath, format.format), device),
          pixelValue -> new FetchedImage(client, device, format, pixelValue.getImageInfo()));
    });
  }

  public static ListenableFuture<FetchedImage> load(
    Client client, Path.Device device, Path.ImageInfo imagePath, Consumer<Info> onInfo) {
  return MoreFutures.transformAsync(client.get(imageInfo(imagePath), device), value -> {
    Images.Format format = getFormat(value.getImageInfo());
    onInfo.accept(value.getImageInfo());
    return MoreFutures.transform(client.get(imageData(imagePath, format.format), device),
        pixelValue -> new FetchedImage(client, device, format, pixelValue.getImageInfo()));
  });
}

  public static ListenableFuture<FetchedImage> load(
      Client client, Path.Device device, Path.ResourceData imagePath) {
    return MoreFutures.transformAsync(client.get(resourceInfo(imagePath), device), value -> {
      API.ResourceData data = value.getResourceData();
      API.Texture texture = data.getTexture();
      switch (texture.getTypeCase()) {
        case TEXTURE_1D:
          return load(client, device, imagePath, getFormat(texture.getTexture1D()));
        case TEXTURE_1D_ARRAY:
          return load(client, device, imagePath, getFormat(texture.getTexture1DArray()));
        case TEXTURE_2D:
          return load(client, device, imagePath, getFormat(texture.getTexture2D()));
        case TEXTURE_2D_ARRAY:
          return load(client, device, imagePath, getFormat(texture.getTexture2DArray()));
        case TEXTURE_3D:
          return load(client, device, imagePath, getFormat(texture.getTexture3D()));
        case CUBEMAP:
          return load(client, device, imagePath, getFormat(texture.getCubemap()));
        case CUBEMAP_ARRAY:
          return load(client, device, imagePath, getFormat(texture.getCubemapArray()));
        default:
          throw new UnsupportedOperationException("Unexpected resource type: " + value);
      }
    });
  }

  public static ListenableFuture<FetchedImage> load(
      Client client, Path.Device device, Path.ResourceData imagePath, Images.Format format) {
    return MoreFutures.transform(client.get(imageData(imagePath, format.format), device), value -> {
      API.ResourceData data = value.getResourceData();
      API.Texture texture = data.getTexture();
      switch (texture.getTypeCase()) {
        case TEXTURE_1D:
          return new FetchedImage(client, device, format, texture.getTexture1D());
        case TEXTURE_1D_ARRAY:
          return new FetchedImage(client, device, format, texture.getTexture1DArray());
        case TEXTURE_2D:
          return new FetchedImage(client, device, format, texture.getTexture2D());
        case TEXTURE_2D_ARRAY:
          return new FetchedImage(client, device, format, texture.getTexture2DArray());
        case TEXTURE_3D:
          return new FetchedImage(client, device, format, texture.getTexture3D());
        case CUBEMAP:
          return new FetchedImage(client, device, format, texture.getCubemap());
        case CUBEMAP_ARRAY:
          throw new UnsupportedOperationException("Cubemap Array images not yet implemented");
        default:
          throw new UnsupportedOperationException("Unexpected resource type: " + value);
      }
    });
  }

  public static ListenableFuture<ImageData> loadImage(
      ListenableFuture<FetchedImage> futureImage, final int layer, final int level) {
    return MoreFutures.transformAsync(futureImage, image -> MoreFutures.transform(
        image.getImage(
            Math.min(layer, image.getLayerCount() - 1),
            Math.min(level, image.getLevelCount() - 1)), (l) -> l.getImageData()));
  }

  public static ListenableFuture<ImageData> loadThumbnail(
      Client client, Path.Device device, Path.Thumbnail path, Consumer<Info> onInfo) {
    return loadImage(MoreFutures.transform(client.get(thumbnail(path), device), value -> {
      onInfo.accept(value.getImageInfo());
      return new FetchedImage(client, device, Images.Format.Color8, value.getImageInfo());
    }), 0, 0);
  }

  public static ListenableFuture<ImageData> loadThumbnail(
      Client client, Path.Device device, com.google.gapid.proto.image.Image.Info info) {
    return loadImage(
        immediateFuture(new FetchedImage(client, device, Images.Format.Color8, info)), 0, 0);
  }

  private static Images.Format getFormat(Info imageInfo) {
    return Images.Format.from(imageInfo.getFormat());
  }

  private static Images.Format getFormat(API.Texture1D texture) {
    return (texture.getLevelsCount() == 0) ? Images.Format.Color8 : getFormat(texture.getLevels(0));
  }

  private static Images.Format getFormat(API.Texture1DArray texture) {
    return (texture.getLayersCount() == 0 || texture.getLayers(0).getLevelsCount() == 0)
        ? Images.Format.Color8 : getFormat(texture.getLayers(0).getLevels(0));
  }

  private static Images.Format getFormat(API.Texture2D texture) {
    return (texture.getLevelsCount() == 0) ? Images.Format.Color8 : getFormat(texture.getLevels(0));
  }

  private static Images.Format getFormat(API.Texture2DArray texture) {
    return (texture.getLayersCount() == 0 || texture.getLayers(0).getLevelsCount() == 0)
        ? Images.Format.Color8 : getFormat(texture.getLayers(0).getLevels(0));
  }

  private static Images.Format getFormat(API.Texture3D texture) {
    return (texture.getLevelsCount() == 0) ? Images.Format.Color8 : getFormat(texture.getLevels(0));
  }

  private static Images.Format getFormat(API.Cubemap cubemap) {
    return (cubemap.getLevelsCount() == 0) ?
        Images.Format.Color8 : getFormat(cubemap.getLevels(0).getNegativeZ());
  }

  private static Images.Format getFormat(API.CubemapArray texture) {
    return (texture.getLayersCount() == 0 || texture.getLayers(0).getLevelsCount() == 0)
        ? Images.Format.Color8 : getFormat(texture.getLayers(0).getLevels(0).getNegativeZ());
  }


  public FetchedImage(Client client, Path.Device device, Images.Format format, Info imageInfo) {
    layers = new Layer[] {
        new Layer(new SingleFacedLevel(client, device, format, imageInfo))
    };
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Texture1D texture) {
    List<Info> infos = texture.getLevelsList();
    Level[] levels = new Level[infos.size()];
    for (int i = 0; i < infos.size(); i++) {
      levels[i] = new SingleFacedLevel(client, device, format, infos.get(i));
    }
    layers = new Layer[] { new Layer(levels) };
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Texture1DArray texture) {
    layers = new Layer[texture.getLayersCount()];
    for (int i = 0; i < layers.length; i++) {
      List<Info> infos = texture.getLayers(i).getLevelsList();
      Level[] levels = new Level[infos.size()];
      for (int j = 0; j < infos.size(); j++) {
        levels[j] = new SingleFacedLevel(client, device, format, infos.get(j));
      }
      layers[i] = new Layer(levels);
    }
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Texture2D texture) {
    List<Info> infos = texture.getLevelsList();
    Level[] levels = new Level[infos.size()];
    for (int i = 0; i < infos.size(); i++) {
      levels[i] = new SingleFacedLevel(client, device, format, infos.get(i));
    }
    layers = new Layer[] { new Layer(levels) };
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Texture2DArray texture) {
    layers = new Layer[texture.getLayersCount()];
    for (int i = 0; i < layers.length; i++) {
      List<Info> infos = texture.getLayers(i).getLevelsList();
      Level[] levels = new Level[infos.size()];
      for (int j = 0; j < infos.size(); j++) {
        levels[j] = new SingleFacedLevel(client, device, format, infos.get(j));
      }
      layers[i] = new Layer(levels);
    }
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Texture3D texture) {
    List<Info> infos = texture.getLevelsList();
    Level[] levels = new Level[infos.size()];
    for (int i = 0; i < infos.size(); i++) {
      levels[i] = new SingleFacedLevel(client, device, format, infos.get(i));
    }
    layers = new Layer[] { new Layer(levels) };
  }

  public FetchedImage(
      Client client, Path.Device device, Images.Format format, API.Cubemap cubemap) {
    List<API.CubemapLevel> infos = cubemap.getLevelsList();
    Level[] levels = new Level[infos.size()];
    for (int i = 0; i < infos.size(); i++) {
      levels[i] = new SixFacedLevel(client, device, format, infos.get(i));
    }
    layers = new Layer[] { new Layer(levels) };
  }

  @Override
  public int getLayerCount() {
    return layers.length;
  }

  @Override
  public int getLevelCount() {
    int count = 0;
    for (Layer layer : layers) {
      count = Math.max(count, layer.levels.length);
    }
    return count;
  }

  @Override
  public ListenableFuture<Image> getImage(int layerIdx, int levelIdx) {
    return (layerIdx < 0 || layerIdx >= layers.length) ?
        immediateFailedFuture(new IllegalArgumentException("Invalid image layer " + layerIdx)) :
        layers[layerIdx].getImage(levelIdx);
  }

  @Override
  public Image.Key getLevelKey(int level) {
    Key.Builder builder = new Key.Builder();
    for (int i = 0; i < layers.length; i++) {
      layers[i].appendLevelTo(level, builder);
    }
    return builder.build();
  }

  /**
   * A single {@link Image} layer.
   */
  private static class Layer {
    public final Level[] levels;

    public Layer(Level... levels) {
      this.levels = levels;
    }

    public ListenableFuture<Image> getImage(int level) {
      return level < 0 || level >= levels.length ?
          immediateFailedFuture(new IllegalArgumentException("Invalid image level " + level)) :
          levels[level].get();
    }

    public void appendLevelTo(int level, Image.Key.Builder keyBuilder) {
      levels[level].appendTo(keyBuilder);
    }
  }

  /**
   * A single mipmap level {@link Image} of a {@link FetchedImage}.
   */
  private abstract static class Level implements Function<Image, Image> {
    public static final Level EMPTY_LEVEL = new Level(null) {
      @Override
      public ListenableFuture<Image> get() {
        return immediateFuture(Image.EMPTY);
      }

      @Override
      protected ListenableFuture<Image> doLoad() {
        return null;
      }

      @Override
      public void appendTo(Image.Key.Builder keyBuilder) {
        // Do nothing.
      }
    };

    protected final Images.Format format;
    private Image image;

    public Level(Images.Format format) {
      this.format = format;
    }

    public ListenableFuture<Image> get() {
      Image result;
      synchronized (this) {
        result = image;
      }
      return (result == null) ? MoreFutures.transform(doLoad(), this) : immediateFuture(result);
    }

    @Override
    public Image apply(Image input) {
      synchronized (this) {
        image = input;
      }
      return image;
    }

    protected abstract ListenableFuture<Image> doLoad();

    public abstract void appendTo(Image.Key.Builder keyBuilder);

    protected static Image convertImage(Info info, Images.Format format, byte[] data) {
      return format.builder(Image.Key.of(info), info.getWidth(), info.getHeight(), info.getDepth())
          .update(data, 0, 0, 0, info.getWidth(), info.getHeight(), info.getDepth())
          .build();
    }

    protected static Image convertImage(Info[] infos, Images.Format format, byte[][] data) {
      assert (infos.length == data.length && infos.length == 6);
      // Typically these are all the same, but let's be safe.
      int width = Math.max(
          Math.max(Math.max(Math.max(Math.max(infos[0].getWidth(), infos[1].getWidth()),
              infos[2].getWidth()), infos[3].getWidth()), infos[4].getWidth()),
          infos[5].getWidth());
      int height =
          Math.max(Math.max(
              Math.max(Math.max(Math.max(infos[0].getHeight(), infos[1].getHeight()),
                  infos[2].getHeight()), infos[3].getHeight()),
              infos[4].getHeight()), infos[5].getHeight());

      // +----+----+----+----+
      // |    | -Y |    |    |
      // +----+----+----+----+
      // | -X | +Z | +X | -Z |
      // +----+----+----+----+
      // |    | +Y |    |    |
      // +----+----+----+----+
      return format.builder(Image.Key.of(infos), 4 * width, 3 * height, 1)
          .update(data[0], 0 * width, 1 * height, 0, infos[0].getWidth(), infos[0].getHeight(), 1) // -X
          .update(data[1], 2 * width, 1 * height, 0, infos[1].getWidth(), infos[1].getHeight(), 1) // +X
          .update(data[2], 1 * width, 2 * height, 0, infos[2].getWidth(), infos[2].getHeight(), 1) // -Y
          .update(data[3], 1 * width, 0 * height, 0, infos[3].getWidth(), infos[3].getHeight(), 1) // +Y
          .update(data[4], 3 * width, 1 * height, 0, infos[4].getWidth(), infos[4].getHeight(), 1) // -Z
          .update(data[5], 1 * width, 1 * height, 0, infos[5].getWidth(), infos[5].getHeight(), 1) // +Z
          .flip()
          .build();
    }
  }

  /**
   * A {@link Level} of a simple 2D texture.
   */
  private static class SingleFacedLevel extends Level {
    private final Client client;
    private final Path.Device device;
    protected final Info imageInfo;

    public SingleFacedLevel(
        Client client, Path.Device device, Images.Format format, Info imageInfo) {
      super(format);
      this.client = client;
      this.device = device;
      this.imageInfo = imageInfo;
    }

    @Override
    protected ListenableFuture<Image> doLoad() {
      return MoreFutures.transform(client.get(blob(imageInfo.getBytes()), device), data ->
        convertImage(imageInfo, format, Values.getBytes(data)));
    }

    @Override
    public void appendTo(Image.Key.Builder keyBuilder) {
      keyBuilder.add(imageInfo);
    }
  }

  /**
   * A {@link Level} of a cubemap texture.
   */
  private static class SixFacedLevel extends Level {
    private final Client client;
    private final Path.Device device;
    protected final Info[] imageInfos;

    public SixFacedLevel(
        Client client, Path.Device device, Images.Format format, API.CubemapLevel level) {
      super(format);
      this.device = device;
      this.client = client;
      this.imageInfos = new Info[] {
        level.getNegativeX(), level.getPositiveX(),
        level.getNegativeY(), level.getPositiveY(),
        level.getNegativeZ(), level.getPositiveZ()
      };
    }

    @Override
    protected ListenableFuture<Image> doLoad() {
      @SuppressWarnings("unchecked")
      ListenableFuture<Service.Value>[] futures = new ListenableFuture[imageInfos.length];
      for (int i = 0; i < imageInfos.length; i++) {
        futures[i] = client.get(blob(imageInfos[i].getBytes()), device);
      }
      return MoreFutures.transform(Futures.allAsList(futures), values -> {
        byte[][] data = new byte[values.size()][];
        for (int i = 0; i < data.length; i++) {
          data[i] = Values.getBytes(values.get(i));
        }
        return convertImage(imageInfos, format, data);
      });
    }

    @Override
    public void appendTo(Image.Key.Builder keyBuilder) {
      keyBuilder.add(imageInfos);
    }
  }
}
