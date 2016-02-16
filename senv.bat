@setlocal enableextensions enabledelayedexpansion
@echo off
set GOPATH=%~dp0
if "x%PATH:senvgo=%"=="x%PATH%" (
	set PATH=%PATH%;%GOPATH%bin
)
echo GOPATH=%GOPATH%
echo PATH=%PATH%
