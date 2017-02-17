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

/**
 * Interface implemented by types that modify or simply observe {@link Future}s
 * as they are started and finished.
 */
public interface FutureController {
    /**
     * Called just after the {@link Future} is started.
     *
     * The function is free to cancel the {@link Future}.
     */
    void onStart(Future<?> future);

    /**
     * Called just after the {@link Future} has finished.
     *
     * @return true if the {@link Future} was considered active by the controller.
     */
    boolean onStop(Future<?> future);

    /**
     * Helper implementation of the interface that does nothing.
     */
    FutureController NULL_CONTROLLER = new FutureController() {
        @Override
        public void onStart(Future<?> future) {}

        @Override
        public boolean onStop(Future<?> future) { return true; }
    };
}
