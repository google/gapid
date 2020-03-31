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
package com.google.android.gapid;

import android.app.IntentService;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.os.Build;

/**
 * {@link IntentService} that can be run in the foreground. Newer versions of Android no longer
 * allow services to be run in the background, unless certain restrictive conditions are met.
 * Instead, services should run in the foreground and show a notification that they are running.
 */
public abstract class GapidService extends IntentService {
  private static final String CHANNEL_ID = "AGI_notification_channel";
  private final Type type;

  public GapidService(String name, Type type) {
    super(name);
    this.type = type;
  }

  @Override
  public void onCreate() {
    super.onCreate();
    // Show a notification, so Android doesn't take us down.
    Notification.Builder notification = new Notification.Builder(this)
        .setOngoing(true)
        .setContentTitle("Android GPU Inspector")
        .setContentText("AGI is examining your device...")
        // the package name for resources "R" is derived from the "custom_package" field in
        // gapidapk/android/app/src/main/BUILD.bazel
        .setSmallIcon(com.google.android.gapid.R.drawable.logo)
        // TODO: Show something if the user taps the notification?
        .setContentIntent(PendingIntent.getActivity(this, 0, new Intent(), 0));

    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      NotificationChannel channel = new NotificationChannel(
          CHANNEL_ID, "Service Noticiations", NotificationManager.IMPORTANCE_LOW);
      channel.setDescription("Notifications from the AGI background services");
      getSystemService(NotificationManager.class).createNotificationChannel(channel);
      notification.setChannelId(CHANNEL_ID);
    } else {
      notification.setPriority(Notification.PRIORITY_LOW);
    }

    startForeground(type.notificationId, notification.build());
  }

  protected static enum Type {
    DeviceInfo(0x6A91D000),
    PackageInfo(0x6A91D001);

    public final int notificationId;

    private Type(int notificationId) {
      this.notificationId = notificationId;
    }
  }
}
