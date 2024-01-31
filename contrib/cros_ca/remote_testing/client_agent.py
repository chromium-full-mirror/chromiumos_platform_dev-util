# pylint: disable= R1710,W1203, W0612, W0718
"""Client agent for the DUT machine to run test and connect with the server
agent"""

import argparse
import ctypes
import logging
import os
import shlex
import socket
import subprocess
import shutil
import time
import platform

SYSTEM = platform.system().lower()
IN_PROGRESS_RESULTS_FILE = "in_progress_results.txt"
RESULTS_FILE = "DUT_automation_results.txt"


def logging_definition():
    """Configure logging to write logs to a file."""
    logging.basicConfig(
        level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
    )


def logging_setup():
    """Setup file handler to store the logs inside it.

    Returns:
        logging file handler.
    """
    results_file_handler = logging.FileHandler(IN_PROGRESS_RESULTS_FILE)
    results_file_handler.setLevel(logging.INFO)
    results_file_handler.setFormatter(
        logging.Formatter("%(asctime)s - %(levelname)s - %(message)s")
    )
    logging.getLogger().addHandler(results_file_handler)
    return results_file_handler


def search_file_recursive(directory, file_name):
    """Searches for a file within a directory and its subdirectories.

    Args:
        directory: The directory to search within.
        file_name: The name of the file to search for.

    Returns:
        path of the file within the directory.
    """
    try:
        for root, dirs, files in os.walk(directory):
            if file_name in files:
                return os.path.join(root, file_name)
        raise FileNotFoundError(
            f"Error! Cannot find {file_name} in {directory} " f"directory"
        )
    except FileNotFoundError as e:
        logging.error(e)

    except Exception as e:
        logging.error(e)


def install_requirements_file(code_directory):
    """Helper function to install the contents of the requirements.txt file.

    Args:
        code_directory: The directory that contains the requirements.txt.
    """
    try:
        requirements_file_path = search_file_recursive(
            code_directory, "requirements.txt"
        )
        logging.info("Installing the requirements file")
        if SYSTEM == "linux":
            command = f'python3 -m pip install -r "{requirements_file_path}"'
            subprocess.run(
                command, check=True, capture_output=True, text=True, shell=True
            )
        else:
            command = f'pip install -r "{requirements_file_path}"'
            subprocess.run(
                command, shell=True, check=True, capture_output=True, text=True
            )

        logging.info("Finish installing the requirements file successfully")

    except subprocess.CalledProcessError as e:
        logging.error(f"Error while installing the requirements file: {e}")
    except FileNotFoundError as e:
        logging.error(e)


def run_automation_file(automation_code_directory, automation_file):
    """This function is responsible for running the use case code by
    executed its logic.

    Args:
        automation_code_directory: The name of the directory that contains the
        code.
        automation_file: The path of the automation file that contains the use
        case logic.
    """
    try:
        automation_file_path = search_file_recursive(
            automation_code_directory, automation_file
        )
        full_path = automation_file_path
        if SYSTEM == "linux":
            run_command = ["python3", full_path]
            subprocess.run(run_command, check=True)
        else:
            run_command = ["python", full_path]
            subprocess.run(run_command, shell=True, check=True)

    except subprocess.CalledProcessError as e:
        logging.error(f"Error while running the automation file: {e}")
    except FileNotFoundError as e:
        logging.error(f"Error! Cannot find the automation file: {e}")


def extract_command_values(command_string):
    """Helper function to extract the arguments from the command string.

    Args:
        command_string: The command received from the controller machine.

    Returns:
        A tuple containing the status, remote directory, and file name
        extracted from the command string.
    """
    parser = argparse.ArgumentParser()
    parser.add_argument("--status", required=True)
    parser.add_argument("-d", "--remote_directory", required=True)
    parser.add_argument("-f", "--file_name", required=True)
    input_list = shlex.split(command_string)
    args = parser.parse_args(input_list)
    return args.status, args.remote_directory, args.file_name


def delete_file(file_name):
    """Helper function to delete a given file from the local machine.

    Args:
        file_name: The name of the file to be deleted.
    """
    if os.path.exists(file_name):
        os.remove(file_name)
        logging.info(f"File {file_name} deleted successfully")
    else:
        logging.warning(f"Results file '{file_name}' not found")


