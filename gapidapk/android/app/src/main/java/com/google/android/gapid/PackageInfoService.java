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

package com.google.android.gapid;

import android.app.IntentService;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.pm.ActivityInfo;
import android.content.pm.ApplicationInfo;
import android.content.pm.PackageInfo;
import android.content.pm.PackageManager;
import android.content.pm.ResolveInfo;
import android.content.res.Resources;
import android.graphics.Bitmap;
import android.graphics.drawable.BitmapDrawable;
import android.graphics.drawable.Drawable;
import android.util.Base64;
import android.util.DisplayMetrics;
import android.util.Log;

import org.json.JSONArray;
import org.json.JSONException;
import org.json.JSONObject;

import java.io.ByteArrayOutputStream;
import java.lang.reflect.Field;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.IdentityHashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;

/**
 * An {@link IntentService} subclass for providing installed package information to GAPIS / GAPIC.
 * <p/>
 * When the service is sent the {@link #ACTION_SEND_PKG_INFO} action, the service will begin
 * listening on the supplied local-abstract socket provided in the {@link #EXTRA_SOCKET_NAME} extra,
 * or if the extra is absent, {@link #DEFAULT_SOCKET_NAME}. When an incoming connection to this
 * socket is made, the service will send the installed package information on the accepted
 * connection, then close the accepted connection and the listening socket.
 */
public class PackageInfoService extends IntentService {
    private static final String TAG = "gapid-pkginfo";

    private static final int BASE_ICON_DENSITY = DisplayMetrics.DENSITY_MEDIUM;

    /**
     * Action used to start waiting for an incoming connection on the local-abstract port
     * {@link #EXTRA_SOCKET_NAME}. When a connection is made, the package information is send to the
     * connected socket, the socket is closed and the service stops listening on
     * {@link #EXTRA_SOCKET_NAME}.
     */
    private static final String ACTION_SEND_PKG_INFO = "com.google.android.gapid.action.SEND_PKG_INFO";

    /**
     * Optional parameter for {@link #ACTION_SEND_PKG_INFO} that changes the local-abstract port
     * used to listen for incoming connections. The default value is {@link #DEFAULT_SOCKET_NAME}.
     */
    private static final String EXTRA_SOCKET_NAME = "com.google.android.gapid.extra.SOCKET_NAME";

    /**
     * Optional parameter for {@link #ACTION_SEND_PKG_INFO} that makes the service include icons.
     */
    private static final String EXTRA_INCLUDE_ICONS = "com.google.android.gapid.extra.INCLUDE_ICONS";

    /**
     * Optional parameter for {@link #ACTION_SEND_PKG_INFO} that scales the icon density.
     */
    private static final String EXTRA_ICON_DENSITY_SCALE = "com.google.android.gapid.extra.ICON_DENSITY_SCALE";

    /**
     * Optional parameter for {@link #ACTION_SEND_PKG_INFO} that makes the service only report
     * debuggable packages, for use on production Android builds.
     */
    private static final String EXTRA_ONLY_DEBUG = "com.google.android.gapid.extra.ONLY_DEBUG";

    /**
     * The default socket name when {@link #EXTRA_SOCKET_NAME} is not provided.
     */
    private static final String DEFAULT_SOCKET_NAME = "gapid-pkginfo";

    public PackageInfoService() {
        super("PackageInfoService");
    }

    @Override
    protected void onHandleIntent(Intent intent) {
        if (intent != null) {
            final String action = intent.getAction();
            if (ACTION_SEND_PKG_INFO.equals(action)) {
                String socketName = intent.getStringExtra(EXTRA_SOCKET_NAME);
                if (socketName == null) {
                    socketName = DEFAULT_SOCKET_NAME;
                }
                boolean onlyDebug = intent.getBooleanExtra(EXTRA_ONLY_DEBUG, false);
                boolean includeIcons = intent.getBooleanExtra(EXTRA_INCLUDE_ICONS, false);
                float iconDensityScale = intent.getFloatExtra(EXTRA_ICON_DENSITY_SCALE, 1.0f);
                handleSendPackageInfo(socketName, onlyDebug, includeIcons, iconDensityScale);
            }
        }
    }

