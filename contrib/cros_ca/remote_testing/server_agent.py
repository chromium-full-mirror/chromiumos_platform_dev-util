# pylint: disable= W1203, W0718, R1710
"""Server Agent For Remote Connection with a DUT."""

import argparse
import getpass
import logging
import os
import platform
import shutil
import socket
import subprocess
import sys
import tarfile
import time
import zipfile
import urllib.request
from datetime import datetime

import paramiko
import pysftp
import requests

SYSTEM = platform.system().lower()
AUTOMATION_RESULTS_FILE = "remote_connection_cuj_results.txt"
DUT_RESULTS_FILE = "DUT_automation_results.txt"
IN_PROGRESS_RESULTS_FILE = "in_progress_results.txt"
PRIVATE_KEY_PATH = os.path.expanduser("~/.ssh/testing_rsa")


def logging_definition():
    """Configure logging to write logs to a file."""
    logging.basicConfig(
        level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
    )


def logs_file_setup():
    """Setup file handler to store the logs inside it.

    Returns:
        logging file handler.
    """
    file_handler = logging.FileHandler(AUTOMATION_RESULTS_FILE)
    file_handler.setLevel(logging.INFO)
    file_handler.setFormatter(
        logging.Formatter("%(asctime)s - %(levelname)s - %(message)s")
    )
    logging.getLogger().addHandler(file_handler)
    return file_handler


def get_user_input():
    """Helper function to  get the user arguments
    The arguments will be used to access the target (DUT) machine and do the
    required processes.

    Returns:
    Tuple: A tuple containing the following configuration parameters:
        - target_ip: The IP address of the target (DUT) machine.
        - target_mac_address: The mac address of the target (DUT) machine.
        - username: The username for authentication.
        - status: The status for the starting point of the test.
        - test_path: Absolute path for the automation file.
    """
    parser = argparse.ArgumentParser(description="Automation Script for Target Machine")

    # Add required command-line arguments
    parser.add_argument("target_ip", help="Target machine IP address")
    parser.add_argument("target_mac_address", help="Target machine mac address")
    parser.add_argument("username", help="Target machine username")

    # Add an optional flag for automation running status
    parser.add_argument(
        "--status",
        choices=["sleep"],
        default="current",
        help="Automation running status (current or sleep ) mode",
    )

    parser.add_argument("test_path", help="Absolute path of the automation " "file")

    try:
        args = parser.parse_args()
        return args
    except SystemExit:
        logging.error("Error: Missing required arguments.")
        logging.error("Expected command:")
        logging.error(
            "python script.py <target_ip> <target_phy_address> <username> "
            "[--status <status>] <test_path>"
        )
        sys.exit(1)
    except FileNotFoundError as e:
        logging.error(e)
        sys.exit(1)


def extract_path_prefix(full_path, keyword):
    """Get the prefix for a given file path.

    Args:
        full_path: The full path of the file or directory.
        keyword: The word to search for in the path.

    Returns:
        The prefix for the given path.
    """
    return full_path.split(keyword)[0] + keyword


def push_files(test_path, remote_directory, sftp):
    """Push the required files and scripts to the remote DUT to execute the
    required automation test.

    Args:
        test_path: Absolute path for the automation file.
        remote_directory: Name of the directory where the automation file is
        stored in the remote machine.
        sftp: SFTP connection to copy the files to the remote machine.
    """
    cros_ca = extract_path_prefix(test_path, "cros_ca")
    sftp.put_r(cros_ca, remote_directory)


