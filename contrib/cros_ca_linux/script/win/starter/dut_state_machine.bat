@echo off

set exepath="c:\cros_ca\execution"
set toolspath="c:\cros_ca\scripts\tools"

cd %exepath%
set currentState=DeviceIdle
echo "device is ready" > device_ready.txt


:mainLoop
set oldState=%currentState%
set currentState=DeviceIdle

if exist start_test.txt (
    echo found start_test.txt
    del test_ended.txt
    del device_ready.txt
    set oldState=%currentState%
    set currentState=TestStarted
    echo "%oldState% to %currentState%"
    goto :testStarted
)

timeout 30
rem check intenet connection
ping www.google.com -n 1 >nul
if %errorlevel% == 0 (
    time /t > current_time.txt
    type current_time.txt
) else (
    shutdown /r /t 1
)

rem Optional hook to call. For example: reset the device
call %toolspath%\hooks.bat

goto :mainloop

:testStarted
echo "Enter testStarted State"
start /wait /b run.bat
del start_test.txt
goto :testEnded

:testEnded
set oldState=%currentState%
set currentState=TestEnded
echo "%oldState% to %currentState%"
echo testEnded
echo "%oldState% to %currentState%" > test_ended.txt
goto :mainLoop

:resultReady

:resultCopied

:readyForNewTest

:eof
