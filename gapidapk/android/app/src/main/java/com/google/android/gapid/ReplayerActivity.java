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

import android.app.NativeActivity;
import android.os.Bundle;
import android.view.Window;

// This class exists to disambiguate activity names between native activities inside the GAPID
// APK. It just needs to extend NativeActivity.
public class ReplayerActivity extends NativeActivity {
    public void onCreate(Bundle savedInstanceState) {
        getWindow().requestFeature(Window.FEATURE_NO_TITLE);
        
        super.onCreate(savedInstanceState);
        
        getWindow().takeSurface( /* callback= */null);
        getWindow().setContentView(R.layout.replayer_main);
    }
}