    /**
     * Handler for the {@link #ACTION_SEND_PKG_INFO} intent.
     */
    private void handleSendPackageInfo(
            final String socketName,
            final boolean onlyDebug,
            final boolean includeIcons,
            final float iconDensityScale) {

        final ExecutorService executor = Executors.newCachedThreadPool();
        final IconStore icons = new IconStore((int)(BASE_ICON_DENSITY * iconDensityScale));
        final PackageManager pm = getPackageManager();

        final Future<List<PackageInfo>> installedPackagesFuture = executor.submit(
                new Callable<List<PackageInfo>>() {
                    @Override
                    public List<PackageInfo> call() throws Exception {
                        List<PackageInfo> packages = pm.getInstalledPackages(
                                PackageManager.GET_ACTIVITIES | PackageManager.GET_SIGNATURES);
                        return packages;
                    }
                });

        Callable<byte[]> packageInfoFuture = new Callable<byte[]>() {
            @Override
            public byte[] call() throws Exception {
                String json = getPackageInfo(includeIcons ? icons : null, onlyDebug, pm, installedPackagesFuture.get());
                return json.getBytes("UTF-8");
            }
        };

        try {
            SocketWriter.connectAndWrite(socketName, executor.submit(packageInfoFuture));
        } catch (Exception ex) {
            Log.e(TAG, "Error occurred", ex);
        } finally {
            executor.shutdown();
        }
    }

    private String getPackageInfo(
            IconStore icons,
            boolean onlyDebug,
            PackageManager pm,
            List<PackageInfo> packages) throws JSONException {

        // The ApplicationInfo.primaryCpuAbi field is hidden. Use reflection to get at it.
        Field primaryCpuAbiField = null;
        try {
            primaryCpuAbiField = ApplicationInfo.class.getField("primaryCpuAbi");
        } catch (NoSuchFieldException e) {
            Log.w(TAG, "Unable to find 'primaryCpuAbi' ApplicationInfo hidden field");
        }

        JSONArray packagesJson = new JSONArray();

        for (PackageInfo packageInfo : packages) {
            ApplicationInfo applicationInfo = packageInfo.applicationInfo;
            boolean isDebuggable =
                    applicationInfo != null &&
                            (applicationInfo.flags & ApplicationInfo.FLAG_DEBUGGABLE) > 0;
            if (!isDebuggable && onlyDebug) {
                continue;
            }

            JSONObject packageJson = getPackageJson(
                    pm, packageInfo, icons, primaryCpuAbiField, isDebuggable);

            packagesJson.put(packageJson);
        }

        JSONObject root = new JSONObject();
        root.put("packages", packagesJson);
        root.put("icons", icons != null ? icons.json() : new JSONArray());

        return root.toString();
    }

