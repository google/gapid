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

#pragma once

// -----------------------------------------------------------------------------
// extern "C" interface designed for users who dlopen the interceptor-lib
// instead of linking against it. The API for these functions using C structures
// only to support users compiled with different STL library.
// -----------------------------------------------------------------------------

extern "C" {

void *InitializeInterceptor();

void TerminateInterceptor(void *interceptor);

void *FindFunctionByName(void *interceptor, const char *symbol_name);

bool InterceptFunction(void *interceptor, void *old_function,
                       void *new_function, void **callback_function,
                       void (*error_callback)(void *, const char *) = nullptr,
                       void *error_callback_baton = nullptr);

bool InterceptSymbol(void *interceptor, const char *symbol_name,
                     void *new_function, void **callback_function,
                     void (*error_callback)(void *, const char *) = nullptr,
                     void *error_callback_baton = nullptr);

} // extern "C"
