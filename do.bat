@echo off
SETLOCAL

REM -- glob all go files in cmd\do
set DO_GO_FILES=
for /f "tokens=*" %%F in ('dir /b /a:-d "cmd\do\*.go"') do call set DO_GO_FILES=%%DO_GO_FILES%% "cmd\do\%%F"

REM -- the directory in which do.bat lives
SET DO_DIR=%~dp0

REM -- set GOPATH
pushd .
cd %DO_DIR%..\..\..\..\
SET GOPATH=%DO_DIR%third_party;%CD%
popd

pushd .
cd %DO_DIR%
go run %DO_GO_FILES% %*
popd


