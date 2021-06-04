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
package com.google.gapid.util;

import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

public class URLs {
  public static final String FILE_BUG_URL =
      "https://github.com/google/agi/issues/new?template=standard-bug-report-for-gapid.md";
  public static final String DEVICE_COMPATIBILITY_URL = "https://gpuinspector.dev/validation";
  public static final String EXPECTED_ANGLE_PREFIX = "https://agi-angle.storage.googleapis.com/";
  public static final String ANGLE_DOWNLOAD = EXPECTED_ANGLE_PREFIX + "index.html";

  private URLs() {
  }

  public static boolean downloadWithProgressUpdates(
      URL url, OutputStream out, DownloadProgressListener listener) throws IOException {
    HttpURLConnection con = (HttpURLConnection) url.openConnection();
    long size = con.getContentLength();
    listener.onProgress(0, size);

    try (BufferedInputStream in = new BufferedInputStream(con.getInputStream())) {
      byte[] buffer = new byte[4096];
      long done = 0;
      int now;
      while ((now = in.read(buffer, 0, buffer.length)) >= 0) {
        done += now;
        out.write(buffer, 0, now);
        if (!listener.onProgress(done, size)) {
          return false;
        }
      }
    }
    return true;
  }

  public static interface DownloadProgressListener {
    @SuppressWarnings("unused")
    public default boolean onProgress(long done, long total) {
      return true;
    }
  }
}
