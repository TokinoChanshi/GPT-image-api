import json
import os
import shutil
import socket
import subprocess
import sys
import time
import traceback
from pathlib import Path

import requests
from playwright.sync_api import sync_playwright


ROOT = Path(__file__).resolve().parent
BACKEND_DIR = ROOT / "backend"
FRONTEND_DIR = ROOT / "frontend"
ARTIFACTS_DIR = ROOT / "ui_test_artifacts"

BACKEND_BASE = os.environ.get("BACKEND_BASE", "http://127.0.0.1:8080").rstrip("/")
FRONTEND_PORT = int(os.environ.get("FRONTEND_PORT", "3010"))
FRONTEND_BASE = os.environ.get("FRONTEND_BASE", f"http://127.0.0.1:{FRONTEND_PORT}").rstrip("/")


def find_free_port(start_port: int, host: str = "127.0.0.1", max_tries: int = 20) -> int:
    port = start_port
    for _ in range(max_tries):
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            try:
                s.bind((host, port))
                return port
            except OSError:
                port += 1
    raise RuntimeError(f"no free port found from {start_port} (tries={max_tries})")


def wait_http_ok(url: str, timeout_s: int = 90) -> None:
    start = time.time()
    last_err = None
    while time.time() - start < timeout_s:
        try:
            r = requests.get(url, timeout=2)
            if 200 <= r.status_code < 500:
                return
            last_err = f"HTTP {r.status_code}"
        except Exception as e:
            last_err = repr(e)
        time.sleep(1)
    raise RuntimeError(f"timeout waiting for {url} (last_err={last_err})")


def start_process(cmd: list[str], cwd: Path, log_path: Path, env: dict | None = None) -> subprocess.Popen:
    log_path.parent.mkdir(parents=True, exist_ok=True)
    log_f = open(log_path, "w", encoding="utf-8", errors="replace")
    # Keep a handle on the file so it isn't GC closed early.
    proc = subprocess.Popen(
        cmd,
        cwd=str(cwd),
        stdout=log_f,
        stderr=subprocess.STDOUT,
        env=env,
        creationflags=getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0),
    )
    proc._codex_log_f = log_f  # type: ignore[attr-defined]
    return proc


def stop_process(proc: subprocess.Popen, name: str) -> None:
    if not proc:
        return
    if proc.poll() is not None:
        return
    try:
        proc.terminate()
        proc.wait(timeout=10)
        return
    except Exception:
        pass
    try:
        proc.kill()
    except Exception:
        pass


def safe_screenshot(page, path: Path, full_page: bool = True) -> None:
    try:
        page.screenshot(path=str(path), full_page=full_page, timeout=120_000)
    except Exception:
        # Screenshot is diagnostic only; don't fail the smoke test for it.
        return


