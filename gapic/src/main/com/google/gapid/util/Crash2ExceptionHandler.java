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
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.SEVERE;

import com.google.common.base.Throwables;
import com.google.gapid.models.Settings;

import com.squareup.okhttp.MultipartBuilder;
import com.squareup.okhttp.OkHttpClient;
import com.squareup.okhttp.Request;
import com.squareup.okhttp.Response;

import java.io.IOException;
import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.util.logging.Logger;

/**
 * Handles uncaught exceptions and sends stacktraces to Crash2 server.
 */
public class Crash2ExceptionHandler implements Thread.UncaughtExceptionHandler, ExceptionHandler {
  protected static final Logger LOG = Logger.getLogger(Crash2ExceptionHandler.class.getName());

  protected static final String CRASH_REPORT_URL_BASE = "https://clients2.google.com/cr/report?";
  protected static final String CRASH_REPORT_PRODUCT = "GAPID";
  protected static final String CRASH_REPORT_VERSION = "Client:" + GAPID_VERSION.toString();
  protected static final String CRASH_REPORT_URL = CRASH_REPORT_URL_BASE + getUrlParameters();

  private final Settings settings;
  private final Thread.UncaughtExceptionHandler previousHandler;

  private Crash2ExceptionHandler(Settings settings) {
    this.settings = settings;
    this.previousHandler = getChildHandler(Thread.getDefaultUncaughtExceptionHandler());
  }

  private static String getUrlParameters() {
    StringBuilder result = new StringBuilder()
        .append("product=").append(CRASH_REPORT_PRODUCT)
        .append("&version=");
    try {
      result.append(URLEncoder.encode(CRASH_REPORT_VERSION, "UTF-8"));
    } catch (UnsupportedEncodingException e) {
      LOG.log(SEVERE, "UTF-8 not found", e);
      result.append("UnknownVersion");
    }
    return result.toString();
  }

  private static Thread.UncaughtExceptionHandler getChildHandler(
      Thread.UncaughtExceptionHandler handler) {
    return (handler instanceof Crash2ExceptionHandler) ?
        getChildHandler(((Crash2ExceptionHandler)handler).previousHandler) : handler;
  }

  public static Crash2ExceptionHandler register(Settings settings) {
    Crash2ExceptionHandler handler = new Crash2ExceptionHandler(settings);
    Thread.setDefaultUncaughtExceptionHandler(handler);
    return handler;
  }

  @Override
  public void uncaughtException(Thread thread, Throwable thrown) {
    reportException(thrown);

    // Pass the exception back to the os to get logged.
    if (previousHandler != null) {
      previousHandler.uncaughtException(thread, thrown);
    } else {
      System.err.println("Uncaught exception in thread " + thread + ":");
      thrown.printStackTrace();
    }
  }

  @Override
  public void reportException(Throwable thrown) {
    if (!settings.preferences().getReportCrashes()) {
      return;
    }

    new Thread() {
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
            LOG.log(INFO,
                "Crash Report Uploaded Successfully; Crash Report ID: " + response.body().string());
          } else {
            LOG.log(SEVERE, "Crash Report Not Uploaded; Response Code: " + response.code());
          }
        } catch (IOException e) {
          LOG.log(SEVERE, "Unable to upload exception to crash2", e);
        }
      }
    }.start();
  }
}