def copy_test_files(args):
    """Open SFTP connection with the DUT to push the files that will be used
    to run the automation code of the required use case.

    Args:
        args: Command line arguments passed by the user.

    Returns:
        remote_automation_directory: Name of the directory where the
        automation file is stored in the remote machine.
    """
    cnopts = pysftp.CnOpts()
    cnopts.hostkeys = None
    try:
        with pysftp.Connection(
            host=args.target_ip,
            port=22,
            username=args.username,
            private_key=PRIVATE_KEY_PATH,
            private_key_pass=None,
            cnopts=cnopts,
        ) as sftp:
            remote_automation_directory = "MyFiles/Automation Code"
            if not sftp.exists(remote_automation_directory):
                try:
                    sftp.mkdir(remote_automation_directory)
                except FileNotFoundError:
                    remote_automation_directory = "Automation Code"
                    sftp.mkdir(remote_automation_directory)

            logging.info("Pushing test files to the DUT")
            push_files(
                test_path=args.test_path,
                remote_directory=remote_automation_directory,
                sftp=sftp,
            )
            logging.info("Pushing completed successfully.")
            return remote_automation_directory
    except Exception as e:
        logging.error(f"SFTP transfer failed: {e}")


def install_wakeonlan_locally_windows():
    """Helper function to install Wake On Lan command line on a Windows
    operating system."""
    user_home = os.path.join("C:\\Users", getpass.getuser())
    url = "https://www.depicus.com/downloads/wolcmd.zip"
    response = requests.get(url, timeout=20)
    compressed_file_path = "wolcmd.zip"
    with open(compressed_file_path, "wb") as f:
        f.write(response.content)

    with zipfile.ZipFile(compressed_file_path, "r") as zip_ref:
        zip_ref.extractall("wolcmd")

    shutil.move("wolcmd/WolCmd.exe", os.path.join(user_home, "WolCmd.exe"))
    logging.info("wolcmd installed successfully for the current user on Windows.")

    if os.path.exists(compressed_file_path):
        os.remove(compressed_file_path)
        logging.info("Compressed file wolcmd.zip removed.")


def install_wakeonlan_locally_linux():
    """Helper function to install Wake On Lan command line on a linux
    operating system."""
    url = "https://github.com/jpoliv/wakeonlan/archive/master.tar.gz"
    install_dir = os.path.join(os.path.expanduser("~"), ".local", "bin")
    os.makedirs(install_dir, exist_ok=True)
    try:
        with urllib.request.urlopen(url) as response, open(
            "wakeonlan.tar.gz", "wb"
        ) as out_file:
            shutil.copyfileobj(response, out_file)
        with tarfile.open("wakeonlan.tar.gz", "r:gz") as tar:
            tar.extractall()
        extracted_dir = "wakeonlan-master"
        os.rename(extracted_dir, "wakeonlan")
        shutil.move(os.path.join("wakeonlan", "wakeonlan"), install_dir)
        logging.info("wakeonlan installed successfully locally.")
    except Exception as e:
        logging.error(f"Error installing wakeonlan: {e}")
    finally:
        os.remove("wakeonlan.tar.gz")
        shutil.rmtree("wakeonlan", ignore_errors=True)
    os.environ["PATH"] += os.pathsep + install_dir


def install_wolcmd():
    """Install Wake On Lan command for both Windows and Linux operating
    systems."""
    logging.info("Installing wakeonlan")
    if SYSTEM == "windows":
        install_wakeonlan_locally_windows()
    elif SYSTEM == "linux":
        install_wakeonlan_locally_linux()
    else:
        logging.info("Unsupported operating system.")


def uninstall_wolcmd():
    """Uninstall WCL command for both Windows and Linux operating systems."""
    logging.info("Removing wakeonlan.")
    if SYSTEM == "windows":
        user_home = os.path.join("C:\\Users", getpass.getuser())
        wolcmd_path = os.path.join(user_home, "WolCmd.exe")
        if os.path.exists(wolcmd_path):
            os.remove(wolcmd_path)
            logging.info("wolcmd uninstalled successfully from Windows.")
        else:
            logging.info("wolcmd is not installed on Windows.")
    elif SYSTEM == "linux":
        install_dir = os.path.join(os.path.expanduser("~"), ".local", "bin")
        # Define the path to the wakeonlan script
        wakeonlan_path = os.path.join(install_dir, "wakeonlan")
        try:
            # Remove the wakeonlan script
            os.remove(wakeonlan_path)
            logging.info("wakeonlan uninstalled successfully.")
        except FileNotFoundError:
            logging.error("wakeonlan is not installed.")
    else:
        logging.info("Unsupported operating system.")