def main() -> int:
    ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)

    backend_log = ARTIFACTS_DIR / "backend.log"
    frontend_log = ARTIFACTS_DIR / "frontend.log"

    go_exe = shutil.which("go") or shutil.which("go.exe")
    npm_exe = shutil.which("npm") or shutil.which("npm.cmd")
    node_exe = shutil.which("node") or shutil.which("node.exe")
    if not go_exe:
        raise RuntimeError("go executable not found in PATH")
    if not npm_exe:
        raise RuntimeError("npm executable not found in PATH")
    if not node_exe:
        raise RuntimeError("node executable not found in PATH")

    backend_proc = None
    frontend_proc = None

    results: list[str] = []
    console_errors: list[str] = []

    try:
        # 1) Start backend (8080)
        backend_exe = ARTIFACTS_DIR / "backend_smoke.exe"
        # Build once into artifacts to avoid go-run child process leaks.
        with open(backend_log, "w", encoding="utf-8", errors="replace") as blog:
            build = subprocess.run(
                [go_exe, "build", "-o", str(backend_exe), "."],
                cwd=str(BACKEND_DIR),
                stdout=blog,
                stderr=subprocess.STDOUT,
                env=os.environ.copy(),
            )
        if build.returncode != 0:
            raise RuntimeError(f"backend build failed (rc={build.returncode}), see {backend_log}")

        backend_proc = start_process(
            [str(backend_exe)],
            cwd=BACKEND_DIR,
            log_path=backend_log,
            env={**os.environ, "PORT": "8080"},
        )
        wait_http_ok(f"{BACKEND_BASE}/ping", timeout_s=120)
        if backend_proc.poll() is not None:
            raise RuntimeError(f"backend exited early (rc={backend_proc.returncode}), see {backend_log}")
        results.append(f"[OK] backend up: {BACKEND_BASE}/ping")

        # 2) Start frontend (vite dev server)
        port = find_free_port(FRONTEND_PORT)
        global FRONTEND_BASE
        FRONTEND_BASE = os.environ.get("FRONTEND_BASE", f"http://127.0.0.1:{port}").rstrip("/")

        vite_cli = FRONTEND_DIR / "node_modules" / "vite" / "bin" / "vite.js"
        if not vite_cli.exists():
            # Fallback to npm if node_modules missing
            frontend_cmd = [npm_exe, "run", "dev", "--", f"--port={port}", "--host=127.0.0.1", "--strictPort"]
        else:
            frontend_cmd = [node_exe, str(vite_cli), f"--port={port}", "--host=127.0.0.1", "--strictPort"]

        frontend_proc = start_process(
            frontend_cmd,
            cwd=FRONTEND_DIR,
            log_path=frontend_log,
            env=os.environ.copy(),
        )
        wait_http_ok(f"{FRONTEND_BASE}/", timeout_s=120)
        if frontend_proc.poll() is not None:
            raise RuntimeError(f"frontend exited early (rc={frontend_proc.returncode}), see {frontend_log}")
        results.append(f"[OK] frontend up: {FRONTEND_BASE}/")

        # 3) Headless browser smoke tests
        with sync_playwright() as p:
            browser = p.chromium.launch(headless=True)
            context = browser.new_context(viewport={"width": 1400, "height": 900}, accept_downloads=True)
            page = context.new_page()

            def on_console(msg):
                if msg.type in ("error",):
                    console_errors.append(f"[console:{msg.type}] {msg.text}")

            def on_page_error(err):
                console_errors.append(f"[pageerror] {err}")

            page.on("console", on_console)
            page.on("pageerror", on_page_error)

            # Dashboard
            page.goto(f"{FRONTEND_BASE}/", wait_until="domcontentloaded", timeout=60_000)
            page.wait_for_timeout(1000)
            safe_screenshot(page, ARTIFACTS_DIR / "01_dashboard.png", full_page=True)

            # Validate dashboard has the 4 stat cards rendered
            stat_values = page.locator(".stat-card .value")
            if stat_values.count() < 4:
                raise RuntimeError(f"dashboard stat cards not rendered, found {stat_values.count()}")
            results.append("[OK] dashboard rendered stat cards")

            # Navigate to accounts
            with page.expect_response(lambda r: "/v1/admin/accounts" in r.url and r.status == 200, timeout=60_000):
                page.click("text=NODE CLUSTER", timeout=15_000)
            page.wait_for_timeout(800)
            safe_screenshot(page, ARTIFACTS_DIR / "02_accounts.png", full_page=True)

            results.append("[OK] accounts initial load /v1/admin/accounts=200")

            with page.expect_response(lambda r: "/v1/admin/accounts" in r.url and r.status == 200, timeout=60_000):
                page.click("text=SYNC", timeout=15_000)
            results.append("[OK] SYNC button /v1/admin/accounts=200")

            # EXPORT: window.open to /v1/admin/accounts/export (response is attachment; may download or show JSON)
            with page.expect_popup(timeout=15_000) as pop:
                page.click("text=EXPORT", timeout=15_000)
            export_page = pop.value
            # Try to observe navigation; if it stays about:blank it's likely an attachment download.
            try:
                export_page.wait_for_url("**/v1/admin/accounts/export**", timeout=10_000)
                results.append("[OK] EXPORT opened popup to /v1/admin/accounts/export")
            except Exception:
                results.append(f"[WARN] EXPORT popup did not navigate (likely download), url={export_page.url}")
            safe_screenshot(export_page, ARTIFACTS_DIR / "03_export_popup.png", full_page=True)
            export_page.close()

            # Verify backend export endpoint returns JSON list
            r = requests.get(f"{BACKEND_BASE}/v1/admin/accounts/export", timeout=10)
            if r.status_code != 200:
                raise RuntimeError(f"backend export HTTP {r.status_code}")
            parsed = r.json()
            if not isinstance(parsed, list):
                raise RuntimeError("backend export is not a JSON list")
            (ARTIFACTS_DIR / "03_export_accounts.json").write_text(json.dumps(parsed, ensure_ascii=False), encoding="utf-8")
            results.append(f"[OK] backend /v1/admin/accounts/export JSON list (len={len(parsed)})")

            # IMPORT dialog open + upload a temp JSON
            page.click("text=IMPORT", timeout=15_000)
            page.wait_for_timeout(500)
            safe_screenshot(page, ARTIFACTS_DIR / "04_import_dialog.png", full_page=True)

            test_import = ARTIFACTS_DIR / "import_test.json"
            test_import.write_text(
                json.dumps(
                    [
                        {
                            "email": "ui_test@example.com",
                            "access_token": "at_ui_test",
                            "refresh_token": "rt_ui_test",
                            "session_id": "",
                            "account_id": "",
                        }
                    ],
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )

            file_input = page.locator("input[type=file]").first
            with page.expect_response(lambda r: "/v1/admin/accounts" in r.url and r.status == 200, timeout=60_000):
                with page.expect_response(lambda r: "/v1/admin/accounts/import" in r.url and r.status == 200, timeout=60_000):
                    file_input.set_input_files(str(test_import))
            # Expect a success toast
            page.wait_for_selector(".el-message", timeout=60_000)
            safe_screenshot(page, ARTIFACTS_DIR / "05_import_toast.png", full_page=True)
            results.append("[OK] IMPORT upload triggered toast")

            # Wait for accounts refresh after import
            results.append("[OK] IMPORT caused accounts refresh /v1/admin/accounts=200")

            # Navigate to API keys (static page)
            page.click("text=ACCESS KEYS", timeout=15_000)
            page.wait_for_timeout(600)
            safe_screenshot(page, ARTIFACTS_DIR / "06_apikeys.png", full_page=True)
            if page.locator("text=API Key Management").count() == 0:
                raise RuntimeError("APIKeys page not rendered")
            results.append("[OK] APIKeys page rendered")

            browser.close()

        # Console errors are not always fatal, but we report them.
        if console_errors:
            (ARTIFACTS_DIR / "console_errors.txt").write_text("\n".join(console_errors), encoding="utf-8")
            results.append(f"[WARN] console errors captured: {len(console_errors)} (see ui_test_artifacts/console_errors.txt)")
        else:
            results.append("[OK] no browser console errors captured")

        (ARTIFACTS_DIR / "summary.txt").write_text("\n".join(results), encoding="utf-8")
        print("\n".join(results))
        print(f"Artifacts: {ARTIFACTS_DIR}")
        return 0

    except Exception as e:
        (ARTIFACTS_DIR / "error.txt").write_text(traceback.format_exc(), encoding="utf-8")
        if results:
            (ARTIFACTS_DIR / "partial_summary.txt").write_text("\n".join(results), encoding="utf-8")
        print("UI smoke test FAILED:", e)
        print(f"See logs: {backend_log}, {frontend_log}, {ARTIFACTS_DIR / 'error.txt'}")
        return 2

    finally:
        try:
            if frontend_proc:
                stop_process(frontend_proc, "frontend")
        finally:
            if backend_proc:
                stop_process(backend_proc, "backend")


if __name__ == "__main__":
    sys.exit(main())
