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

public interface MultiLevelImage {
  int getLevelCount();

  ListenableFuture<Image> getLevel(int index);

  MultiLevelImage EMPTY = new MultiLevelImage() {
    @Override
    public int getLevelCount() {
      return 1;
    }

    @Override
    public ListenableFuture<Image> getLevel(int index) {
      return Futures.immediateFuture(Image.EMPTY);
    }
  };
}
