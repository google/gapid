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
xcopy /e "%JAVA_HOME%\jre" "%1\"

REM Remove unnecessary files.
REM See http://www.oracle.com/technetwork/java/javase/jre-8-readme-2095710.html
del /q "%1\bin\dtplugin\*"
rmdir "%1\bin\dtplugin"
del /q "%1\bin\plugin2\*"
rmdir "%1\bin\plugin2"
del "%1\bin\decora?sse.dll"
del "%1\bin\deploy.dll"
del "%1\bin\eula.dll"
del "%1\bin\fxplugins.dll"
del "%1\bin\glass.dll"
del "%1\bin\glib?lite.dll"
del "%1\bin\gstreamer?lite.dll"
del "%1\bin\jabswitch.exe"
del "%1\bin\JavaAccessBridge?32.dll"
del "%1\bin\JavaAccessBridge?64.dll"
del "%1\bin\javacpl.cpl"
del "%1\bin\javacpl.exe"
del "%1\bin\java?crw?demo.dll"
del "%1\bin\javafx?font.dll"
del "%1\bin\javafx?font?t2k.dll"
del "%1\bin\javafx?iio.dll"
del "%1\bin\java-rmi.exe"
del "%1\bin\javaws.exe"
del "%1\bin\JAWTAccessBridge?32.dll"
del "%1\bin\JAWTAccessBridge?64.dll"
del "%1\bin\jfr.dll"
del "%1\bin\jfxmedia.dll"
del "%1\bin\jfxwebkit.dll"
del "%1\bin\jjs.exe"
del "%1\bin\jp2*"
del "%1\bin\jucheck.exe"
del "%1\bin\keytool.exe"
del "%1\bin\kinit.exe"
del "%1\bin\klist.exe"
del "%1\bin\ktab.exe"
del "%1\bin\orbd.exe"
del "%1\bin\pack200.exe"
del "%1\bin\policytool.exe"
del "%1\bin\prism?common.dll"
del "%1\bin\prism?d3d.dll"
del "%1\bin\prism?es2.dll"
del "%1\bin\prism?sw.dll"
del "%1\bin\rmid.exe"
del "%1\bin\rmiregistry.exe"
del "%1\bin\servertool.exe"
del "%1\bin\ssv*"
del "%1\bin\tnameserv.exe"
del "%1\bin\unpack200.exe"
del "%1\bin\WindowsAccessBridge?32.dll"
del "%1\bin\WindowsAccessBridge?64.dll"
del "%1\bin\wsdetect.dll"

del /q "%1\lib\deploy\*"
rmdir "%1\lib\deploy"
del "%1\lib\ant?javafx.jar"
del "%1\lib\deploy.jar"
del "%1\lib\javafx.properties"
del "%1\lib\javaws.jar"
del "%1\lib\jfr.exe"
del "%1\lib\jfr.jar"
del "%1\lib\jfxswt.jar"
del "%1\lib\plugin.jar"
