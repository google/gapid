@echo off

for /f "usebackq tokens=*" %%i in (`"%ProgramFiles(x86)%\Microsoft Visual Studio\Installer\vswhere.exe" -prerelease -latest -property installationPath`) do (
    if exist "%%i" (
        echo "Checking %%i\VC\Auxiliary\Build\vcvars64.bat"
        if exist "%%i\VC\Auxiliary\Build\vcvars64.bat" (
            set VS16INSTALLDIR=%%i
            set VSVARS=%%i\VC\Auxiliary\Build\vcvars64.bat
        ) else (
            set VS16INSTALLDIR=%%i
            set VSVARS=%%i\VC\vcvarsall.bat
        )
    )
)

call "%VSVARS%"
@echo on
echo %*
md %4

cmake -GNinja -S %~dp0 -B %4 -DLIB_NAME=%2 -DLIB_SRC=%1 -DCMAKE_BUILD_TYPE=%3
if %errorlevel% neq 0 pause && exit /b %errorlevel%
cmake --build %4
if %errorlevel% neq 0 pause && exit /b %errorlevel%
exit /b 0