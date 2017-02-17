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
package com.google.gapid.rpclib.futures;

import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicReference;

/**
 * An implementation of {@link FutureController} that ensures that only a single {@link Future} is
 * in flight at any given time for a given system. If a {@link Future} is started before another
 * has finished, then the first {@link Future} is cancelled.
 *
 * <p>This class can be used to automatically cancel older asynchronous tasks when newer ones are
 * created. By using the constructor that takes a {@link Listener}, a loading progress element can
 * be driven, even when task lifetimes overlap.
 */
public class SingleInFlight implements FutureController {
    private final AtomicReference<Future<?>> mActive = new AtomicReference<Future<?>>();
    private final Listener mListener;
    private final Listener NULL_LISTENER = new Listener() {
        @Override
        public void onIdleToWorking() {}

      @Override
        public void onWorkingToIdle() {}
    };

    public interface Listener {
        /**
         * Called when transitioning from a state where no {@link Future}s are running to a
         * single {@link Future} is running.
         */
        void onIdleToWorking();

        /**
         * Called when transitioning from a state where a single {@link Future} is running
         * to no {@link Future}s are running.
         */
        void onWorkingToIdle();
    }

    public SingleInFlight() {
        this.mListener = NULL_LISTENER;
    }

    public SingleInFlight(Listener listener) {
        this.mListener = listener;
    }

    @Override
    public void onStart(Future<?> future) {
        Future<?> prev = mActive.getAndSet(future);
        if (prev != null) {
            prev.cancel(true);
        } else {
            mListener.onIdleToWorking();
        }
    }

    @Override
    public boolean onStop(Future<?> future) {
        if (mActive.compareAndSet(future, null)) {
            mListener.onWorkingToIdle();
            return true;
        }
        return false;
    }
}