def establish_ssh_connection(hostname, username, passphrase=None):
    """Establishes an SSH connection to a remote server.

    Args:
        hostname: The hostname or IP address of the remote server.
        username: The username for authentication.

    Returns:
        paramiko.SSHClient: An SSHClient object representing the established SSH connection.
    """
    ssh_client = paramiko.SSHClient()
    ssh_client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

    # Load the private key
    private_key = paramiko.RSAKey.from_private_key_file(
        PRIVATE_KEY_PATH, password=passphrase
    )

    ssh_client.connect(hostname=hostname, username=username, pkey=private_key)
    logging.info("Connected to remote server via SSH successfully!")
    return ssh_client


def is_ssh_connection_available(hostname, port=22):
    """Helper function to check the SSH connection availability every 1
    second.

    Args:
        hostname: The hostname or IP address of the remote server.
        port: the default port number for the socket.

    Returns:
        Boolean: True if the connection is available, False otherwise.
    """
    try:
        sock = socket.create_connection((hostname, port), timeout=1)
        sock.close()
        return True
    except socket.timeout:
        logging.warning(f"Connection to {hostname} timed out")
        return False
    except ConnectionRefusedError:
        logging.warning(f"Connection to {hostname} refused")
        return False
    except Exception as e:
        logging.error(f"An error occurred: {e}")
        return False


def receive_data_over_ssh(ssh_client):
    """Helper function used to get data from the remote client over SFTP
    protocol when it's available.

    The function will keep checking if the data available until the timout
    period is reached.

    Args:
        ssh_client: An SSHClient object representing the SSH connection.
    """
    logging.info("Fetching results from the DUT")
    sftp = ssh_client.open_sftp()
    local_file_path = os.path.join(os.getcwd(), DUT_RESULTS_FILE)
    timeout = 400
    start_time = time.time()

    while True:
        try:
            sftp.get(DUT_RESULTS_FILE, local_file_path)
            logging.info("Results received successfully")
            print_file_content(DUT_RESULTS_FILE)
            time.sleep(10)
            sftp.remove(IN_PROGRESS_RESULTS_FILE)
            break
        except IOError as e:
            if time.time() - start_time >= timeout:
                logging.error(f"Timeout reached. File not found. {e}")
                break
            time.sleep(2)
    sftp.close()


def send_data_over_ssh(
    args,
    remote_automation_directory,
):
    """Send data to a socket on a remote machine over SSH."""
    ssh_client = paramiko.SSHClient()

    try:
        ssh_client = establish_ssh_connection(
            hostname=args.target_ip,
            username=args.username,
        )
        ssh_transport = ssh_client.get_transport()
        client_agent_port = 8888
        ssh_channel = ssh_transport.open_channel(
            "direct-tcpip",
            (args.target_ip, client_agent_port),
            ("127.0.0.1", client_agent_port),
        )
        logging.info(
            f"SSH port forwarding established. {remote_automation_directory} "
            f"{str(os.path.basename(args.test_path))}"
        )
        execute_automation_command = (
            f"--status {args.status} -d '{remote_automation_directory}' -f"
            f" {str(os.path.basename(args.test_path))}"
        )

        with ssh_channel as channel:
            channel.sendall(execute_automation_command.encode())
            logging.info("Data sent to the socket server on the remote machine.")

        if args.status == "sleep":
            logging.info("Checking if the device enter sleep mode")
            while is_ssh_connection_available(hostname=args.target_ip, port=22):
                pass
            install_wolcmd()
            time.sleep(2)
            if SYSTEM == "windows":
                subprocess.run(
                    [
                        "wolcmd",
                        args.target_mac_address,
                        args.target_ip,
                        "255.255.255.0",
                        "7",
                    ],
                    check=True,
                )
            elif SYSTEM == "linux":
                subprocess.run(
                    [
                        "wakeonlan",
                        args.target_mac_address,
                    ],
                    check=True,
                )
            logging.info("Checking SSH connection availability on remote machine.")
            start_time = time.time()
            while time.time() - start_time < 120:
                connection_availability = is_ssh_connection_available(
                    hostname=args.target_ip, port=22
                )
                if connection_availability:
                    ssh_client = establish_ssh_connection(
                        hostname=args.target_ip,
                        username=args.username,
                    )
                    receive_data_over_ssh(ssh_client)
                    break
                logging.info("Attempting to reconnect...")
            else:
                logging.error(
                    "Timeout reached. Remote device may not have woken up "
                    "from sleep mode"
                )
            uninstall_wolcmd()
        else:
            receive_data_over_ssh(ssh_client)

    except paramiko.AuthenticationException:
        logging.error("Authentication failed. Please check your credentials.")
    except paramiko.SSHException as ssh_exception:
        logging.error(f"Unable to establish SSH connection: {ssh_exception}")
    except Exception as e:
        logging.error(f"An error occurred: {e}")
    finally:
        ssh_client.close()
        logging.info("Disconnected from remote server.")


