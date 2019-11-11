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

package com.google.android.gapid;

import android.app.Activity;
import android.content.res.AssetManager;
import android.os.Bundle;
import android.util.Log;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.Window;

public class VKCubeActivity extends Activity implements SurfaceHolder.Callback  {
  private static final String APP_NAME = "VKCubeActivity";

  // Used to load the 'native-cube' library on application startup.
  static {
    try {
      System.loadLibrary("native_cube");
    } catch (Exception e) {
      Log.e(APP_NAME, "Native code library failed to load.\n" + e);
    }
  }

  @Override
  protected void onCreate(Bundle savedInstanceState) {
    super.onCreate(savedInstanceState);
    requestWindowFeature(Window.FEATURE_NO_TITLE);
    SurfaceView surfaceView = new SurfaceView(this);
    surfaceView.getHolder().addCallback(this);
    setContentView(surfaceView);
  }

  @Override
  public void surfaceCreated(SurfaceHolder holder) {
    Log.v(APP_NAME, "Surface created.");
    Surface surface = holder.getSurface();
    nStartCube(surface, getAssets());
  }

  @Override
  public void surfaceDestroyed(SurfaceHolder holder) {
    Log.v(APP_NAME, "Surface destroyed.");
    nStopCube();
  }

  @Override
  public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {}

  public native void nStartCube(Surface holder, AssetManager assetManager);
  public native void nStopCube();
}
