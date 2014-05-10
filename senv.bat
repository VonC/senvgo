@echo off
if "%PRGS2%"=="" (
	echo.PRGS2 not defined
	goto ko
)
pushd "%CD%"
cd %~dp0
rem http://stackoverflow.com/questions/15890856/difference-between-dp0-and
C:\prgs\go\go1.2.1.windows-amd64\bin\go.exe build
if ERRORLEVEL 1 (
	echo.build failed
	popd
	goto ko
)
echo.built
call %~dp0\senvgo.exe
call %PRGS2%\env.bat
popd


rem http://stackoverflow.com/questions/4632891/exiting-batch-with-exit-b-x-where-x-1-acts-as-if-command-completed-successfu
:ok
@%COMSPEC% /C exit 0 >nul
goto EOF
:ko
@%COMSPEC% /C exit 1 >nul
:EOF