def read_file(file_name):
    """Reads the content of the specified file.

    Args:
        file_name: The path to the file to be read.

    Returns:
        str: The content of the file as a string.
    """
    try:
        with open(file_name, "r", encoding="utf-8") as file:
            file_content = file.read()
        return file_content
    except FileNotFoundError as e:
        logging.error(f"The specified file does not exist: {e}")
    except IOError as e:
        logging.error(f"Error reading file '{file_name}'.\n:{e}")


def delete_file(file_name):
    """Deletes the specified file if it exists.

    Args:
        file_name: The path to the file to be deleted.
    """
    if os.path.exists(file_name):
        os.remove(file_name)
        logging.info(f"File {file_name} deleted successfully")
    else:
        logging.error(f"Results file '{file_name}' not found")


def print_file_content(file_name):
    """Prints the content of a given file.

    Args:
        file_name: The name of the file to print its content.
    """
    try:
        with open(file_name, "r", encoding="utf-8") as file:
            results_content = file.read()
            logging.info("File content of %s:\n%s", file_name, results_content)
    except FileNotFoundError:
        logging.error("File '%s' not found.", file_name)


def move_to_out(automation_path):
    """Moves the results file to the out directory.

    Args:
        automation_path: The absolute path of the automation test.
    """
    automation_parent_dir = "cros_ca"
    dir_path = extract_path_prefix(automation_path, automation_parent_dir)
    current_time = datetime.now().strftime("%Y%m%d-%H%M%S")
    results_dir = os.path.join(dir_path, rf"out/{current_time}")
    logging.info("Results directory Created")
    if not os.path.exists(results_dir):
        os.makedirs(results_dir)
    source_file = os.path.abspath(AUTOMATION_RESULTS_FILE)
    destination_file = os.path.join(results_dir, AUTOMATION_RESULTS_FILE)
    shutil.move(source_file, destination_file)
    logging.info("===========================================================")
    logging.info(f"Results saved to {results_dir}")


def main():
    """The entry point for the server agent.

    This function will perform the following:
    1. Set up the logging and logging file handler.
    2. Accept the arguments form the user and handle them.
    3. Copy the test required files to the DUT.
    4. Send test run command over SSH protocol to the DUT.
    5. Clean up the additional generated files in the server machine.
    6. Move the test results to the out directory.
    """
    logging_definition()
    file_handler = logs_file_setup()
    args = get_user_input()
    remote_automation_directory = copy_test_files(args=args)
    send_data_over_ssh(
        args=args, remote_automation_directory=remote_automation_directory
    )
    file_handler.close()
    delete_file(DUT_RESULTS_FILE)
    move_to_out(args.test_path)


if __name__ == "__main__":
    main()
