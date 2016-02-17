@echo off
set GOPATH=%~dp0
if "x%PATH:senvgo=%"=="x%PATH%" (
	set PATH=%PATH%;%GOPATH%bin
)
echo GOPATH=%GOPATH%
echo PATH=%PATH%

if not exist %GOPATH%src\github.com\VonC\senvgo (
	mkdir %GOPATH%src\github.com\VonC
	mklink /J %GOPATH%src\github.com\VonC\senvgo %GOPATH%
)
if not exist %GOPATH%src\vendor (
	mklink /J %GOPATH%src\vendor %GOPATH%\vendor
)
