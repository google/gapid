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

#ifndef GAPII_ANDROID_INSTALLER_H
#define GAPII_ANDROID_INSTALLER_H

namespace gapii {

class Installer {
public:
    Installer(const char* libInterceptorPath);
    ~Installer();

    // install_function installs a hook into func_import to call func_export.
    // The returned function allows func_export to call back to the original
    // function that was at func_import.
    void* install(void* func_import, const void* func_export);

private:
    void install_gles();
};

} // namespace gapii

#endif // GAPII_ANDROID_INSTALLER_H
