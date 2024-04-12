@echo off

set exepath="c:\cros_ca\execution"
for /L %%i in (1, 1, 100000000000000000000000000000000000) DO (
    ping www.google.com -n 1 >nul
    if %errorlevel% == 0 (
        time /t >> %exepath%\device_up_time.txt
    ) else (
	    del device_up_time_old.txt
	    move %exepath%\device_up_time.txt %exepath%\device_up_time_old.txt
        shutdown /r /t 1
    )
    timeout 300
)
