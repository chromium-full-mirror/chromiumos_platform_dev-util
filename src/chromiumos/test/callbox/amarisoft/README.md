# Setup
To setup the callbox, copy all files from this directory to anywhere in the callbox(e.g. /tmp/).

Run the setup script corresponding to the configuration you want for the callbox.

Example:
>cd /tmp; bash setup4G.sh


# Some basic commands to troubleshoot issues on the callbox:
Restart all services. Use this after changing configuration files.
>service lte restart

Open the callbox software monitors
>screen -x lte

# Callbox commands inside "screen"
Switch between MME/ENB monitors
>Ctrl+a 0, Ctrl+a 1

Exit screen
>Ctrl+a d

Set cell gain(ENB)
Example: Set cell 1's gain to -200dB
>cell_gain 1 -200

Check cell gain(ENB)
>cell

# Read service logs
>journalctl -u cros_server -f
>journalctl -u named --since "1 hour ago"

# callbox log location
>/tmp/mme.log\
>/tmp/enb0.log
