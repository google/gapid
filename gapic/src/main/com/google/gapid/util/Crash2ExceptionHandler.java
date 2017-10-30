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
package com.google.gapid.util;

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;

import com.google.common.base.Throwables;

import com.squareup.okhttp.OkHttpClient;
import com.squareup.okhttp.MultipartBuilder;
import com.squareup.okhttp.Request;
import com.squareup.okhttp.RequestBody;
import com.squareup.okhttp.Response;

import okio.Buffer;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.PrintStream;
import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.logging.Logger;
import java.util.logging.Level;

/**
 * Handles uncaught exceptions and sends stacktraces to Crash2 server.
 */
public class Crash2ExceptionHandler implements Thread.UncaughtExceptionHandler {
  private static final Logger LOG = Logger.getLogger(Crash2ExceptionHandler.class.getName());

  // TODO(baldwinn860): Send to production url when we get approval.
  private static final String CRASH_REPORT_URL_BASE = "https://clients2.google.com/cr/staging_report?";
  private static final String CRASH_REPORT_PRODUCT = "GAPID";
  private static final String CRASH_REPORT_VERSION = "Client:" + GAPID_VERSION.toString();
  private static final String CRASH_REPORT_VERSION_ENCODED;
  private static final String CRASH_REPORT_URL_FIELDS;
  private static final String CRASH_REPORT_URL;
  static { 
    String temp;
    try {
      temp = URLEncoder.encode(CRASH_REPORT_VERSION, "UTF-8");
    } catch(UnsupportedEncodingException e) {
      temp = "UnknownVersion";
    }
    CRASH_REPORT_VERSION_ENCODED = temp;
    CRASH_REPORT_URL_FIELDS = "product=" + CRASH_REPORT_PRODUCT + "&version=" + CRASH_REPORT_VERSION_ENCODED;
    CRASH_REPORT_URL = CRASH_REPORT_URL_BASE + CRASH_REPORT_URL_FIELDS;
  }

  private final Thread.UncaughtExceptionHandler previousHandler;

  private Crash2ExceptionHandler(Thread.UncaughtExceptionHandler oldHandler) {
    previousHandler = oldHandler;
  }

  public static void registerAsDefault() {
    if (!(Thread.getDefaultUncaughtExceptionHandler() instanceof Crash2ExceptionHandler)) {
      Crash2ExceptionHandler handler = new Crash2ExceptionHandler(Thread.getDefaultUncaughtExceptionHandler());
      Thread.setDefaultUncaughtExceptionHandler(handler);
    }
  }

  @Override
  public void uncaughtException(Thread thread, Throwable thrown) {
    reportException(thrown);
    // Pass the exception back to the os to get logged
    previousHandler.uncaughtException(thread, thrown);
  }

  public static void reportException(Throwable thrown) {
    Thread uploadThread = new Thread(new StackTraceUploader(thrown));
    uploadThread.start();
  }

  private static class StackTraceUploader implements Runnable {
    private final Throwable thrown;

    public StackTraceUploader(Throwable thrownIn) {
      thrown = thrownIn;
    }

    @Override
    public void run() {
      try {
        // Creates a connection to crash2
        OkHttpClient client = new OkHttpClient();
        Response response = client.newCall(new Request.Builder()
                .url(CRASH_REPORT_URL)
                .post(new MultipartBuilder()
                    .type(MultipartBuilder.FORM)
                    .addFormDataPart("product", CRASH_REPORT_PRODUCT)
                    .addFormDataPart("version", CRASH_REPORT_VERSION)
                    .addFormDataPart("exception_info", Throwables.getStackTraceAsString(thrown))
                    .build())
                .build())
            .execute();
        if (response.isSuccessful()) {
          LOG.log(Level.INFO, "Crash Report Uploaded Successfully; Crash Report ID: " + response.body().string());
        } else {
          LOG.log(Level.SEVERE, "Crash Report Not Uploaded; Response Code: " + response.code());
        }
      } catch (IOException e) {
        LOG.log(Level.SEVERE, "Unable to upload exception to crash2", e);
      }
    }
  }
}
