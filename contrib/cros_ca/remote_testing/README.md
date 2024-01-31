# Remote end-to-end testing

## Description

This package contains code for running an automation test on a remote
machine (DUT) from a Linux server. The system comprises two main components: a
server agent and a client agent. These agents communicate over OpenSSH (Secure
Shell) and move files between them using SFTP (SSH File Transfer Protocol). The
server initiates file transfers to the client and instructs it to execute an
automation code. The client executes the provided code and generate results
that the server will get and save them back.

The Remote end-to-end testing also contain the option to run the test from
the sleep mode on the Windows OS for now, where the server agent will send
wake on lan signal to the client to make it wake up and complete execution.

- <b>Server agent</b>: This agent will be executed on the host machine. It is
  responsible for:
    - Accept the required arguments to connect with the DUT and specify the
      automation code to be executed on the DUT.
    - Send the required files and directories for running the automation code to
      the DUT using SFTP protocol.
    - Send a request over SSH to run the automation test on the DUT device.
    - Get the test results from the DUT device.<br><br>


- <b>Client agent</b>: This agent will be executed on the DUT device to
  perform the
  required actions on it. It is responsible for several tasks, such as:
    - Keep listening until a request received from the server agent over SSH
      protocol.
    - Install the required packages using the requirements file from the
      server agent.
    - Run the automation test.
    - Save the agent and test results.
    - Remove the files and directories originating from the host machine,
      as well as any generated files from the client agent.

<b style="color:yellow">Note</b>: You can use the scripts provided in ```/remote_testing/setup_scripts``` that reduce
the manual effort in
some steps, please refer to [README.md](setup_scripts%2FREADME.md) for more description.

## Prerequisites

### Prerequisites for Server (Host) Machine

The host machine intended to run the ```server_agent.py``` script should have
the
Linux (Ubuntu) operating system installed. Below are the steps that should be
completed before executing the server agent:

1. Python installed on the host machine.
2. The code of the automation tests ```cros_ca``` and its content should be
   existed in the host machine.
3. ```server_agent.py``` should be moved to the home
   directory:```/home/user_name/server_agent.py```.

### Prerequisites for Client (DUT) Machine

#### Client Machine With Windows OS:

<b style="color:yellow">Note</b>: the first two steps covered by ```wi_dut_setup``` scripts, if you want to know more
please refer to [README.md](setup_scripts%2FREADME.md) file that describe the scripts.

1. OpenSSH Server should be installed on the Windows machine:
    - Open the device <b>Settings -> System -> Optional features -> Add an
      optional feature</b>.
    - Search for <b>OpenSSH Server</b> and Install it.

2. OpenSSH Server should be start running to accept signals over SSH, this
   can be done by following the below steps:
    - Search for <b>Services</b> in the taskbar search box.
    - Find <b>OpenSSH SSH Server-> right click -> properties</b> change the
      Startup type to Automatic (To make it started automatically when open the
      device).
    - Apply and press <b>Start</b> then <b>OK</b>.
3. Now you need to follow the steps in ([README.md](..%2FREADME.md)) file
   under <b>
   Prerequisites for Windows </b> title.

4. Update the ```PYTHONPATH``` in the system environment variables by replacing
   its value
   with ```C:\Users\your_username\Automation Code```.

5. Copy the ```client_agent.py``` script to your user where the path of the
   agent will be like this: ```C:\Users\your_username\client_agent.py```.

6. For <b style="color:blue"> Sleep scenario </b> that is supported for
   windows you can follow
   this [[Sleep Mode Scenario](#sleep-mode)]

#### Client Machine With Chrome OS:

1. For the Chrome OS to enable OpenSSH testing image installation required,
   so please follow the steps in this ([README.md](..%2FREADME.md)) file
   under <b>
   Prerequisites for ChromeOS </b> title.
2. Update the ```PYTHONPATH``` in the bashrc file by replacing its value
   with ```/home/chronos/user/MyFiles/Automation Code``` where this directory
   will
   contain the code that will be moved by the server agent.
3. Copy the ```client_agent.py``` script to ```/home/chronos/user/```.

4. It will be better to restart the device after doing the above steps.

### Usage

1. You need to run the ```client_agent.py``` on the DUT first, this can be
   done as below:
    - For Windows: open the cmd in the client_agent.py directory, then:
         ```
             python client_agent.py
         ```
    - For Chrome: open the shell terminal and navigate to
      ```/home/chronos/user```, then:
         ```
             python3 client_agent.py
         ```
2. Run the ```server_agent.py``` on the server machine and pass the
   arguments:
    ```
   Arguments Descriptions:
   
    DUT_IP: The IP address of the DUT machine.
    DUT_MAC_ADDRESS: The mac address of the DUT machine
    DUT_username: The username of the DUT machine for authentication.
    DUT_password: The password of the DUT machine for authentication.
    status: An optional argument to specifiy from which status to start the 
   test (sleep or current) mode.
    test_path: Absolute path for the automation file
    ```

    - Run with the required arguments:
         ```
             python3 server_agent.py DUT_IP DUT_MAC_ADDRESS DUT_username 
             DUT_password "test_abs_path"
         ```
    - Run with all arguments:
         ```
             python3 server_agent.py DUT_IP DUT_MAC_ADDRESS DUT_username 
             DUT_password --status STATUS_VALUE "test_abs_path"
         ```

### <a id="sleep-mode"></a>Sleep Mode Scenario

This scenario is currently supported on Windows client machines. It involves
running a test after forcing the device to enter sleep mode. Subsequently, the
server agent sends a signal to awaken the device and execute the automation
test. To enable this scenario, adhere to the following prerequisites:

1. The DUT device must be connected to the network through ethernet cable,
   and you can use ethernet adapter.
2. Change the network adapter settings to enable it to receive magic puckets:
    - Open the <b> Control Panel -> Network and Internet -> Network and
      Sharing Center</b>
    - Select <b> Change adapter settings</b> option.
    - Right-click on the ethernet adapter <b> properties -> Configure</b>
    - In the <b>Advanced</b> window from the Property choices change the
      value of <b>Wake On Magic Packet and Wake on pattern match</b> to <b>
      Enabled </b>.
3. Open the BIOS settings and enable Wake On Lan.
4. Disable sign in for sleep mode,this can be done by following the below
   steps:
    - Search for <b>Sign-in options</b> in the taskbar search box.
    - In the <b> Additional Settings </b> or <b> Require sign-in </b> change
      the value of <b> If you've been away, when should Windows require you
      to sign in again?</b> to <b>Never</b>

#### Sleep Scenario Execution:

In the server side you need to add a new argument which is ```--status sleep```
before the test file path, the command will be as below:

   ```
     python3 server_agent.py DUT_IP DUT_MAC_ADDRESS DUT_username 
     DUT_password --status sleep "test_abs_path"
   ```

where the ```--status``` argument is an optional argument, and it's default
behavior is to run the automation test from the current mode.

In this scenario the ```--status sleep``` and the ```DUT_MAC_ADDRESS``` will
be used to make the device wake up from the sleep mode.