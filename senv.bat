@echo off
if "%PRGS2%"=="" (
	echo.PRGS2 not defined
	goto ko
)
pushd "%CD%"
cd %~dp0
rem http://stackoverflow.com/questions/15890856/difference-between-dp0-and
set cpl=""
if not exist senvgo.exe (
	set cpl=true
)
REM http://stackoverflow.com/questions/1687014/how-do-i-compare-timestamps-of-files-in-a-dos-batch-script
FOR /F %%i IN ('DIR /B /O:D senvgo.exe main.go') DO SET NEWEST=%%i
REM echo.NEWEST %NEWEST%
if "%NEWEST%"=="main.go" (
	set cpl=true
)
REM echo.cpl %cpl%
if "%cpl%"=="true" (
	echo building
	call go build
	if ERRORLEVEL 1 (
		echo.build failed
		popd
		goto ko
	)
	echo.built
)
echo.done check build
call %~dp0\senvgo.exe
rem call %PRGS2%\env.bat
type %PRGS2%\env.bat
popd

rem http://stackoverflow.com/questions/4632891/exiting-batch-with-exit-b-x-where-x-1-acts-as-if-command-completed-successfu
:ok
@%COMSPEC% /C exit 0 >nul
goto EOF
:ko
@%COMSPEC% /C exit 1 >nul
:EOF