def teardown_automation(automation_code_directory):
    """
    Teardown function to clean up resources after automation execution.

    Args:
        automation_code_directory (str): Path to the directory containing
        automation code.
    """
    try:
        # Remove the automation code directory
        if os.path.exists(automation_code_directory):
            logging.info("Deleting Automation Code directory")
            shutil.rmtree(automation_code_directory)
            logging.info("Successfully deleted Automation Code directory")
        else:
            logging.warning(
                f"Automation Code directory '{automation_code_directory}' not found"
            )
        time.sleep(3)
        delete_file(RESULTS_FILE)

    except Exception as e:
        logging.error(f"Error occurred during teardown: {e}")


def set_system_suspend_state():
    """Helper function to put the device in the sleep mode.

    This function will simulate the process that we execute it when sleep
    choice selected from the start menu.
    """
    system_power_state = ctypes.c_int
    suspend = 0
    ctypes.windll.powrprof.SetSuspendState(system_power_state(suspend), 0, 0)


def read_file_content(file_path):
    """Helper function to read the content of a given file path.

    Args:
        file_path: the path of the required file to be read.

    Returns:
        The content of the given file.
    """
    try:
        with open(str(file_path), "r", encoding="utf-8") as file:
            return file.readlines()
    except FileNotFoundError:
        return ["File not found."]


def append_to_results_file(file_name, code_directory):
    """Helper function to append the content of a given file to the results
    file.

    Args:
        file_name : The name of the file to be appended.
        code_directory: The directory of the automation tests.
    """
    full_path = search_file_recursive(rf"{code_directory}/out", file_name)
    file_content = read_file_content(full_path)
    with open(IN_PROGRESS_RESULTS_FILE, "a", encoding="utf-8") as file:
        file.writelines(file_content)


def rename_file():
    """Helper function to rename a specific file."""
    try:
        os.rename(IN_PROGRESS_RESULTS_FILE, RESULTS_FILE)
    except FileNotFoundError:
        print(f"File {IN_PROGRESS_RESULTS_FILE} not found")
    except FileExistsError:
        print(f"File {RESULTS_FILE} already exists")


def main():
    """The entry point of the target agent that will:
    1. Set up the logging and logging file handler.
    2. Listen to the SSH request from the server machine.
    3. Receive command from the server machine.
    4. Install the requirements file.
    5. Run the use case automation test based on the received status.
    6. Delete the files and directories for the automation test.
    """

    host = ""
    socket_port = 8888
    logging_definition()
    logs_file_handler = logging_setup()

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
        server_socket.bind((host, socket_port))
        server_socket.listen()

        logging.info(f"Server is listening on port {socket_port}")

        while True:
            conn, addr = server_socket.accept()
            with conn:

                logging.info(f"Connected by {addr}")
                server_command = conn.makefile().readline().strip()
                logging.info("Received command from the server agent.")

                if ("--status" and "-d" and "-f") in server_command:
                    status, code_directory, automation_file = (
                        extract_command_values(server_command)
                    )

                    install_requirements_file(code_directory=code_directory)

                    if status == "current":
                        logging.info(
                            f"Started test '{automation_file}' from the "
                            "current mode"
                        )
                        run_automation_file(code_directory, automation_file)
                        logging.info(f"Finish executing {automation_file} test")

                    elif status == "sleep":
                        logging.info(
                            f"Started test {automation_file} from sleep mode"
                        )
                        logging.info("Entering sleep mode")
                        time_before_sleep = time.time()
                        set_system_suspend_state()
                        time_after_wakeup = time.time()
                        logging.info("Waking up from sleep mode")
                        logging.info(f"Started test {automation_file}")
                        run_automation_file(code_directory, automation_file)
                        time_from_wakeup_to_finish = (
                            time.time() - time_after_wakeup
                        )
                        time_from_sleep_to_finish = (
                            time.time() - time_before_sleep
                        )
                        logging.info(
                            "Time from wake up until finish the test = "
                            f"{time_from_wakeup_to_finish}"
                        )
                        logging.info(
                            "Time from entering sleep mode until finish the "
                            f"test = {time_from_sleep_to_finish}"
                        )
                        logging.info(f"Finish executing {automation_file} test")

                    time.sleep(5)

                    append_to_results_file("logs.log", code_directory)
                    append_to_results_file("results.log", code_directory)
                    logs_file_handler.close()
                    rename_file()
                    logs_file_handler.close()
                    teardown_automation(code_directory)
                    return "Finish DUT Automation Code"


if __name__ == "__main__":
    main()
