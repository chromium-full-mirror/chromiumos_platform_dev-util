REM Call the power shell to start the other bat files. Power shell will
REM     exit after the CMD bat programs are running.
PowerShell Start-Process -WindowStyle Minimized C:\cros_ca\scripts\starter\dut_state_machine.bat
PowerShell Start-Process -WindowStyle Minimized C:\cros_ca\scripts\starter\check_internet_connection.bat
