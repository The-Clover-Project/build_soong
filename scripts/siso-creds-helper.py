#!/usr/bin/env python3

import datetime
import json
import os
import shlex
import subprocess
import sys


def convert_unixdate_to_rfc3339(unix_date_str: str) -> str:
  """Converts a time string from Go's time.UnixDate format to

  an RFC 3339 formatted string.

  The UnixDate format is: "Mon Jan _2 15:04:05 MST 2006"
  (with a space-padded day).

  The RFC 3339 format is: "2006-01-02T15:04:05-07:00"

  Args:
      unix_date_str: A time string in time.UnixDate format.

  Returns:
      A time string in RFC 3339 format.
  """

  # Python's strptime %d directive expects a zero-padded day (e.g., "02").
  # Go's UnixDate format uses a space-padded day (e.g., " 2").
  # To make this robust, we split the string and manually add a '0'
  # if the day part (parts[2]) is a single digit.

  parts = unix_date_str.split()

  # The day is the 3rd element (index 2)
  # e.g., ["Mon", "Jan", "2", "15:04:05", "MST", "2006"]
  if len(parts[2]) == 1:
    parts[2] = "0" + parts[2]
    # Rejoin the string for parsing
    parseable_str = " ".join(parts)
  else:
    parseable_str = unix_date_str

  # Define the format string, now using %d for the zero-padded day
  # %Z handles parsing the timezone abbreviation (e.g., "MST", "JST")
  date_format = "%a %b %d %H:%M:%S %Z %Y"

  try:
    # Parse the string into an aware datetime object
    dt = datetime.datetime.strptime(parseable_str, date_format)

    # .isoformat() on an aware datetime object produces
    # a string that is compatible with RFC 3339.
    # e.g., "2006-01-02T15:04:05-07:00"
    return dt.isoformat()

  except ValueError as e:
    return f"Error parsing date string: {e}"


def run_credshelper():
  """Executes the 'credshelper' command and parses its stdout as JSON."""

  out_dir = os.environ.get("OUT_DIR")
  if not out_dir:
    out_dir = "out"
  cmd_file = os.path.join(out_dir, "soong", "rbe", "soong-convert-command")

  command = []
  try:
    with open(cmd_file, "r") as f:
      command_string = f.read().strip()
    if not command_string:
      print(f"Error: Command file '{cmd_file}' is empty.", file=sys.stderr)
      sys.exit(3)
    # Use shlex.split to correctly handle quoted arguments
    command = shlex.split(command_string)

  except FileNotFoundError:
    print(f"Error: Command file not found: '{cmd_file}'", file=sys.stderr)
    sys.exit(3)
  except Exception as e:
    print(
        f"Error reading or parsing command file '{cmd_file}': {e}",
        file=sys.stderr,
    )
    sys.exit(3)

  try:
    result = subprocess.run(
        command, capture_output=True, text=True, check=True, encoding="utf-8"
    )

    # Command succeeded, try to parse stdout as JSON
    try:
      data = json.loads(result.stdout)
      # Convert "expiry": time.UnixDate() into "expires": RFC3339 format, with
      # a trailing "Z", which is what Siso needs to see.
      resp = {
          "headers": {"Authorization": [f"Bearer {data['token']}"]},
          "expires": convert_unixdate_to_rfc3339(data["expiry"]) + "Z",
      }
      print(json.dumps(resp))
      return resp
    except json.JSONDecodeError:
      print(
          f"Error: {command} succeeded, but did not produce valid"
          f" JSON:\n{result.stdout}",
          file=sys.stderr,
      )
      sys.exit(2)

  except FileNotFoundError:
    print(
        f"Error: Command not found: '{command[0]}'\nPlease ensure 'credshelper'"
        " is installed and in your PATH.",
        file=sys.stderr,
    )
    sys.exit(127)

  except subprocess.CalledProcessError as e:
    # Command failed (returned a non-zero exit code)
    print(
        f"Error: Command '{' '.join(command)}' failed with exit code"
        f" {e.returncode}.",
        file=sys.stderr,
    )
    print(f"Stderr:\n{e.stderr}", file=sys.stderr)
    print(f"Stdout:\n{e.stdout}", file=sys.stderr)
    sys.exit(e.returncode)

  except Exception as e:
    # Catch any other unexpected errors
    print(f"An unexpected error occurred: {e}", file=sys.stderr)
    sys.exit(1)


if __name__ == "__main__":
  run_credshelper()
