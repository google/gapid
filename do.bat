@echo off
SETLOCAL

REM -- the directory in which do.bat lives
SET DO_DIR=%~dp0

IF NOT EXIST %DO_DIR%..\..\..\..\src\github.com\google\gapid (
  echo It looks like the directory structure of
  echo   %DO_DIR%
  echo doesn't match what is expected.
  echo.
  echo Did you 'git clone' the project instead of using 'go get'?
  echo Please follow the building directions found here:
  echo   https://github.com/google/gapid/blob/master/BUILDING.md
  exit /B
)

REM -- glob all go files in cmd\do
set DO_GO_FILES=
for /f "tokens=*" %%F in ('dir /b /a:-d "cmd\do\*.go"') do call set DO_GO_FILES=%%DO_GO_FILES%% "cmd\do\%%F"

REM -- set GOPATH
pushd .
cd %DO_DIR%..\..\..\..\
SET GOPATH=%DO_DIR%third_party;%CD%
popd

REM -- Do some env cleanup.
SET LIB=
SET LIBPATH=

pushd .
cd %DO_DIR%
go run %DO_GO_FILES% %*
popd


