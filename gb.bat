@echo off
rem @setlocal enableextensions enabledelayedexpansion
call %~dp0senv.bat
go test github.com/VonC/senvgo/inst
endlocal