    private JSONObject getPackageJson(
            PackageManager pm,
            PackageInfo packageInfo,
            IconStore icons,
            Field primaryCpuAbiField,
            boolean isDebuggable) throws JSONException {

        ApplicationInfo applicationInfo = packageInfo.applicationInfo;
        Intent launchIntent = pm.getLaunchIntentForPackage(packageInfo.packageName);
        ActivityInfo launchActivityInfo = null;
        if (launchIntent != null) {
            launchActivityInfo = launchIntent.resolveActivityInfo(pm, 0);
        }

        Resources resources = null;
        try {
            resources = pm.getResourcesForApplication(applicationInfo);
        } catch (PackageManager.NameNotFoundException ex) {}

        Map<String, List<IntentFilter>> activityIntents = new HashMap<String, List<IntentFilter>>();

        Intent queryIntent = new Intent();
        queryIntent.setPackage(packageInfo.packageName);
        List<ResolveInfo> resolveInfos = pm.queryIntentActivities(queryIntent, PackageManager.GET_RESOLVED_FILTER);
        for (ResolveInfo resolveInfo : resolveInfos) {
            IntentFilter intent = resolveInfo.filter;
            if (intent == null) {
                continue;
            }
            List<IntentFilter> intents = activityIntents.get(resolveInfo.activityInfo.name);
            if (intents == null) {
                intents = new ArrayList<IntentFilter>();
                activityIntents.put(resolveInfo.activityInfo.name, intents);
            }
            intents.add(intent);
        }

        JSONArray activitiesJson = new JSONArray();
        if (packageInfo.activities != null) {
            for (ActivityInfo activityInfo : packageInfo.activities) {
                int iconIndex = -1;
                if (icons != null) {
                    iconIndex = icons.add(resources, activityInfo.icon);
                }

                boolean isLaunchActivity = (launchActivityInfo != null) ?
                        launchActivityInfo.name.equals(activityInfo.name) : false;
                JSONArray actionsJson = new JSONArray();
                List<IntentFilter> intents = activityIntents.get(activityInfo.name);
                if (intents != null) {
                    for (IntentFilter intent : intents) {
                        for (int i = 0; i < intent.countActions(); i++) {
                            String action = intent.getAction(i);
                            JSONObject actionJson = new JSONObject();
                            actionJson.put("name", action);
                            if (isLaunchActivity) {
                                actionJson.put("isLaunch", action.equals(launchIntent.getAction()));
                            }
                            actionsJson.put(actionJson);
                        }
                    }
                }

                JSONObject activityJson = new JSONObject();
                activityJson.put("name", activityInfo.name);
                activityJson.put("icon", iconIndex);
                activityJson.put("actions", actionsJson);
                activitiesJson.put(activityJson);
            }
        }

        int iconIndex = -1;
        String primaryCpuAbi = null;

        if (applicationInfo != null) {
            if (icons != null) {
                iconIndex = icons.add(resources, applicationInfo.icon);
            }
            if (primaryCpuAbiField != null) {
                try {
                    primaryCpuAbi = (String) primaryCpuAbiField.get(applicationInfo);
                } catch (Exception e) {
                    Log.w(TAG, "Exception thrown accessing 'primaryCpuAbi': " + e.getMessage());
                }
            }
        }

        JSONObject packageJson = new JSONObject();
        packageJson.put("name", packageInfo.packageName);
        packageJson.put("debuggable", isDebuggable);
        packageJson.put("icon", iconIndex);
        if (primaryCpuAbi != null) {
            packageJson.put("abi", primaryCpuAbi);
        }
        packageJson.put("activities", activitiesJson);
        return packageJson;
    }

    /**
     * IconStore stores all {@link Drawable}s as PNG, base-64 encoded images.
     * Duplicates are only stored once.
     */
    private class IconStore {
        private final Map<String, Integer> mMap = new HashMap();
        private final IdentityHashMap<Bitmap, Integer> mBitmapMap = new IdentityHashMap<Bitmap, Integer>();
        private final JSONArray mJson = new JSONArray();
        private final int iconDensity;

        IconStore(int iconDensity) {
            this.iconDensity = iconDensity;
        }

        /**
         * add adds the specified drawable to the store.
         *
         * @return The index of the image stored in the {@link JSONArray} returned by {@link #json}.
         */
        public int add(Resources resources, int iconId) {
            if (resources == null || iconId <= 0) {
                return -1;
            }

            Drawable drawable = null;
            try {
                drawable = resources.getDrawableForDensity(iconId, iconDensity);
            } catch (Resources.NotFoundException ex) {
                return -1;
            }
            if (drawable == null || !(drawable instanceof BitmapDrawable)) {
                return -1;
            }

            Bitmap bitmap = ((BitmapDrawable) drawable).getBitmap();
            if (mBitmapMap.containsKey(bitmap)) {
                return mBitmapMap.get(bitmap);
            }

            ByteArrayOutputStream stream = new ByteArrayOutputStream();
            bitmap.compress(Bitmap.CompressFormat.PNG, 100, stream);
            byte[] pngBytes = stream.toByteArray();
            String pngBase64 = Base64.encodeToString(pngBytes, Base64.NO_WRAP);
            if (!mMap.containsKey(pngBase64)) {
                int index = mJson.length();
                mMap.put(pngBase64, index);
                mBitmapMap.put(bitmap, index);
                mJson.put(pngBase64);
                return index;
            } else {
                return mMap.get(pngBase64);
            }
        }

        /**
         * @return The {@link JSONArray} object holding all the base-64, PNG encoded images.
         */
        public JSONArray json() {
            return mJson;
        }
    }
}
