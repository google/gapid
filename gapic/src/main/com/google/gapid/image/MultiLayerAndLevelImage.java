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

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;

/**
 * An image with (optionally) multiple layers and mipmap levels.
 */
public interface MultiLayerAndLevelImage {
  /**
   * @return the number of layers in this image.
   */
  public int getLayerCount();

  /**
   * @return the number of levels in this image.
   */
  public int getLevelCount();

  /**
   * @return a future {@link Image} representing the given 0-based level.
   */
  public ListenableFuture<Image> getImage(int layer, int level);

  public static final MultiLayerAndLevelImage EMPTY = new MultiLayerAndLevelImage() {
    @Override
    public int getLayerCount() {
      return 1;
    }

    @Override
    public int getLevelCount() {
      return 1;
    }

    @Override
    public ListenableFuture<Image> getImage(int layer, int level) {
      return Futures.immediateFuture(Image.EMPTY);
    }
  };
}
