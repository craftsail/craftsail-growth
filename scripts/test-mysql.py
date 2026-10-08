#!/usr/bin/env python3
# SPDX-License-Identifier: AGPL-3.0-or-later
"""Run database acceptance in a disposable MySQL 8 container."""

import argparse
import os
from pathlib import Path
import secrets
import shutil
import signal
import subprocess
import sys
import tempfile
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", default="mysql:8.0.39", help="MySQL 8 Docker image")
    args = parser.parse_args()
    for command in ("docker", "go"):
        if shutil.which(command) is None:
            parser.error(f"{command} is required")
    root = Path(__file__).resolve().parent.parent
    name = "craftsail-mysql-test-" + secrets.token_hex(8)
    password = secrets.token_hex(24)
    result = 1
    # SIGTERM, Ctrl-C, failed startup and failed tests all enter the same cleanup.
    def terminate(signum, _frame):
        raise SystemExit(128 + signum)
    signal.signal(signal.SIGTERM, terminate)
    with tempfile.TemporaryDirectory(prefix="craftsail-mysql-test-") as directory:
        env_file = Path(directory) / "mysql.env"
        env_file.touch(mode=0o600)
        env_file.write_text(f"MYSQL_ROOT_PASSWORD={password}\nMYSQL_ROOT_HOST=%\n")
        try:
            print(f"Starting isolated {args.image}: {name}", flush=True)
            subprocess.run([
                "docker", "run", "--detach", "--rm", "--name", name,
                "--env-file", str(env_file), "--publish", "127.0.0.1::3306",
                args.image, "--character-set-server=utf8mb4",
                "--collation-server=utf8mb4_unicode_ci",
            ], check=True, stdout=subprocess.DEVNULL, timeout=300)
            address = subprocess.check_output(
                ["docker", "port", name, "3306/tcp"], text=True, timeout=10
            ).strip()
            deadline = time.monotonic() + 180
            while True:
                # Use container environment instead of putting a password in argv.
                ready = subprocess.run([
                    "docker", "exec", name, "sh", "-c",
                    'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysqladmin --protocol=tcp -h 127.0.0.1 -u root ping --silent',
                ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
                if ready.returncode == 0:
                    break
                if time.monotonic() >= deadline:
                    raise RuntimeError("MySQL did not become ready within 180 seconds")
                time.sleep(1)
            env = os.environ.copy()
            env["CRAFTSAIL_TEST_MYSQL_DSN"] = f"root:{password}@tcp({address})/?parseTime=true&loc=UTC"
            result = subprocess.run(
                ["go", "test", "./internal/integration", "-count=1", "-v", "-timeout=3m"],
                cwd=root, env=env,
            ).returncode
        finally:
            cleanup = subprocess.run(
                ["docker", "rm", "--force", "--volumes", name],
                stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True, timeout=30,
            )
            if cleanup.returncode and "No such container" not in cleanup.stderr:
                print(f"Could not remove temporary container {name}: {cleanup.stderr.strip()}", file=sys.stderr)
                result = 1
            else:
                print("Temporary MySQL container and volumes removed.", flush=True)
    return result


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, subprocess.SubprocessError) as exc:
        print(f"MySQL acceptance failed: {exc}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
