goto :start
Copyright (C) 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Creates a distributable copy of the JRE.

:start

if "%1." == "." (
  echo Expected the destination folder as an argument.
  exit /b
)

REM Copy the JRE.
xcopy /e "%JAVA_HOME%"\* "%1\"

REM Remove unnecessary files.
REM See kokoro/macos/copy_jre.sh for details.
rmdir /q /s "%1\demo"
rmdir /q /s "%1\include"
rmdir /q /s "%1\jmods"
del /f /q "%1\bin\attach.dll"
del /f /q "%1\bin\dt_shmem.dll"
del /f /q "%1\bin\jar.exe"
del /f /q "%1\bin\jarsigner.exe"
del /f /q "%1\bin\javac.exe"
del /f /q "%1\bin\javadoc.exe"
del /f /q "%1\bin\javap.exe"
del /f /q "%1\bin\jcmd.exe"
del /f /q "%1\bin\jconsole.exe"
del /f /q "%1\bin\jdb.exe"
del /f /q "%1\bin\jdeprscan.exe"
del /f /q "%1\bin\jdeps.exe"
del /f /q "%1\bin\jhsdb.exe"
del /f /q "%1\bin\jimage.exe"
del /f /q "%1\bin\jinfo.exe"
del /f /q "%1\bin\jlink.exe"
del /f /q "%1\bin\jmap.exe"
del /f /q "%1\bin\jmod.exe"
del /f /q "%1\bin\jps.exe"
del /f /q "%1\bin\jshell.exe"
del /f /q "%1\bin\jstack.exe"
del /f /q "%1\bin\jstatd.exe"
del /f /q "%1\bin\jstat.exe"
del /f /q "%1\bin\rmic.exe"
del /f /q "%1\bin\saproc.dll"
del /f /q "%1\bin\serialver.exe"
del /f /q "%1\lib\ct.sym"
del /f /q "%1\lib\src.zip"
