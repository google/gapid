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

if "%JRE_HOME%." == "." (
  echo Expected the JRE_HOME env variable.
  exit /b
)

REM Copy the JRE.
xcopy /e "%JRE_HOME%" "%1\"

REM Remove unnecessary files.
del "%1\bin\jabswitch.exe"
del "%1\bin\jaccessinspector.exe"
del "%1\bin\jaccesswalker.exe"
del "%1\bin\jaotc.exe"
del "%1\bin\JavaAccessBridge?64.dll"
del "%1\bin\java?crw?demo.dll"
del "%1\bin\java-rmi.exe"
del "%1\bin\jfr.exe"
del "%1\bin\jjs.exe"
del "%1\bin\jrunscript.exe"
del "%1\bin\keytool.exe"
del "%1\bin\kinit.exe"
del "%1\bin\klist.exe"
del "%1\bin\ktab.exe"
del "%1\bin\orbd.exe"
del "%1\bin\pack200.exe"
del "%1\bin\policytool.exe"
del "%1\bin\rmid.exe"
del "%1\bin\rmiregistry.exe"
del "%1\bin\servertool.exe"
del "%1\bin\tnameserv.exe"
del "%1\bin\unpack200.exe"
del "%1\bin\WindowsAccessBridge?64.dll"

del /q "%1\lib\jfr\*"
rmdir "%1\lib\jfr"
del "%1\lib\jfr.jar"
