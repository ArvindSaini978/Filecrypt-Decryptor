# ==============================================================================
# Filecrypt Container Decryptor v1.0
# Copyright (C) 2026 ArvindSaini978 (https://github.com/ArvindSaini978)
# Repository: https://github.com/ArvindSaini978/Filecrypt-Decryptor
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.
# ==============================================================================

import asyncio
import configparser
import ctypes
from datetime import datetime
import multiprocessing
import os
import re
import subprocess
import sys
from urllib.parse import urljoin, urlparse
import webbrowser

os.environ.setdefault("PLAYWRIGHT_BROWSERS_PATH", "0")

from playwright.async_api import Error as PlaywrightError, async_playwright
from rich import box
from rich.align import Align
from rich.console import Console, Group
from rich.markup import escape
from rich.panel import Panel
from rich.progress import BarColumn, Progress, SpinnerColumn, TaskProgressColumn, TextColumn
from rich.rule import Rule
from rich.table import Table

if os.name == "nt":
    import msvcrt
    os.system("chcp 65001 >nul 2>&1")
    try:
        sys.stdout.reconfigure(encoding="utf-8")
        sys.stderr.reconfigure(encoding="utf-8")
    except Exception:
        pass

console = Console(highlight=False)

UI_WIDTH = 108

if os.name == "nt":
    try:
        if console.size.width < UI_WIDTH + 2:
            os.system(f"mode con: cols={UI_WIDTH + 6} lines=42 >nul 2>&1")
    except Exception:
        pass

# ==============================================================================
# SOFT NORD / CATPPUCCIN MOCHA COLOR PALETTE
# ==============================================================================
BORDER_SUBTLE = "#3b4252"
BORDER_ACCENT = "#5e81ac"
TEXT_MAIN = "#eceff4"
TEXT_MUTED = "#7f849c"
ACCENT_TEAL = "#88c0d0"
ACCENT_LAVENDER = "#b4befe"
ACCENT_GREEN = "#a3be8c"
ACCENT_AMBER = "#ebcb8b"
ACCENT_CORAL = "#bf616a"
ACCENT_MAUVE = "#cba6f7"

if getattr(sys, "frozen", False):
    APP_DIR = os.path.dirname(sys.executable)
else:
    APP_DIR = os.path.dirname(os.path.abspath(__file__))

CONFIG_PATH = os.path.join(APP_DIR, "config.ini")

# ==============================================================================
# APPLICATION METADATA
# ==============================================================================
APP_TITLE = "FILECRYPT CONTAINER DECRYPTOR"
APP_VERSION = "v1.0"
APP_AUTHOR = "ArvindSaini978"
APP_REPO_URL = "https://github.com/ArvindSaini978/Filecrypt-Decryptor"
APP_GITHUB_PROFILE = "https://github.com/ArvindSaini978"
APP_LICENSE = "GPL-3.0 License"
APP_LICENSE_URL = "https://github.com/ArvindSaini978/Filecrypt-Decryptor/blob/master/LICENSE"
APP_YEAR = "2026"


def set_window_title(subtitle=""):
    """Updates the native Windows CMD / PowerShell / Terminal window title bar on Win 10 & 11."""
    full_title = f"Filecrypt Container Decryptor {APP_VERSION}"
    if subtitle:
        full_title = f"{full_title}  —  {subtitle}"
    if os.name == "nt":
        try:
            ctypes.windll.kernel32.SetConsoleTitleW(full_title)
        except Exception:
            pass
    else:
        sys.stdout.write(f"\033]0;{full_title}\007")
        sys.stdout.flush()


def clear_screen():
    if os.name == "nt":
        os.system("cls")
    else:
        console.clear()


def print_app_header(breadcrumb_path, cfg, clear=True, window_subtitle=None):
    """
    Renders a sleek, two-tier header card:
      Line 1: Centered Prominent App Title + v1.0
      Line 2: Full-width Location Breadcrumb + Subtle Author/License tag
    """
    if clear:
        clear_screen()

    parts = [p.strip() for p in breadcrumb_path.split("/")]
    set_window_title(
        window_subtitle if window_subtitle else " › ".join(parts[1:] or parts)
    )

    colored_parts = []
    for i, part in enumerate(parts):
        safe_part = escape(part)
        if i == len(parts) - 1:
            colored_parts.append(
                f"[bold {ACCENT_LAVENDER}]{safe_part}[/bold {ACCENT_LAVENDER}]"
            )
        else:
            colored_parts.append(f"[{TEXT_MUTED}]{safe_part}[/{TEXT_MUTED}]")
    crumb_str = f" [{TEXT_MUTED}]›[/{TEXT_MUTED}] ".join(colored_parts)

    centered_title = Align.center(
        f"[bold {ACCENT_TEAL}]◆   {APP_TITLE}[/bold {ACCENT_TEAL}]  "
        f"[bold {ACCENT_LAVENDER}]{APP_VERSION}[/bold {ACCENT_LAVENDER}]   "
        f"[bold {ACCENT_TEAL}]◆[/bold {ACCENT_TEAL}]"
    )

    bottom_row = Table.grid(expand=True)
    bottom_row.add_column(justify="left", ratio=1)
    bottom_row.add_column(justify="right")

    location_line = f"[{TEXT_MUTED}]Location:[/{TEXT_MUTED}] {crumb_str}"
    if len(breadcrumb_path) < 35:
        meta_badge = (
            f"[{TEXT_MUTED}]by[/{TEXT_MUTED}] [{ACCENT_TEAL}]{escape(APP_AUTHOR)}[/{ACCENT_TEAL}] "
            f"[{BORDER_SUBTLE}]│[/{BORDER_SUBTLE}] [{TEXT_MUTED}]{escape(APP_LICENSE)}[/{TEXT_MUTED}]"
        )
    else:
        meta_badge = ""

    bottom_row.add_row(location_line, meta_badge)

    console.print(
        Panel(
            Group(
                centered_title,
                Rule(style=BORDER_SUBTLE),
                bottom_row,
            ),
            box=box.ROUNDED,
            border_style=BORDER_ACCENT,
            padding=(0, 2),
            width=UI_WIDTH,
        )
    )


def render_settings_preview_panel(cfg):
    """Renders a compact, scannable Settings Preview section on the Home screen."""
    b_name = cfg.get("General", "browser", fallback="chrome").upper()
    dbg_on = cfg.getboolean("General", "debug_mode", fallback=False)
    invis = (
        cfg.get("Behavior", "resolve_mode", fallback="invisible") == "invisible"
    )
    clip_on = cfg.getboolean("Output", "copy_to_clipboard", fallback=False)
    keep_on = cfg.getboolean(
        "Session", "keep_session_across_containers", fallback=True
    )
    skip_on = cfg.getboolean("Behavior", "skip_offline_links", fallback=False)

    dbg_str = (
        f"[bold {ACCENT_AMBER}]ON[/bold {ACCENT_AMBER}]"
        if dbg_on
        else f"[{TEXT_MUTED}]OFF[/{TEXT_MUTED}]"
    )
    res_str = (
        f"[bold {ACCENT_TEAL}]INVISIBLE[/bold {ACCENT_TEAL}]"
        if invis
        else f"[bold {ACCENT_LAVENDER}]SINGLE TAB[/bold {ACCENT_LAVENDER}]"
    )
    clip_str = (
        f"[bold {ACCENT_GREEN}]YES[/bold {ACCENT_GREEN}]"
        if clip_on
        else f"[{TEXT_MUTED}]NO[/{TEXT_MUTED}]"
    )
    sess_str = (
        f"[bold {ACCENT_GREEN}]KEEP ALIVE[/bold {ACCENT_GREEN}]"
        if keep_on
        else f"[bold {ACCENT_AMBER}]RESET EACH[/bold {ACCENT_AMBER}]"
    )
    skip_str = (
        f"[bold {ACCENT_AMBER}]YES[/bold {ACCENT_AMBER}]"
        if skip_on
        else f"[{TEXT_MUTED}]NO[/{TEXT_MUTED}]"
    )

    grid = Table.grid(expand=True)
    grid.add_column(ratio=1)
    grid.add_column(ratio=1)
    grid.add_column(ratio=1)

    grid.add_row(
        f"[{TEXT_MUTED}]Browser      :[/{TEXT_MUTED}] [bold {ACCENT_LAVENDER}]{b_name}[/bold {ACCENT_LAVENDER}]",
        f"[{TEXT_MUTED}]Resolver   :[/{TEXT_MUTED}] {res_str}",
        f"[{TEXT_MUTED}]Session Mode :[/{TEXT_MUTED}] {sess_str}",
    )
    grid.add_row(
        f"[{TEXT_MUTED}]Auto-Copy    :[/{TEXT_MUTED}] {clip_str}",
        f"[{TEXT_MUTED}]Debug Logs :[/{TEXT_MUTED}] {dbg_str}",
        f"[{TEXT_MUTED}]Skip Offline :[/{TEXT_MUTED}] {skip_str}",
    )

    console.print(
        Panel(
            grid,
            title=f"[{TEXT_MUTED}]Current Settings Preview (Press [bold {ACCENT_MAUVE}]S[/bold {ACCENT_MAUVE}] to Customize)[/{TEXT_MUTED}]",
            title_align="left",
            box=box.ROUNDED,
            border_style=BORDER_SUBTLE,
            padding=(0, 2),
            width=UI_WIDTH,
        )
    )


def render_container_info_panel(
    container_title,
    total_links=None,
    total_hosts=None,
    host_name=None,
    host_links=None,
):
    """Renders the container title and link count cleanly across the full 108-col width."""
    info_grid = Table.grid(expand=True)
    info_grid.add_column(
        justify="left", ratio=1, overflow="ellipsis", no_wrap=True
    )
    info_grid.add_column(justify="right", no_wrap=True)

    left_text = f"[{TEXT_MUTED}]Container:[/{TEXT_MUTED}] [bold {TEXT_MAIN}]{escape(container_title)}[/bold {TEXT_MAIN}]"

    if host_name is not None:
        right_text = (
            f"  [{BORDER_SUBTLE}]│[/{BORDER_SUBTLE}]  "
            f"[{TEXT_MUTED}]Host:[/{TEXT_MUTED}] [bold {ACCENT_TEAL}]{escape(host_name)}[/bold {ACCENT_TEAL}] "
            f"[{TEXT_MUTED}]({host_links} links)[/{TEXT_MUTED}]"
        )
    else:
        right_text = (
            f"  [{BORDER_SUBTLE}]│[/{BORDER_SUBTLE}]  "
            f"[{TEXT_MUTED}]Found:[/{TEXT_MUTED}] [bold {ACCENT_TEAL}]{total_links}[/bold {ACCENT_TEAL}] "
            f"[{TEXT_MUTED}]links across[/{TEXT_MUTED}] [bold {ACCENT_TEAL}]{total_hosts}[/bold {ACCENT_TEAL}] "
            f"[{TEXT_MUTED}]hosts[/{TEXT_MUTED}]"
        )

    info_grid.add_row(left_text, right_text)

    console.print(
        Panel(
            info_grid,
            box=box.ROUNDED,
            border_style=BORDER_SUBTLE,
            padding=(0, 2),
            width=UI_WIDTH,
        )
    )


async def show_about_license_screen(cfg):
    """Displays an About, Author, GitHub Repository & GPL-3.0 License screen when the user presses 'I'."""
    while True:
        print_app_header("App / About & License", cfg, clear=True)

        about_body = (
            f"[bold {ACCENT_TEAL}]{APP_TITLE} {APP_VERSION}[/bold {ACCENT_TEAL}]\n\n"
            f"[{TEXT_MUTED}]Author     :[/{TEXT_MUTED}]  [bold {TEXT_MAIN}]{escape(APP_AUTHOR)}[/bold {TEXT_MAIN}]  [{TEXT_MUTED}]({escape(APP_GITHUB_PROFILE)})[/{TEXT_MUTED}]\n"
            f"[{TEXT_MUTED}]Repository :[/{TEXT_MUTED}]  [bold {ACCENT_LAVENDER}]{escape(APP_REPO_URL)}[/bold {ACCENT_LAVENDER}]\n"
            f"[{TEXT_MUTED}]License    :[/{TEXT_MUTED}]  [bold {ACCENT_GREEN}]GNU General Public License v3.0 (GPL-3.0)[/bold {ACCENT_GREEN}]\n"
            f"[{TEXT_MUTED}]Copyright  :[/{TEXT_MUTED}]  [{TEXT_MAIN}]Copyright © {APP_YEAR} {escape(APP_AUTHOR)}[/{TEXT_MAIN}]\n\n"
            f"[{TEXT_MUTED}]This program is free software: you can redistribute it and/or modify it under the terms of\n"
            f"the GNU General Public License as published by the Free Software Foundation, either version 3\n"
            f"of the License, or (at your option) any later version.\n\n"
            f"This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;\n"
            f"without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.\n"
            f"See the GNU General Public License for more details:\n"
            f"{escape(APP_LICENSE_URL)}[/{TEXT_MUTED}]\n\n"
            f"  [bold {ACCENT_TEAL}][G][/bold {ACCENT_TEAL}] Open GitHub Repository    "
            f"[bold {ACCENT_MAUVE}][L][/bold {ACCENT_MAUVE}] View Repository License    "
            f"[bold {ACCENT_GREEN}][Esc][/bold {ACCENT_GREEN}] Back to Home"
        )
        console.print(
            Panel(
                about_body,
                box=box.ROUNDED,
                border_style=BORDER_SUBTLE,
                padding=(1, 2),
                width=UI_WIDTH,
            )
        )
        key = await wait_for_single_key(
            f"\n  [{ACCENT_TEAL}]Press [G] GitHub Repo, [L] License, or [Esc] to return: [/{ACCENT_TEAL}]",
            valid_letters=["G", "L", "B", "Q", "I"],
            allow_esc=True,
        )
        if key == "G":
            webbrowser.open(APP_REPO_URL)
            continue
        elif key == "L":
            webbrowser.open(APP_LICENSE_URL)
            continue
        break


# ==============================================================================
# SHIPPING DEFAULT SETTINGS
# ==============================================================================
DEFAULT_CONFIG = {
    "General": {
        "browser": "chrome",
        "executable_path": "",
        "debug_mode": "false",
    },
    "Output": {
        "save_directory": "./decrypted_links",
        "copy_to_clipboard": "false",
        "include_container_title_in_filename": "true",
    },
    "Session": {
        "keep_session_across_containers": "true",
        "profile_directory": "./chrome_filecrypt_profile",
    },
    "Behavior": {
        "resolve_mode": "invisible",
        "skip_offline_links": "false",
        "request_delay": "0.2",
    },
}

SETTINGS_METADATA = {
    "1": {
        "section": "General",
        "key": "browser",
        "title": "Web Browser",
        "info": "Select which installed web browser Playwright should launch.",
        "warning": (
            "Make sure the browser you choose is installed on your PC (or set"
            " executable_path)."
        ),
        "options": [
            ("chrome", "Google Chrome", "Recommended"),
            ("msedge", "Microsoft Edge", "Pre-installed on Windows"),
            ("brave", "Brave Browser", "Auto-detects installation path"),
            (
                "firefox",
                "Mozilla Firefox",
                "Requires Playwright Firefox installed",
            ),
            (
                "chromium",
                "Playwright Chromium",
                "Requires Playwright Chromium installed",
            ),
        ],
    },
    "2": {
        "section": "General",
        "key": "debug_mode",
        "title": "Debug Logging Mode",
        "info": "Controls how much technical detail is printed in the console.",
        "warning": (
            "Keep OFF for a clean progress bar UI. Turn ON only if links fail"
            " and you need to debug."
        ),
        "options": [
            (
                "false",
                "OFF",
                "Clean App UI with Progress Bar (Recommended)",
            ),
            (
                "true",
                "ON",
                "Verbose Debug Logs (Shows every redirect hop, HTTP header & ad block)",
            ),
        ],
    },
    "3": {
        "section": "Behavior",
        "key": "resolve_mode",
        "title": "Link Resolution Method",
        "info": (
            "Controls how Filecrypt '/Link/' -> '/Go/' redirects are followed"
            " to get final URLs."
        ),
        "warning": (
            "'Invisible' resolves in memory with ZERO screen flashing."
            " 'Single Tab' uses one background tab."
        ),
        "options": [
            (
                "invisible",
                "INVISIBLE",
                "Zero-Flash In-Memory Resolver (Recommended - Fastest & Easy on Eyes)",
            ),
            (
                "single_tab",
                "SINGLE TAB",
                "Uses One Persistent Dark Background Tab (Fallback if Invisible is blocked)",
            ),
        ],
    },
    "4": {
        "section": "Output",
        "key": "copy_to_clipboard",
        "title": "Auto-Copy Decrypted URLs to Clipboard",
        "info": (
            "Automatically copies decrypted URLs to your Windows Clipboard when"
            " a host finishes."
        ),
        "warning": (
            "If YES, whatever text you currently have copied in your clipboard"
            " will be overwritten."
        ),
        "options": [
            (
                "false",
                "NO",
                "Only save to .txt file, do NOT touch my clipboard (Recommended Default)",
            ),
            (
                "true",
                "YES",
                "Copy decrypted URLs to Windows Clipboard automatically",
            ),
        ],
    },
    "5": {
        "section": "Output",
        "key": "include_container_title_in_filename",
        "title": "Include Container Title in Saved Filename",
        "info": (
            "Uses the container's title in the .txt filename (e.g."
            " 'Show_Name_1080p_mega_nz.txt')."
        ),
        "warning": (
            "If NO, files save as 'decrypted_mega_nz.txt' and WILL OVERWRITE"
            " previous containers from the same host!"
        ),
        "options": [
            (
                "true",
                "YES",
                "Save as '<Container_Title>_<Host>.txt' (Recommended - Prevents Overwriting!)",
            ),
            (
                "false",
                "NO",
                "Save as 'decrypted_<Host>.txt' (Warning: Overwrites previous file for same host!)",
            ),
        ],
    },
    "6": {
        "section": "Session",
        "key": "keep_session_across_containers",
        "title": "Session Persistence Across Containers ('N')",
        "info": (
            "Filecrypt uses a temporary session cookie (PHPSESSID) that only"
            " lives while the browser stays open."
        ),
        "warning": (
            "If KEEP ALIVE, pressing 'N' keeps the browser open so solved"
            " Captchas stay remembered! (Closing browser resets Captchas)."
        ),
        "options": [
            (
                "true",
                "KEEP ALIVE",
                "Keep Browser Open & Remember Solved Captchas During Session (Recommended)",
            ),
            (
                "false",
                "RESET EACH",
                "Clear Cookies & Start Fresh on Every New Container (Forces New Captcha)",
            ),
        ],
    },
    "7": {
        "section": "Behavior",
        "key": "skip_offline_links",
        "title": "Skip Offline / Dead Links",
        "info": (
            "Controls whether rows marked with a Red Offline icon on Filecrypt"
            " are skipped."
        ),
        "warning": (
            "Sometimes Filecrypt falsely marks working links as offline. Keep"
            " NO if you want every link."
        ),
        "options": [
            (
                "false",
                "NO",
                "Decrypt All Links for Selected Host (Recommended - Even if marked red/offline)",
            ),
            (
                "true",
                "YES",
                "Automatically Skip Links Marked Offline (Red Status Icon)",
            ),
        ],
    },
}


def write_clean_config_file(cfg):
    content = f"""# ====================================================================
# FILECRYPT CONTAINER DECRYPTOR {APP_VERSION} - CONFIGURATION FILE
# Author     : {APP_AUTHOR}
# Repository : {APP_REPO_URL}
# License    : {APP_LICENSE}
# Tip: You can edit these settings safely inside the app by pressing 'S'!
# ====================================================================

[General]
# Allowed: chrome, msedge, brave, firefox, chromium  (Recommended: chrome)
browser = {cfg.get("General", "browser", fallback="chrome")}

# Optional full path to custom browser .exe (Leave empty for default installation)
executable_path = {cfg.get("General", "executable_path", fallback="")}

# Allowed: true, false  (Recommended: false for clean UI)
debug_mode = {cfg.get("General", "debug_mode", fallback="false")}

[Output]
# Folder where decrypted .txt files are saved
save_directory = {cfg.get("Output", "save_directory", fallback="./decrypted_links")}

# Allowed: true, false  (Recommended: false so your clipboard isn't overwritten unexpectedly)
copy_to_clipboard = {cfg.get("Output", "copy_to_clipboard", fallback="false")}

# Allowed: true, false  (Recommended: true! If false, 'decrypted_mega_nz.txt' gets overwritten by new containers!)
include_container_title_in_filename = {cfg.get("Output", "include_container_title_in_filename", fallback="true")}

[Session]
# Allowed: true, false  (Recommended: true - Keeps browser open when pressing 'N' so solved Captchas are remembered!)
# Note: Closing the browser window always ends Filecrypt's temporary PHPSESSID session cookie.
keep_session_across_containers = {cfg.get("Session", "keep_session_across_containers", fallback="true")}

# Folder where browser profile is stored
profile_directory = {cfg.get("Session", "profile_directory", fallback="./chrome_filecrypt_profile")}

[Behavior]
# Allowed: invisible, single_tab  (Recommended: invisible for ZERO tab flashing!)
resolve_mode = {cfg.get("Behavior", "resolve_mode", fallback="invisible")}

# Allowed: true, false  (Recommended: false in case Filecrypt falsely reports a link offline)
skip_offline_links = {cfg.get("Behavior", "skip_offline_links", fallback="false")}

# Delay in seconds between links (Allowed: 0.0 to 10.0, Recommended: 0.2)
request_delay = {cfg.get("Behavior", "request_delay", fallback="0.2")}
"""
    with open(CONFIG_PATH, "w", encoding="utf-8") as f:
        f.write(content)


def reset_config_to_defaults():
    cfg = configparser.ConfigParser()
    for section, keys in DEFAULT_CONFIG.items():
        cfg.add_section(section)
        for k, v in keys.items():
            cfg.set(section, k, v)
    write_clean_config_file(cfg)
    return cfg


def validate_and_load_config():
    if not os.path.exists(CONFIG_PATH):
        return reset_config_to_defaults()

    while True:
        cfg = configparser.ConfigParser()
        errors = []

        try:
            cfg.read(CONFIG_PATH, encoding="utf-8")
        except configparser.Error as parse_err:
            errors.append(f"File syntax corrupted: {parse_err}")

        if not errors:
            if cfg.has_section("Session") and not cfg.has_option(
                "Session", "keep_session_across_containers"
            ):
                cfg.set("Session", "keep_session_across_containers", "true")
                write_clean_config_file(cfg)

            for section, keys in DEFAULT_CONFIG.items():
                if not cfg.has_section(section):
                    errors.append(f"Missing section: [{section}]")
                    continue
                for k in keys:
                    if not cfg.has_option(section, k):
                        errors.append(f"Missing setting: [{section}] {k}")

            for meta in SETTINGS_METADATA.values():
                sec, k = meta["section"], meta["key"]
                if cfg.has_option(sec, k):
                    val = cfg.get(sec, k).strip().lower()
                    allowed = [opt[0] for opt in meta["options"]]
                    if val not in allowed:
                        errors.append(
                            f"[{sec}] {k} = '{val}' is invalid! (Allowed: {', '.join(allowed)})"
                        )
                    else:
                        cfg.set(sec, k, val)

            if cfg.has_option("Behavior", "request_delay"):
                raw_delay = cfg.get("Behavior", "request_delay").strip()
                try:
                    d_val = float(raw_delay)
                    if d_val < 0 or d_val > 30:
                        errors.append(
                            f"[Behavior] request_delay = '{raw_delay}' must be between 0.0 and 30.0"
                        )
                except ValueError:
                    errors.append(
                        f"[Behavior] request_delay = '{raw_delay}' is not a valid number (Recommended: 0.2)"
                    )

            if cfg.has_option("General", "executable_path"):
                exe_p = cfg.get("General", "executable_path").strip()
                if exe_p and not os.path.exists(exe_p):
                    errors.append(
                        f"[General] executable_path = '{exe_p}' does not exist on this PC!"
                    )

        if not errors:
            return cfg

        clear_screen()
        set_window_title("Config Validator Warning")
        err_text = "\n".join(
            f"  [{ACCENT_CORAL}]•[/{ACCENT_CORAL}] [{TEXT_MAIN}]{escape(e)}[/{TEXT_MAIN}]"
            for e in errors
        )
        panel_body = (
            f"[bold {ACCENT_AMBER}]Your config.ini has corrupted or invalid settings:[/bold {ACCENT_AMBER}]\n\n"
            f"{err_text}\n\n"
            f"  [bold {ACCENT_TEAL}][R][/bold {ACCENT_TEAL}] Reset to Recommended Default Settings [{ACCENT_GREEN}](Recommended)[/{ACCENT_GREEN}]\n"
            f"  [bold {ACCENT_TEAL}][F][/bold {ACCENT_TEAL}] Fix Only Invalid Values & Keep My Other Settings\n"
            f"  [bold {ACCENT_TEAL}][E][/bold {ACCENT_TEAL}] Open config.ini in Notepad to Fix Manually"
        )
        console.print(
            Panel(
                panel_body,
                title=f"[bold {ACCENT_CORAL}]Config Validator Warning[/bold {ACCENT_CORAL}]",
                border_style=ACCENT_CORAL,
                box=box.ROUNDED,
                padding=(1, 2),
                width=UI_WIDTH,
            )
        )

        choice = (
            console.input(
                f"\n  [{ACCENT_TEAL}]Choose an option [R / F / E] (default: R): [/{ACCENT_TEAL}]"
            )
            .strip()
            .upper()
        )
        if choice == "E":
            if os.name == "nt":
                os.startfile(CONFIG_PATH)
            console.input(
                f"  [{ACCENT_AMBER}]Press Enter after saving config.ini in Notepad...[/{ACCENT_AMBER}]"
            )
            continue
        elif choice == "F":
            fixed_cfg = configparser.ConfigParser()
            for sec, keys in DEFAULT_CONFIG.items():
                fixed_cfg.add_section(sec)
                for k, default_val in keys.items():
                    if cfg.has_option(sec, k):
                        fixed_cfg.set(sec, k, cfg.get(sec, k).strip())
                    else:
                        fixed_cfg.set(sec, k, default_val)

            for meta in SETTINGS_METADATA.values():
                sec, k = meta["section"], meta["key"]
                val = fixed_cfg.get(sec, k).lower()
                allowed = [opt[0] for opt in meta["options"]]
                if val not in allowed:
                    fixed_cfg.set(sec, k, DEFAULT_CONFIG[sec][k])

            try:
                d_val = float(fixed_cfg.get("Behavior", "request_delay"))
                if d_val < 0 or d_val > 30:
                    raise ValueError
            except ValueError:
                fixed_cfg.set(
                    "Behavior",
                    "request_delay",
                    DEFAULT_CONFIG["Behavior"]["request_delay"],
                )

            exe_p = fixed_cfg.get("General", "executable_path")
            if exe_p and not os.path.exists(exe_p):
                fixed_cfg.set("General", "executable_path", "")

            write_clean_config_file(fixed_cfg)
            return fixed_cfg
        else:
            return reset_config_to_defaults()


def is_debug(cfg):
    return cfg.getboolean("General", "debug_mode", fallback=False)


def log(msg, level="INFO", cfg=None):
    if level == "DEBUG" and (cfg is None or not is_debug(cfg)):
        return
    timestamp = datetime.now().strftime("%H:%M:%S")
    styles = {
        "INFO": ACCENT_TEAL,
        "SUCCESS": ACCENT_GREEN,
        "WARN": ACCENT_AMBER,
        "ERROR": ACCENT_CORAL,
        "DEBUG": TEXT_MUTED,
    }
    col = styles.get(level, TEXT_MAIN)
    console.print(
        f"  [{TEXT_MUTED}]{timestamp}[/{TEXT_MUTED}]  [bold {col}]{level:<7}[/bold {col}]  [{TEXT_MAIN}]{escape(str(msg))}[/{TEXT_MAIN}]"
    )


def copy_text_to_clipboard(text):
    if os.name == "nt":
        try:
            p = subprocess.Popen("clip", stdin=subprocess.PIPE, shell=False)
            p.communicate(input=text.encode("utf-8"))
            return True
        except Exception:
            return False
    return False


def open_folder_in_explorer(folder_path):
    os.makedirs(folder_path, exist_ok=True)
    if os.name == "nt":
        os.startfile(folder_path)


async def read_text_with_esc(prompt_markup):
    """Custom character-by-character input reader with Rich prompt styling and `Esc` cancel support."""
    console.print(prompt_markup, end="")

    if os.name != "nt":
        res = await asyncio.to_thread(input)
        return res.strip()

    while msvcrt.kbhit():
        msvcrt.getwch()

    chars = []
    while True:
        if msvcrt.kbhit():
            ch = msvcrt.getwch()

            if ch == "\x03":
                raise KeyboardInterrupt

            if ch == "\x1b":
                console.print(f" [{ACCENT_AMBER}](Cancelled)[/{ACCENT_AMBER}]")
                return None

            if ch in ("\x00", "\xe0"):
                if msvcrt.kbhit():
                    msvcrt.getwch()
                continue

            if ch in ("\r", "\n"):
                sys.stdout.write("\n")
                sys.stdout.flush()
                return "".join(chars).strip()

            if ch in ("\x08", "\x7f"):
                if chars:
                    chars.pop()
                    sys.stdout.write("\b \b")
                    sys.stdout.flush()
                continue

            if ch.isprintable():
                chars.append(ch)
                sys.stdout.write(ch)
                sys.stdout.flush()

        await asyncio.sleep(0.02)


async def wait_for_single_key(
    prompt_markup, valid_letters, max_num=0, abort_check_fn=None, allow_esc=True
):
    console.print(prompt_markup, end="")

    if os.name != "nt":
        res = await asyncio.to_thread(input)
        return res.strip().upper()

    while msvcrt.kbhit():
        msvcrt.getwch()

    buffer = ""
    valid_letters_up = {k.upper() for k in valid_letters}

    while True:
        if abort_check_fn and abort_check_fn():
            sys.stdout.write("\n")
            sys.stdout.flush()
            return None

        if msvcrt.kbhit():
            ch = msvcrt.getwch()

            if ch == "\x03":
                raise KeyboardInterrupt

            if ch == "\x1b" and allow_esc:
                console.print(f"[{ACCENT_AMBER}]Esc[/{ACCENT_AMBER}]")
                return "ESC"

            if ch in ("\x00", "\xe0"):
                if msvcrt.kbhit():
                    msvcrt.getwch()
                continue

            if ch in ("\x08", "\x7f"):
                if buffer:
                    buffer = buffer[:-1]
                    sys.stdout.write("\b \b")
                    sys.stdout.flush()
                continue

            if ch in ("\r", "\n"):
                if buffer:
                    sys.stdout.write("\n")
                    sys.stdout.flush()
                    return buffer
                continue

            ch_up = ch.upper()

            if not buffer and ch_up in valid_letters_up:
                console.print(
                    f"[bold {ACCENT_TEAL}]{ch_up}[/bold {ACCENT_TEAL}]"
                )
                return ch_up

            if max_num > 0 and ch.isdigit():
                buffer += ch
                sys.stdout.write(ch)
                sys.stdout.flush()

                if int(buffer + "0") > max_num:
                    sys.stdout.write("\n")
                    sys.stdout.flush()
                    return buffer

                for _ in range(9):
                    await asyncio.sleep(0.05)
                    if msvcrt.kbhit():
                        break
                else:
                    sys.stdout.write("\n")
                    sys.stdout.flush()
                    return buffer

        await asyncio.sleep(0.05)


async def choose_setting_option_submenu(
    cfg, setting_id, parent_path="App / Settings"
):
    """Displays an explicit option sub-menu inside a soft-bordered Rich Table."""
    meta = SETTINGS_METADATA[setting_id]
    sec, key = meta["section"], meta["key"]
    current_val = cfg.get(sec, key).strip().lower()
    options = meta["options"]

    print_app_header(f"{parent_path} / {meta['title']}", cfg, clear=True)

    info_panel = (
        f"[bold {TEXT_MAIN}]{escape(meta['title'])}[/bold {TEXT_MAIN}]\n"
        f"[{TEXT_MUTED}]{escape(meta['info'])}[/{TEXT_MUTED}]\n"
        f"[{ACCENT_AMBER}]Note: {escape(meta['warning'])}[/{ACCENT_AMBER}]"
    )

    opt_table = Table(
        box=box.SIMPLE_HEAVY,
        border_style=BORDER_SUBTLE,
        expand=True,
        show_header=True,
        header_style=f"bold {TEXT_MUTED}",
        padding=(0, 1),
    )
    opt_table.add_column("Key", width=6, justify="center")
    opt_table.add_column("Option", width=22)
    opt_table.add_column("Description", ratio=1)
    opt_table.add_column("Status", width=14, justify="right")

    for idx, (opt_val, opt_label, opt_desc) in enumerate(options, 1):
        is_cur = opt_val == current_val
        key_cell = f"[bold {ACCENT_TEAL}][{idx}][/bold {ACCENT_TEAL}]"
        label_cell = (
            f"[bold {TEXT_MAIN}]{escape(opt_label)}[/bold {TEXT_MAIN}]"
            if is_cur
            else f"[{TEXT_MAIN}]{escape(opt_label)}[/{TEXT_MAIN}]"
        )
        desc_cell = f"[{TEXT_MUTED}]{escape(opt_desc)}[/{TEXT_MUTED}]"
        status_cell = (
            f"[bold {ACCENT_GREEN}]● CURRENT[/bold {ACCENT_GREEN}]"
            if is_cur
            else ""
        )
        opt_table.add_row(key_cell, label_cell, desc_cell, status_cell)

    opt_table.add_section()
    opt_table.add_row(
        f"[bold {ACCENT_AMBER}][Esc][/bold {ACCENT_AMBER}]",
        f"[{TEXT_MUTED}]Cancel[/{TEXT_MUTED}]",
        f"[{TEXT_MUTED}]Return to Settings without changing[/{TEXT_MUTED}]",
        "",
    )

    console.print(
        Panel(
            info_panel,
            box=box.ROUNDED,
            border_style=BORDER_SUBTLE,
            padding=(0, 2),
            width=UI_WIDTH,
        )
    )
    console.print(
        Panel(
            opt_table,
            box=box.ROUNDED,
            border_style=BORDER_SUBTLE,
            padding=(0, 1),
            width=UI_WIDTH,
        )
    )

    choice = await wait_for_single_key(
        f"\n  [{ACCENT_TEAL}]Select option [1-{len(options)}, Esc]: [/{ACCENT_TEAL}]",
        valid_letters=["B", "Q"],
        max_num=len(options),
        allow_esc=True,
    )

    if choice and choice.isdigit() and 1 <= int(choice) <= len(options):
        selected_val, selected_label, _ = options[int(choice) - 1]
        cfg.set(sec, key, selected_val)
        write_clean_config_file(cfg)
        return f"[{ACCENT_GREEN}]✓ Updated '{escape(meta['title'])}' to[/{ACCENT_GREEN}] [bold {TEXT_MAIN}]{escape(selected_label)}[/bold {TEXT_MAIN}]"
    return None


async def interactive_settings_menu(cfg, parent_path="App"):
    """Renders Settings in a clean, multi-column Rich Table where Values and Notes have distinct, soft colors."""
    status_banner = None
    settings_path = f"{parent_path} / Settings"

    while True:
        print_app_header(settings_path, cfg, clear=True)

        if status_banner:
            console.print(
                Panel(
                    f"  {status_banner}",
                    box=box.ROUNDED,
                    border_style=ACCENT_GREEN,
                    padding=(0, 1),
                    width=UI_WIDTH,
                )
            )

        table = Table(
            box=box.SIMPLE,
            border_style=BORDER_SUBTLE,
            expand=True,
            show_header=True,
            header_style=f"bold {TEXT_MUTED}",
            padding=(0, 1),
        )
        table.add_column("Key", width=6, justify="center")
        table.add_column("Setting", width=34)
        table.add_column("Current Value", width=18)
        table.add_column("Notes / Recommendation", ratio=1)

        # 1. Browser
        b_val = cfg.get("General", "browser", fallback="chrome").upper()
        table.add_row(
            f"[bold {ACCENT_TEAL}][1][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Web Browser[/{TEXT_MAIN}]",
            f"[bold {ACCENT_LAVENDER}]{b_val}[/bold {ACCENT_LAVENDER}]",
            f"[{TEXT_MUTED}]Chrome, Edge, Brave, Firefox[/{TEXT_MUTED}]",
        )

        # 2. Debug
        dbg_on = cfg.getboolean("General", "debug_mode", fallback=False)
        table.add_row(
            f"[bold {ACCENT_TEAL}][2][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Debug Logging Mode[/{TEXT_MAIN}]",
            f"[bold {ACCENT_AMBER}]ON[/bold {ACCENT_AMBER}]"
            if dbg_on
            else f"[bold {TEXT_MAIN}]OFF[/bold {TEXT_MAIN}]",
            f"[{ACCENT_AMBER}]Verbose logs active[/{ACCENT_AMBER}]"
            if dbg_on
            else f"[{TEXT_MUTED}]Clean UI (Recommended)[/{TEXT_MUTED}]",
        )

        # 3. Resolve Mode
        invis = (
            cfg.get("Behavior", "resolve_mode", fallback="invisible")
            == "invisible"
        )
        table.add_row(
            f"[bold {ACCENT_TEAL}][3][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Link Resolution Method[/{TEXT_MAIN}]",
            f"[bold {ACCENT_TEAL}]INVISIBLE[/bold {ACCENT_TEAL}]"
            if invis
            else f"[bold {ACCENT_LAVENDER}]SINGLE TAB[/bold {ACCENT_LAVENDER}]",
            f"[{TEXT_MUTED}]Zero-Flash In-Memory (Recommended)[/{TEXT_MUTED}]"
            if invis
            else f"[{TEXT_MUTED}]One Dark Background Tab[/{TEXT_MUTED}]",
        )

        # 4. Clipboard
        clip_on = cfg.getboolean("Output", "copy_to_clipboard", fallback=False)
        table.add_row(
            f"[bold {ACCENT_TEAL}][4][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Auto-Copy URLs to Clipboard[/{TEXT_MAIN}]",
            f"[bold {ACCENT_GREEN}]YES[/bold {ACCENT_GREEN}]"
            if clip_on
            else f"[bold {TEXT_MAIN}]NO[/bold {TEXT_MAIN}]",
            f"[{ACCENT_AMBER}]Overwrites Windows Clipboard[/{ACCENT_AMBER}]"
            if clip_on
            else f"[{TEXT_MUTED}]Save to .txt only (Recommended)[/{TEXT_MUTED}]",
        )

        # 5. Container Title in Filename
        title_on = cfg.getboolean(
            "Output", "include_container_title_in_filename", fallback=True
        )
        table.add_row(
            f"[bold {ACCENT_TEAL}][5][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Include Container Title in File[/{TEXT_MAIN}]",
            f"[bold {ACCENT_GREEN}]YES[/bold {ACCENT_GREEN}]"
            if title_on
            else f"[bold {ACCENT_CORAL}]NO[/bold {ACCENT_CORAL}]",
            f"[{TEXT_MUTED}]'<Title>_<Host>.txt' (Recommended)[/{TEXT_MUTED}]"
            if title_on
            else f"[{ACCENT_CORAL}]Warning: Overwrites previous host file![/{ACCENT_CORAL}]",
        )

        # 6. Session Mode
        keep_on = cfg.getboolean(
            "Session", "keep_session_across_containers", fallback=True
        )
        table.add_row(
            f"[bold {ACCENT_TEAL}][6][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Session Mode on New Container[/{TEXT_MAIN}]",
            f"[bold {ACCENT_GREEN}]KEEP ALIVE[/bold {ACCENT_GREEN}]"
            if keep_on
            else f"[bold {ACCENT_AMBER}]RESET EACH[/bold {ACCENT_AMBER}]",
            f"[{TEXT_MUTED}]Remembers solved Captchas while open[/{TEXT_MUTED}]"
            if keep_on
            else f"[{ACCENT_AMBER}]Forces new Captcha on every URL[/{ACCENT_AMBER}]",
        )

        # 7. Skip Offline
        skip_on = cfg.getboolean("Behavior", "skip_offline_links", fallback=False)
        table.add_row(
            f"[bold {ACCENT_TEAL}][7][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Skip Offline / Dead Links[/{TEXT_MAIN}]",
            f"[bold {ACCENT_AMBER}]YES[/bold {ACCENT_AMBER}]"
            if skip_on
            else f"[bold {TEXT_MAIN}]NO[/bold {TEXT_MAIN}]",
            f"[{ACCENT_AMBER}]Skips links marked Red[/{ACCENT_AMBER}]"
            if skip_on
            else f"[{TEXT_MUTED}]Decrypts all links (Recommended)[/{TEXT_MUTED}]",
        )

        # 8. Save Directory
        save_dir = cfg.get(
            "Output", "save_directory", fallback="./decrypted_links"
        )
        table.add_row(
            f"[bold {ACCENT_TEAL}][8][/bold {ACCENT_TEAL}]",
            f"[{TEXT_MAIN}]Change Save Directory[/{TEXT_MAIN}]",
            f"[{ACCENT_TEAL}]{escape(save_dir)}[/{ACCENT_TEAL}]",
            f"[{TEXT_MUTED}]Output folder for .txt files[/{TEXT_MUTED}]",
        )

        table.add_section()
        table.add_row(
            f"[bold {ACCENT_AMBER}][R][/bold {ACCENT_AMBER}]",
            f"[{TEXT_MAIN}]Reset All Settings[/{TEXT_MAIN}]",
            "",
            f"[{TEXT_MUTED}]Restore recommended defaults[/{TEXT_MUTED}]",
        )
        table.add_row(
            f"[bold {ACCENT_MAUVE}][E][/bold {ACCENT_MAUVE}]",
            f"[{TEXT_MAIN}]Edit config.ini in Notepad[/{TEXT_MAIN}]",
            "",
            f"[{TEXT_MUTED}]Open raw configuration file[/{TEXT_MUTED}]",
        )
        table.add_row(
            f"[bold {ACCENT_GREEN}][Esc][/bold {ACCENT_GREEN}]",
            f"[{TEXT_MAIN}]Back to {escape(parent_path)}[/{TEXT_MAIN}]",
            "",
            "",
        )

        console.print(
            Panel(
                table,
                box=box.ROUNDED,
                border_style=BORDER_SUBTLE,
                padding=(0, 1),
                width=UI_WIDTH,
            )
        )

        key = await wait_for_single_key(
            f"\n  [{ACCENT_TEAL}]Select a setting [1-8, R, E, Esc]: [/{ACCENT_TEAL}]",
            valid_letters=["R", "E", "B", "Q"],
            max_num=8,
            allow_esc=True,
        )

        if key in ("ESC", "B", "Q", None):
            break
        elif key in SETTINGS_METADATA:
            msg = await choose_setting_option_submenu(
                cfg, key, parent_path=settings_path
            )
            if msg:
                status_banner = msg
        elif key == "8":
            print_app_header(
                f"{settings_path} / Save Directory", cfg, clear=True
            )
            dir_panel = (
                f"[{TEXT_MUTED}]Current Folder:[/{TEXT_MUTED}]  [bold {ACCENT_TEAL}]{escape(os.path.abspath(save_dir))}[/bold {ACCENT_TEAL}]\n"
                f"[{TEXT_MUTED}]Instructions  :[/{TEXT_MUTED}]  Type or paste a new folder path and press [bold {ACCENT_GREEN}]Enter[/bold {ACCENT_GREEN}], or press [bold {ACCENT_AMBER}]Esc[/bold {ACCENT_AMBER}] to cancel."
            )
            console.print(
                Panel(
                    dir_panel,
                    box=box.ROUNDED,
                    border_style=BORDER_SUBTLE,
                    padding=(1, 2),
                    width=UI_WIDTH,
                )
            )
            new_dir = await read_text_with_esc(
                f"\n  [{ACCENT_TEAL}]New folder path (or Esc to cancel): [/{ACCENT_TEAL}]"
            )
            if new_dir:
                cfg.set("Output", "save_directory", new_dir)
                write_clean_config_file(cfg)
                status_banner = f"[{ACCENT_GREEN}]✓ Save directory updated to:[/{ACCENT_GREEN}] [bold {TEXT_MAIN}]{escape(new_dir)}[/bold {TEXT_MAIN}]"
            else:
                status_banner = f"[{TEXT_MUTED}]Save directory change cancelled.[/{TEXT_MUTED}]"
        elif key == "R":
            print_app_header(
                f"{settings_path} / Reset Defaults", cfg, clear=True
            )
            confirm = await wait_for_single_key(
                f"\n  [{ACCENT_AMBER}]Reset ALL settings to recommended defaults? [Y / N / Esc]: [/{ACCENT_AMBER}]",
                valid_letters=["Y", "N"],
                allow_esc=True,
            )
            if confirm == "Y":
                new_cfg = reset_config_to_defaults()
                for sec in new_cfg.sections():
                    for k, v in new_cfg.items(sec):
                        cfg.set(sec, k, v)
                status_banner = f"[{ACCENT_GREEN}]✓ All settings restored to recommended defaults![/{ACCENT_GREEN}]"
        elif key == "E":
            write_clean_config_file(cfg)
            if os.name == "nt":
                os.startfile(CONFIG_PATH)
            console.input(
                f"\n  [{ACCENT_AMBER}]Press Enter after saving config.ini in Notepad...[/{ACCENT_AMBER}]"
            )
            validated = validate_and_load_config()
            for sec in validated.sections():
                for k, v in validated.items(sec):
                    cfg.set(sec, k, v)
            status_banner = f"[{ACCENT_GREEN}]✓ Reloaded & validated config.ini from Notepad![/{ACCENT_GREEN}]"


ANTI_AD_INIT_SCRIPT = """
(() => {
    const applyDarkBg = () => {
        if (document.documentElement && (!location.href || location.href === 'about:blank')) {
            document.documentElement.style.backgroundColor = '#0d0f14';
            if (document.body) document.body.style.backgroundColor = '#0d0f14';
        }
    };
    applyDarkBg();

    window.__ALLOW_POPUPS__ = false;
    window.__CAPTURED_LINK_URL__ = null;

    const origOpen = window.open;
    window.open = function(url, target, features) {
        if (window.__CAPTURE_MODE__) {
            window.__CAPTURED_LINK_URL__ = url;
            return null;
        }
        if (!window.__ALLOW_POPUPS__) {
            return null;
        }
        return origOpen.call(window, url, target, features);
    };

    const cleanAds = () => {
        applyDarkBg();
        document.querySelectorAll('#ddownload-rect, [id*="-rect"], [class*="sponsor"]').forEach(el => {
            el.remove();
        });

        document.querySelectorAll('*').forEach(el => {
            if (el.shadowRoot) {
                if (el.shadowRoot.querySelector('.rc-link, .rc-ad, a[href*="/Link/"]')) {
                    el.remove();
                }
            }
        });

        const isCaptchaPage = !!document.querySelector('form#cform, .protection');
        if (isCaptchaPage && !document.querySelector('table tbody tr td.status')) {
            document.querySelectorAll('a[href*="/Link/"], a[target="_blank"]').forEach(a => {
                if (!a.href.includes('cutcaptcha')) {
                    a.remove();
                }
            });
        }
    };

    document.addEventListener('DOMContentLoaded', () => {
        cleanAds();
        const obs = new MutationObserver(cleanAds);
        obs.observe(document.documentElement, { childList: true, subtree: true });
    });
    setInterval(cleanAds, 200);
})();
"""


async def safe_close_page(page):
    try:
        if page and not page.is_closed():
            await page.close()
    except Exception:
        pass


async def safe_close_context(context):
    try:
        if context:
            await context.close()
    except Exception:
        pass


def is_context_alive(context):
    try:
        if context is None:
            return False
        _ = context.pages
        return len(context.pages) > 0
    except Exception:
        return False


async def freeze_page_and_extract_container_info(page, cfg):
    log(
        "Freezing page timers and parsing container title + table...",
        "DEBUG",
        cfg,
    )

    info = await page.evaluate("""() => {
        let highestId = window.setTimeout(() => {}, 0);
        for (let i = 0; i <= highestId; i++) {
            window.clearTimeout(i);
            window.clearInterval(i);
        }
        window.setTimeout = function() { return 0; };
        window.setInterval = function() { return 0; };

        const h2 = document.querySelector('.container h2, .box h2, h2');
        const containerTitle = h2 ? h2.innerText.trim() : 'container';

        const rows = Array.from(document.querySelectorAll('table tbody tr'));
        const parsed = [];

        rows.forEach((tr, idx) => {
            tr.setAttribute('data-pw-index', idx);

            const titleTd = tr.querySelector('td[title]');
            let filename = titleTd ? titleTd.getAttribute('title').trim() : 'unknown';

            let host = 'unknown';
            if (titleTd) {
                const hostSpan = titleTd.querySelector('span, a');
                if (hostSpan && hostSpan.innerText.trim()) {
                    host = hostSpan.innerText.trim().toLowerCase();
                } else {
                    const lines = titleTd.innerText.split('\\n').map(s => s.trim()).filter(Boolean);
                    if (lines.length > 1) host = lines[lines.length - 1].toLowerCase();
                }
            }

            const tds = Array.from(tr.querySelectorAll('td'));
            let size = 'unknown';
            for (const td of tds) {
                const txt = td.innerText.trim();
                if (/^\\d+(\\.\\d+)?\\s*(KB|MB|GB|TB|B|Bytes)$/i.test(txt)) {
                    size = txt;
                    break;
                }
            }

            const statusTd = tr.querySelector('td.status');
            let hasExplicitOfflineIcon = false;
            if (statusTd) {
                if (statusTd.querySelector('i.offline, .offline, [class*="offline"], [title*="offline" i], .status_2')) {
                    hasExplicitOfflineIcon = true;
                } else if (/\\boffline\\b/i.test(statusTd.className)) {
                    hasExplicitOfflineIcon = true;
                }
            }

            let isTinyDeadPlaceholder = (
                filename === 'undefined' ||
                filename === 'unknown' ||
                /^\\d+\\s*(b|bytes)$/i.test(size)
            );
            let hasValidRealFile = (
                !isTinyDeadPlaceholder &&
                /^\\d+(\\.\\d+)?\\s*(KB|MB|GB|TB)$/i.test(size)
            );

            let isOnline = !(hasExplicitOfflineIcon || isTinyDeadPlaceholder);

            const btn = tr.querySelector('button, a.download, td:last-child a, td:last-child button');
            if (btn) {
                btn.setAttribute('data-pw-btn', idx);

                let directLink = null;
                const allEls = [tr, btn, ...Array.from(tr.querySelectorAll('*'))];
                for (const el of allEls) {
                    if (!el.attributes) continue;
                    for (const attr of Array.from(el.attributes)) {
                        const val = attr.value || '';
                        const match = val.match(/(?:https?:\\/\\/[^'\"\\s]+)?\\/Link\\/[A-Za-z0-9]+\\.html/i);
                        if (match) {
                            directLink = match[0];
                            break;
                        }
                    }
                    if (directLink) break;
                }

                parsed.push({
                    index: idx,
                    rowid: tr.getAttribute('data-rowid') || '',
                    filename: filename,
                    host: host,
                    size: size,
                    is_online: isOnline,
                    has_valid_real_file: hasValidRealFile,
                    direct_link: directLink
                });
            }
        });

        return {
            title: containerTitle,
            rows: parsed
        };
    }""")

    rows = info.get("rows", [])
    by_host = {}
    for r in rows:
        by_host.setdefault(r["host"], []).append(r)

    for h, h_rows in by_host.items():
        online_count = sum(1 for r in h_rows if r.get("is_online"))
        if len(h_rows) > 1 and (online_count / len(h_rows)) >= 0.75:
            for r in h_rows:
                if not r.get("is_online") and r.get("has_valid_real_file"):
                    r["is_online"] = True

    return info


async def harvest_intermediate_link_url(
    context, main_page, blank_keepalive_page, row_info, cfg
):
    if main_page.is_closed():
        return None

    if row_info.get("direct_link"):
        full_link = urljoin(main_page.url, row_info["direct_link"])
        log(f"Found direct /Link/ URL in DOM: {full_link}", "DEBUG", cfg)
        return full_link

    idx = row_info["index"]
    btn_selector = f"[data-pw-btn='{idx}']"
    loop = asyncio.get_running_loop()
    captured_future = loop.create_future()

    def on_request(req):
        if "/link/" in req.url.lower() and not captured_future.done():
            captured_future.set_result(req.url)

    context.on("request", on_request)

    try:
        await main_page.evaluate(
            "window.__CAPTURE_MODE__ = true; window.__CAPTURED_LINK_URL__ = null;"
        )

        btn = main_page.locator(btn_selector).first
        await btn.scroll_into_view_if_needed()
        await btn.click(force=True)

        for _ in range(15):
            if captured_future.done() or main_page.is_closed():
                break
            js_captured = await main_page.evaluate(
                "window.__CAPTURED_LINK_URL__"
            )
            if js_captured and "/link/" in js_captured.lower():
                captured_future.set_result(urljoin(main_page.url, js_captured))
                break
            await asyncio.sleep(0.1)

        link_url = await asyncio.wait_for(captured_future, timeout=5.0)
        log(f"Harvested intermediate /Link/ URL: {link_url}", "DEBUG", cfg)
        return link_url

    except Exception as e:
        if not main_page.is_closed():
            log(
                f"Could not harvest /Link/ URL for row {idx}: {e}",
                "DEBUG",
                cfg,
            )
        return None
    finally:
        if not main_page.is_closed():
            try:
                await main_page.evaluate("window.__CAPTURE_MODE__ = false;")
            except Exception:
                pass
        context.remove_listener("request", on_request)
        for p in list(context.pages):
            if p not in (main_page, blank_keepalive_page):
                await safe_close_page(p)


def is_external_target(url):
    if not url or url.startswith(("about:", "chrome", "data:", "blob:")):
        return False
    domain = urlparse(url).netloc.lower()
    ignored = ["filecrypt.", "cutcaptcha.", "cloudflare."]
    return bool(domain) and not any(ig in domain for ig in ignored)


async def resolve_link_invisibly(context, main_page, link_url, cfg):
    current_url = link_url
    referer = main_page.url

    for hop in range(6):
        log(f"[Invisible Hop {hop+1}] GET {current_url}", "DEBUG", cfg)
        resp = await context.request.get(
            current_url,
            headers={"Referer": referer},
            max_redirects=0,
            timeout=15000,
        )

        if resp.status in (301, 302, 303, 307, 308):
            loc = resp.headers.get("location")
            if loc:
                next_url = urljoin(current_url, loc)
                log(
                    f"[Invisible Location Header] {next_url}",
                    "DEBUG",
                    cfg,
                )
                if is_external_target(next_url):
                    return next_url
                referer = current_url
                current_url = next_url
                continue

        body = await resp.text()

        go_match = re.search(
            r"(?:https?://[^'\"\s]+)?/Go/[A-Za-z0-9]+\.html", body, re.I
        )
        if go_match:
            next_url = urljoin(current_url, go_match.group(0))
            log(f"[Invisible Parsed /Go/ URL] {next_url}", "DEBUG", cfg)
            referer = current_url
            current_url = next_url
            continue

        ext_match = re.search(
            r"(?:location\.href\s*=\s*|url=)['\"]?(https?://[^'\"\s>]+)",
            body,
            re.I,
        )
        if ext_match:
            candidate = ext_match.group(1)
            if is_external_target(candidate):
                return candidate

        break

    return None


def format_resolved_item_markup(
    index_num, total_num, filename, final_url, success=True
):
    idx_str = f"[{ACCENT_TEAL}][{index_num:02d}/{total_num:02d}][/{ACCENT_TEAL}]"
    safe_fname = escape(filename)
    safe_url = escape(final_url)
    col = ACCENT_GREEN if success else ACCENT_CORAL
    return (
        f"  {idx_str} [{TEXT_MAIN}]{safe_fname}[/{TEXT_MAIN}]\n"
        f"         [{TEXT_MUTED}]╰──›[/{TEXT_MUTED}] [bold {col}]{safe_url}[/bold {col}]"
    )


def print_resolved_item(index_num, total_num, filename, final_url, success=True):
    console.print(
        format_resolved_item_markup(
            index_num, total_num, filename, final_url, success
        )
    )


async def resolve_link_smart(
    context,
    main_page,
    worker_holder,
    link_url,
    filename,
    index_num,
    total_num,
    cfg,
):
    if main_page.is_closed():
        return None

    resolve_mode = (
        cfg.get("Behavior", "resolve_mode", fallback="invisible")
        .strip()
        .lower()
    )

    if resolve_mode == "invisible":
        try:
            final_url = await resolve_link_invisibly(
                context, main_page, link_url, cfg
            )
            if final_url and (
                "mega.nz" not in final_url.lower() or "#" in final_url
            ):
                print_resolved_item(
                    index_num, total_num, filename, final_url, success=True
                )
                return final_url
            log(
                "Invisible resolver needed DOM fallback, using persistent dark background tab...",
                "DEBUG",
                cfg,
            )
        except Exception as e:
            log(f"Invisible resolver note: {e}", "DEBUG", cfg)

    if worker_holder["page"] is None or worker_holder["page"].is_closed():
        worker_holder["page"] = await context.new_page()
        await main_page.bring_to_front()

    worker_page = worker_holder["page"]
    loop = asyncio.get_running_loop()
    final_url_future = loop.create_future()

    def on_worker_response(response):
        try:
            if response.status in (301, 302, 303, 307, 308):
                loc = response.headers.get("location")
                if loc:
                    full_loc = urljoin(response.url, loc)
                    log(
                        f"[HTTP {response.status} Location Header] {full_loc}",
                        "DEBUG",
                        cfg,
                    )
                    if (
                        is_external_target(full_loc)
                        and not final_url_future.done()
                    ):
                        final_url_future.set_result(full_loc)
        except Exception:
            pass

    def on_frame_navigated(frame):
        try:
            if frame == worker_page.main_frame:
                url = frame.url
                if is_external_target(url) and not final_url_future.done():
                    final_url_future.set_result(url)
        except Exception:
            pass

    async def delayed_request_fallback(req_url):
        await asyncio.sleep(0.35)
        if not final_url_future.done():
            current_page_url = worker_page.url
            if (
                is_external_target(current_page_url)
                and "#" in current_page_url
            ):
                final_url_future.set_result(current_page_url)
            else:
                final_url_future.set_result(req_url)

    def on_worker_request(request):
        try:
            if request.is_navigation_request():
                url = request.url
                if is_external_target(url) and not final_url_future.done():
                    if (
                        is_external_target(worker_page.url)
                        and "#" in worker_page.url
                    ):
                        final_url_future.set_result(worker_page.url)
                    else:
                        asyncio.create_task(delayed_request_fallback(url))
        except Exception:
            pass

    worker_page.on("response", on_worker_response)
    worker_page.on("framenavigated", on_frame_navigated)
    worker_page.on("request", on_worker_request)

    try:
        await main_page.bring_to_front()

        goto_task = asyncio.create_task(
            worker_page.goto(link_url, wait_until="commit", timeout=20000)
        )

        final_url = await asyncio.wait_for(final_url_future, timeout=20.0)

        if "mega.nz" in final_url.lower() and "#" not in final_url:
            for _ in range(10):
                if "#" in worker_page.url:
                    final_url = worker_page.url
                    break
                await asyncio.sleep(0.1)

        print_resolved_item(
            index_num, total_num, filename, final_url, success=True
        )

        if not goto_task.done():
            goto_task.cancel()
        else:
            try:
                await goto_task
            except Exception:
                pass

        return final_url

    except asyncio.TimeoutError:
        print_resolved_item(
            index_num,
            total_num,
            filename,
            "[TIMEOUT - Failed to resolve within 20s]",
            success=False,
        )
        return None
    finally:
        worker_page.remove_listener("response", on_worker_response)
        worker_page.remove_listener("framenavigated", on_frame_navigated)
        worker_page.remove_listener("request", on_worker_request)


async def wait_for_table_or_captcha(context, blank_keepalive_page, cfg):
    announced_captcha = False

    for _ in range(600):
        pages = list(context.pages)

        if not pages or (len(pages) == 1 and pages[0] == blank_keepalive_page):
            log("Filecrypt tab was closed by user.", "WARN", cfg)
            return None

        for p in pages:
            if p == blank_keepalive_page or p.is_closed():
                continue

            url = p.url.lower()

            if url and not url.startswith(("about:", "chrome:")):
                domain = urlparse(url).netloc
                if domain and not any(
                    allowed in domain
                    for allowed in ["filecrypt.", "cutcaptcha.", "cloudflare."]
                ):
                    log(f"Closed hijacked Ad Tab: {p.url}", "DEBUG", cfg)
                    await safe_close_page(p)
                    continue

            try:
                table_count = await p.locator(
                    "table tbody tr td.status, table tbody tr td[title]"
                ).count()
                if table_count > 0:
                    if not announced_captcha:
                        log(
                            "Active Session Cookie Detected! Captcha skipped & table loaded directly.",
                            "SUCCESS",
                            cfg,
                        )
                    else:
                        log(
                            "Captcha solved! Download links table loaded.",
                            "SUCCESS",
                            cfg,
                        )
                    await asyncio.sleep(0.8)
                    return p

                if not announced_captcha:
                    captcha_count = await p.locator(
                        "form#cform, .protection, h2:has-text('Security Check')"
                    ).count()
                    if captcha_count > 0:
                        set_window_title("Waiting for Captcha in Browser...")
                        log(
                            "Security Check detected! Please solve the Captcha in Browser...",
                            "INFO",
                            cfg,
                        )
                        announced_captcha = True
            except Exception:
                pass

        await asyncio.sleep(0.5)

    return None


def save_host_urls(host_name, container_title, extracted_urls, cfg):
    if not extracted_urls:
        return None

    save_dir = os.path.abspath(
        os.path.join(
            APP_DIR,
            cfg.get("Output", "save_directory", fallback="./decrypted_links"),
        )
    )
    copy_clip = cfg.getboolean("Output", "copy_to_clipboard", fallback=False)
    use_title = cfg.getboolean(
        "Output", "include_container_title_in_filename", fallback=True
    )

    os.makedirs(save_dir, exist_ok=True)

    clean_host = re.sub(r"[^a-zA-Z0-9]+", "_", host_name).strip("_")
    if use_title and container_title:
        clean_title = re.sub(r"[^a-zA-Z0-9._-]+", "_", container_title).strip(
            "_"
        )[:60]
        filename = f"{clean_title}_{clean_host}.txt"
    else:
        filename = f"decrypted_{clean_host}.txt"

    output_path = os.path.join(save_dir, filename)

    raw_urls_text = "\n".join(url for _, url in extracted_urls)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(raw_urls_text + "\n")

    clip_msg = ""
    if copy_clip and copy_text_to_clipboard(raw_urls_text):
        clip_msg = f" [{ACCENT_MAUVE}](+ Copied to Clipboard)[/{ACCENT_MAUVE}]"

    safe_host = escape(host_name)
    safe_path = escape(output_path)
    return (
        f"[bold {ACCENT_GREEN}]✓ Saved {len(extracted_urls)} URLs for \\[{safe_host}][/bold {ACCENT_GREEN}] "
        f"[{TEXT_MUTED}]──›[/{TEXT_MUTED}] [{ACCENT_TEAL}]{safe_path}[/{ACCENT_TEAL}]{clip_msg}"
    )


async def launch_configured_browser(playwright_instance, cfg):
    browser_choice = (
        cfg.get("General", "browser", fallback="chrome").strip().lower()
    )
    exe_path = cfg.get("General", "executable_path", fallback="").strip()
    rel_profile = cfg.get(
        "Session", "profile_directory", fallback="./chrome_filecrypt_profile"
    ).strip()
    profile_dir = os.path.abspath(os.path.join(APP_DIR, rel_profile))

    common_args = [
        "--disable-blink-features=AutomationControlled",
        "--start-maximized",
    ]

    if browser_choice == "firefox":
        log("Launching Firefox...", "INFO", cfg)
        return await playwright_instance.firefox.launch_persistent_context(
            user_data_dir=profile_dir + "_firefox",
            executable_path=exe_path if exe_path else None,
            headless=False,
            no_viewport=True,
        )

    if browser_choice == "brave" and not exe_path:
        possible_brave_paths = [
            os.path.expandvars(
                r"%ProgramFiles%\BraveSoftware\Brave-Browser\Application\brave.exe"
            ),
            os.path.expandvars(
                r"%ProgramFiles(x86)%\BraveSoftware\Brave-Browser\Application\brave.exe"
            ),
            os.path.expandvars(
                r"%LocalAppData%\BraveSoftware\Brave-Browser\Application\brave.exe"
            ),
        ]
        for bp in possible_brave_paths:
            if os.path.exists(bp):
                exe_path = bp
                break

    channel = None
    if not exe_path and browser_choice in ("chrome", "msedge"):
        channel = browser_choice

    log(
        f"Launching Browser ({exe_path or channel or 'chromium'})...",
        "INFO",
        cfg,
    )
    return await playwright_instance.chromium.launch_persistent_context(
        user_data_dir=profile_dir,
        channel=channel,
        executable_path=exe_path if exe_path else None,
        headless=False,
        ignore_default_args=["--no-sandbox"],
        args=common_args,
        no_viewport=True,
    )


async def ensure_browser_session(playwright_instance, session_holder, cfg):
    desired_browser = (
        cfg.get("General", "browser", fallback="chrome").strip().lower()
    )
    keep_sess = cfg.getboolean(
        "Session", "keep_session_across_containers", fallback=True
    )

    if (
        is_context_alive(session_holder["context"])
        and session_holder["browser_name"] != desired_browser
    ):
        await safe_close_context(session_holder["context"])
        session_holder["context"] = None

    if not is_context_alive(session_holder["context"]):
        ctx = await launch_configured_browser(playwright_instance, cfg)
        await ctx.add_init_script(ANTI_AD_INIT_SCRIPT)

        keepalive = ctx.pages[0] if ctx.pages else await ctx.new_page()
        await keepalive.goto("about:blank")
        await keepalive.evaluate(
            "document.title = 'Keep-Alive (Do Not Close)'; document.documentElement.style.backgroundColor = '#0d0f14';"
        )

        async def global_ad_router(route):
            try:
                req = route.request
                url = req.url
                url_lower = url.lower()

                ad_keywords = [
                    "popads",
                    "propeller",
                    "adsterra",
                    "exoclick",
                    "doubleclick",
                    "syndication",
                    "ddownload.com/pricing",
                    "werbemanager",
                    "0pixel.php",
                    "this-is-not-tracking.php",
                    "bla3.php",
                    "meh.php",
                    "muh2.php",
                ]
                if any(k in url_lower for k in ad_keywords):
                    await route.abort()
                    return

                if session_holder["phase"] == "LOAD_CONTAINER":
                    if "/link/" in url_lower or "/go/" in url_lower:
                        await route.abort()
                        return

                    if req.is_navigation_request():
                        domain = urlparse(url_lower).netloc
                        if domain and not any(
                            ok in domain
                            for ok in [
                                "filecrypt.",
                                "cutcaptcha.",
                                "cloudflare.",
                            ]
                        ):
                            await route.abort()
                            return

                await route.continue_()
            except Exception:
                pass

        await ctx.route("**/*", global_ad_router)

        session_holder["context"] = ctx
        session_holder["keepalive_page"] = keepalive
        session_holder["browser_name"] = desired_browser
    else:
        if not keep_sess:
            log(
                "Resetting session cookies for new container per Settings...",
                "INFO",
                cfg,
            )
            await session_holder["context"].clear_cookies()
        else:
            log(
                "Re-using active browser session (solved Captchas stay remembered!)...",
                "INFO",
                cfg,
            )

    return session_holder["context"], session_holder["keepalive_page"]


async def process_single_container(
    playwright_instance, container_url, cfg, session_holder
):
    print_app_header(
        "App / Container / Loading & Security Check",
        cfg,
        clear=not is_debug(cfg),
        window_subtitle="Loading Container...",
    )

    try:
        try:
            context, blank_keepalive_page = await ensure_browser_session(
                playwright_instance, session_holder, cfg
            )
        except Exception as launch_err:
            log(f"Could not launch browser: {launch_err}", "ERROR", cfg)
            log(
                "Tip: Make sure another instance of the browser profile isn't already open, or press 'S' to switch browsers.",
                "WARN",
                cfg,
            )
            console.input(
                f"\n  [{ACCENT_AMBER}]Press Enter to return to main menu...[/{ACCENT_AMBER}]"
            )
            return "NEW_CONTAINER"

        for p_old in list(context.pages):
            if p_old != blank_keepalive_page:
                await safe_close_page(p_old)

        session_holder["phase"] = "LOAD_CONTAINER"
        filecrypt_page = await context.new_page()

        log(f"Opening Container: {container_url}", "INFO", cfg)
        try:
            await filecrypt_page.goto(
                container_url, wait_until="domcontentloaded"
            )
        except Exception as e:
            log(f"Initial navigation note: {e}", "DEBUG", cfg)

        active_container_page = await wait_for_table_or_captcha(
            context, blank_keepalive_page, cfg
        )

        if not active_container_page:
            await safe_close_context(context)
            session_holder["context"] = None
            return "NEW_CONTAINER"

        closing_for_new_container = {"flag": False}

        def on_container_tab_closed(_):
            if not closing_for_new_container["flag"]:
                asyncio.create_task(safe_close_context(context))
                session_holder["context"] = None

        active_container_page.on("close", on_container_tab_closed)

        async def lock_main_tab_with_204(route):
            try:
                req = route.request
                if req.is_navigation_request():
                    await route.fulfill(status=204)
                    return
                await route.continue_()
            except Exception:
                pass

        await active_container_page.route("**/*", lock_main_tab_with_204)

        for p_other in list(context.pages):
            if p_other not in (active_container_page, blank_keepalive_page):
                await safe_close_page(p_other)

        info = await freeze_page_and_extract_container_info(
            active_container_page, cfg
        )
        container_title = info.get("title", "container")
        rows = info.get("rows", [])

        if not rows:
            log(
                "No individual download rows found! (This container may only offer DLC / Click'n'Load).",
                "ERROR",
                cfg,
            )
            console.input(
                f"\n  [{ACCENT_AMBER}]Press Enter to return to main menu...[/{ACCENT_AMBER}]"
            )
            closing_for_new_container["flag"] = True
            await safe_close_page(active_container_page)
            return "NEW_CONTAINER"

        hosts_map = {}
        for r in rows:
            h = r["host"]
            hosts_map.setdefault(h, []).append(r)

        host_list = list(hosts_map.keys())
        session_holder["phase"] = "HARVEST_AND_RESOLVE"

        decrypted_status = {}
        harvested_cache = {}
        worker_holder = {"page": None}
        last_action_banners = []
        last_decrypted_batches = []

        while not active_container_page.is_closed():
            print_app_header(
                "App / Container / Host Selection",
                cfg,
                clear=not is_debug(cfg),
                window_subtitle=container_title,
            )

            render_container_info_panel(
                container_title,
                total_links=len(rows),
                total_hosts=len(hosts_map),
            )

            if last_decrypted_batches:
                batch_blocks = []
                for b_host, b_items in last_decrypted_batches:
                    lines = [
                        f"[bold {ACCENT_LAVENDER}]Host: {escape(b_host)} ({len(b_items)} links)[/bold {ACCENT_LAVENDER}]"
                    ]
                    tot_b = len(b_items)
                    for idx_b, (b_fname, b_url, b_ok) in enumerate(b_items, 1):
                        lines.append(
                            format_resolved_item_markup(
                                idx_b, tot_b, b_fname, b_url, success=b_ok
                            )
                        )
                    batch_blocks.append("\n".join(lines))

                console.print(
                    Panel(
                        "\n\n".join(batch_blocks),
                        title=f"[bold {ACCENT_TEAL}]Extracted Links (Press [C] to copy all URLs to clipboard)[/bold {ACCENT_TEAL}]",
                        title_align="left",
                        box=box.ROUNDED,
                        border_style=BORDER_ACCENT,
                        padding=(0, 1),
                        width=UI_WIDTH,
                    )
                )

            if last_action_banners:
                console.print(
                    Panel(
                        "\n".join(f"  {b}" for b in last_action_banners),
                        box=box.ROUNDED,
                        border_style=ACCENT_GREEN,
                        padding=(0, 1),
                        width=UI_WIDTH,
                    )
                )

            host_table = Table(
                box=box.SIMPLE,
                border_style=BORDER_SUBTLE,
                expand=True,
                show_header=True,
                header_style=f"bold {TEXT_MUTED}",
                padding=(0, 1),
            )
            host_table.add_column(
                "Key", width=6, justify="center", no_wrap=True
            )
            host_table.add_column("Host Provider", width=22, no_wrap=True)
            host_table.add_column("Total Links", width=13, no_wrap=True)
            host_table.add_column("Availability", ratio=1, no_wrap=True)
            host_table.add_column(
                "Decryption Status", width=22, justify="right", no_wrap=True
            )

            for idx, h in enumerate(host_list, 1):
                h_rows = hosts_map[h]
                total_h = len(h_rows)
                online_h = sum(1 for r in h_rows if r.get("is_online", True))
                offline_h = total_h - online_h

                if offline_h == 0:
                    health_str = (
                        f"[{ACCENT_GREEN}]● {online_h} Online[/{ACCENT_GREEN}]"
                    )
                elif online_h == 0:
                    health_str = (
                        f"[{ACCENT_CORAL}]○ {offline_h} Offline[/{ACCENT_CORAL}]"
                    )
                else:
                    health_str = f"[{ACCENT_GREEN}]● {online_h} Online[/{ACCENT_GREEN}] [{TEXT_MUTED}]/[/{TEXT_MUTED}] [{ACCENT_CORAL}]{offline_h} Offline[/{ACCENT_CORAL}]"

                status_badge = f"[{TEXT_MUTED}]Pending[/{TEXT_MUTED}]"
                if h in decrypted_status:
                    done_cnt, tot_cnt = decrypted_status[h]
                    status_badge = f"[bold {ACCENT_GREEN}]✓ Decrypted ({done_cnt}/{tot_cnt})[/bold {ACCENT_GREEN}]"

                host_style = f"bold {TEXT_MAIN}" if online_h > 0 else TEXT_MUTED
                host_table.add_row(
                    f"[bold {ACCENT_TEAL}][{idx}][/bold {ACCENT_TEAL}]",
                    f"[{host_style}]{escape(h)}[/{host_style}]",
                    f"[{TEXT_MAIN}]{total_h} links[/{TEXT_MAIN}]",
                    health_str,
                    status_badge,
                )

            all_done = len(decrypted_status) == len(host_list)
            host_table.add_section()
            host_table.add_row(
                f"[bold {ACCENT_GREEN}][A][/bold {ACCENT_GREEN}]",
                f"[bold {TEXT_MAIN}]Decrypt All Hosts[/bold {TEXT_MAIN}]",
                f"[{TEXT_MUTED}]{len(rows)} links[/{TEXT_MUTED}]",
                "",
                f"[bold {ACCENT_GREEN}]✓ All Complete[/bold {ACCENT_GREEN}]"
                if all_done
                else "",
            )
            if last_decrypted_batches:
                host_table.add_row(
                    f"[bold {ACCENT_LAVENDER}][C][/bold {ACCENT_LAVENDER}]",
                    f"[{TEXT_MAIN}]Copy Shown URLs[/{TEXT_MAIN}]",
                    "",
                    f"[{TEXT_MUTED}]Copy extracted links to Clipboard[/{TEXT_MUTED}]",
                    "",
                )
            host_table.add_row(
                f"[bold {ACCENT_TEAL}][N][/bold {ACCENT_TEAL}]",
                f"[{TEXT_MAIN}]New Container URL[/{TEXT_MAIN}]",
                "",
                f"[{TEXT_MUTED}]Keeps session alive[/{TEXT_MUTED}]",
                "",
            )
            host_table.add_row(
                f"[bold {ACCENT_AMBER}][O][/bold {ACCENT_AMBER}]",
                f"[{TEXT_MAIN}]Open Output Folder[/{TEXT_MAIN}]",
                "",
                f"[{TEXT_MUTED}]View saved .txt files[/{TEXT_MUTED}]",
                "",
            )
            host_table.add_row(
                f"[bold {ACCENT_MAUVE}][S][/bold {ACCENT_MAUVE}]",
                f"[{TEXT_MAIN}]Application Settings[/{TEXT_MAIN}]",
                "",
                f"[{TEXT_MUTED}]Browser, Debug, Cookies[/{TEXT_MUTED}]",
                "",
            )
            host_table.add_row(
                f"[bold {ACCENT_CORAL}][Q][/bold {ACCENT_CORAL}]",
                f"[{TEXT_MAIN}]Quit Application[/{TEXT_MAIN}]",
                "",
                "",
                "",
            )

            console.print(
                Panel(
                    host_table,
                    box=box.ROUNDED,
                    border_style=BORDER_SUBTLE,
                    padding=(0, 1),
                    width=UI_WIDTH,
                )
            )

            valid_keys = ["A", "N", "O", "S", "Q"]
            prompt_keys = f"1-{len(host_list)}, A, N, O, S, Q"
            if last_decrypted_batches:
                valid_keys.append("C")
                prompt_keys = f"1-{len(host_list)}, A, C, N, O, S, Q"

            choice = await wait_for_single_key(
                prompt_markup=f"\n  [{ACCENT_TEAL}]Press key [{prompt_keys}]: [/{ACCENT_TEAL}]",
                valid_letters=valid_keys,
                max_num=len(host_list),
                abort_check_fn=lambda: active_container_page.is_closed(),
                allow_esc=False,
            )

            if choice is None or active_container_page.is_closed():
                await safe_close_context(context)
                session_holder["context"] = None
                return "NEW_CONTAINER"

            if choice == "Q":
                await safe_close_context(context)
                session_holder["context"] = None
                return "QUIT_APP"
            elif choice == "N":
                closing_for_new_container["flag"] = True
                await safe_close_page(worker_holder["page"])
                await safe_close_page(active_container_page)
                return "NEW_CONTAINER"
            elif choice == "C" and last_decrypted_batches:
                urls_to_copy = [
                    u
                    for _, items in last_decrypted_batches
                    for _, u, ok in items
                    if ok
                ]
                if urls_to_copy and copy_text_to_clipboard(
                    "\n".join(urls_to_copy)
                ):
                    last_action_banners = [
                        f"[bold {ACCENT_GREEN}]✓ Copied {len(urls_to_copy)} decrypted URLs to Windows Clipboard![/bold {ACCENT_GREEN}]"
                    ]
                else:
                    last_action_banners = [
                        f"[{ACCENT_CORAL}]! Could not copy URLs to clipboard.[/{ACCENT_CORAL}]"
                    ]
                continue
            elif choice == "O":
                save_dir = os.path.abspath(
                    os.path.join(
                        APP_DIR,
                        cfg.get(
                            "Output",
                            "save_directory",
                            fallback="./decrypted_links",
                        ),
                    )
                )
                open_folder_in_explorer(save_dir)
                last_action_banners = [
                    f"[{ACCENT_GREEN}]✓ Opened folder in Explorer:[/{ACCENT_GREEN}] [{ACCENT_TEAL}]{escape(save_dir)}[/{ACCENT_TEAL}]"
                ]
                continue
            elif choice == "S":
                await interactive_settings_menu(
                    cfg, parent_path="App / Container"
                )
                continue
            elif choice == "A":
                hosts_to_process = host_list
            elif choice.isdigit() and 1 <= int(choice) <= len(host_list):
                hosts_to_process = [host_list[int(choice) - 1]]
            else:
                last_action_banners = [
                    f"[{ACCENT_AMBER}]! Invalid key '{escape(choice)}'. Please try again.[/{ACCENT_AMBER}]"
                ]
                continue

            skip_offline = cfg.getboolean(
                "Behavior", "skip_offline_links", fallback=False
            )
            req_delay = cfg.getfloat("Behavior", "request_delay", fallback=0.2)
            last_action_banners = []
            last_decrypted_batches = []

            for target_host in hosts_to_process:
                if active_container_page.is_closed():
                    break

                selected_rows = hosts_map[target_host]
                if skip_offline:
                    selected_rows = [
                        r for r in selected_rows if r.get("is_online", True)
                    ]
                    if not selected_rows:
                        last_action_banners.append(
                            f"[{ACCENT_AMBER}]! Skipped \\[{escape(target_host)}] (all links marked Offline).[/{ACCENT_AMBER}]"
                        )
                        continue

                extracted_for_host = []
                display_items_for_host = []

                print_app_header(
                    f"App / Container / Decrypting ({target_host})",
                    cfg,
                    clear=not is_debug(cfg),
                    window_subtitle=f"Decrypting {target_host} ({len(selected_rows)} links)...",
                )
                render_container_info_panel(
                    container_title,
                    host_name=target_host,
                    host_links=len(selected_rows),
                )

                harvested_links = []
                if not is_debug(cfg):
                    with Progress(
                        SpinnerColumn(style=ACCENT_TEAL),
                        TextColumn(
                            f"  [{TEXT_MAIN}]Step 1/2 (Preparing Links)[/{TEXT_MAIN}]"
                        ),
                        BarColumn(
                            bar_width=38,
                            style=BORDER_SUBTLE,
                            complete_style=ACCENT_TEAL,
                            finished_style=ACCENT_GREEN,
                        ),
                        TaskProgressColumn(),
                        console=console,
                    ) as progress:
                        task_id = progress.add_task(
                            "harvest", total=len(selected_rows)
                        )
                        for row in selected_rows:
                            if active_container_page.is_closed():
                                break
                            row_idx = row["index"]
                            if row_idx in harvested_cache:
                                link_url = harvested_cache[row_idx]
                            else:
                                link_url = await harvest_intermediate_link_url(
                                    context,
                                    active_container_page,
                                    blank_keepalive_page,
                                    row,
                                    cfg,
                                )
                                if link_url:
                                    harvested_cache[row_idx] = link_url
                                else:
                                    await asyncio.sleep(0.2)
                            if link_url:
                                harvested_links.append(
                                    (row["filename"], row["size"], link_url)
                                )
                            progress.advance(task_id)
                else:
                    for row in selected_rows:
                        if active_container_page.is_closed():
                            break
                        row_idx = row["index"]
                        link_url = harvested_cache.get(
                            row_idx
                        ) or await harvest_intermediate_link_url(
                            context,
                            active_container_page,
                            blank_keepalive_page,
                            row,
                            cfg,
                        )
                        if link_url:
                            harvested_cache[row_idx] = link_url
                            harvested_links.append(
                                (row["filename"], row["size"], link_url)
                            )

                if active_container_page.is_closed():
                    break

                console.print()
                for i, (fname, fsize, link_url) in enumerate(
                    harvested_links, 1
                ):
                    if active_container_page.is_closed():
                        break

                    set_window_title(
                        f"[{i:02d}/{len(harvested_links):02d}] Decrypting {target_host} — {container_title}"
                    )
                    final_url = await resolve_link_smart(
                        context,
                        active_container_page,
                        worker_holder,
                        link_url,
                        fname,
                        i,
                        len(harvested_links),
                        cfg,
                    )
                    if final_url:
                        extracted_for_host.append((fname, final_url))
                        display_items_for_host.append((fname, final_url, True))
                    else:
                        display_items_for_host.append(
                            (
                                fname,
                                "[TIMEOUT - Failed to resolve within 20s]",
                                False,
                            )
                        )
                    await asyncio.sleep(req_delay)

                if display_items_for_host:
                    last_decrypted_batches.append(
                        (target_host, display_items_for_host)
                    )

                if extracted_for_host:
                    if len(extracted_for_host) == len(hosts_map[target_host]):
                        for r in hosts_map[target_host]:
                            r["is_online"] = True

                    banner_msg = save_host_urls(
                        target_host, container_title, extracted_for_host, cfg
                    )
                    if banner_msg:
                        last_action_banners.append(banner_msg)
                    decrypted_status[target_host] = (
                        len(extracted_for_host),
                        len(selected_rows),
                    )
                else:
                    last_action_banners.append(
                        f"[{ACCENT_CORAL}]! No final URLs were extracted for \\[{escape(target_host)}].[/{ACCENT_CORAL}]"
                    )

        return "NEW_CONTAINER"

    except (PlaywrightError, asyncio.CancelledError):
        await safe_close_context(session_holder["context"])
        session_holder["context"] = None
        return "NEW_CONTAINER"


async def main_app_loop():
    cfg = validate_and_load_config()
    home_warning = None

    session_holder = {
        "context": None,
        "keepalive_page": None,
        "browser_name": None,
        "phase": "LOAD_CONTAINER",
    }

    async with async_playwright() as p:
        try:
            while True:
                cfg = validate_and_load_config()
                print_app_header(
                    "App / Home",
                    cfg,
                    clear=not is_debug(cfg),
                    window_subtitle="Home",
                )

                if home_warning:
                    console.print(
                        Panel(
                            f"  {home_warning}",
                            box=box.ROUNDED,
                            border_style=ACCENT_AMBER,
                            padding=(0, 1),
                            width=UI_WIDTH,
                        )
                    )
                    home_warning = None

                render_settings_preview_panel(cfg)

                if is_context_alive(session_holder["context"]):
                    sess_state = f"[bold {ACCENT_GREEN}]● Active[/bold {ACCENT_GREEN}] [{TEXT_MUTED}](Browser Open — Solved Captchas Remembered)[/{TEXT_MUTED}]"
                else:
                    sess_state = f"[{TEXT_MUTED}]○ Standby (Launches when you enter a Container URL)[/{TEXT_MUTED}]"

                home_body = (
                    f"[{TEXT_MUTED}]Browser Session:[/{TEXT_MUTED}]  {sess_state}\n\n"
                    f"  • Paste a [bold {ACCENT_TEAL}]Filecrypt Container URL[/bold {ACCENT_TEAL}] below and press [bold {ACCENT_GREEN}]Enter[/bold {ACCENT_GREEN}]\n"
                    f"  • Quick Commands:  [bold {ACCENT_MAUVE}][S][/bold {ACCENT_MAUVE}] Settings    "
                    f"[bold {ACCENT_AMBER}][O][/bold {ACCENT_AMBER}] Open Saved Folder    "
                    f"[bold {ACCENT_LAVENDER}][I][/bold {ACCENT_LAVENDER}] About / License    "
                    f"[bold {ACCENT_CORAL}][Q][/bold {ACCENT_CORAL}] Quit App"
                )
                console.print(
                    Panel(
                        home_body,
                        box=box.ROUNDED,
                        border_style=BORDER_SUBTLE,
                        padding=(1, 2),
                        width=UI_WIDTH,
                    )
                )

                user_inp = await read_text_with_esc(
                    f"\n  [{ACCENT_TEAL}]Container URL or Command [S / O / I / Q]: [/{ACCENT_TEAL}]"
                )

                if not user_inp:
                    continue

                cmd = user_inp.upper()
                if cmd == "Q":
                    break
                elif cmd == "S":
                    await interactive_settings_menu(cfg, parent_path="App")
                    continue
                elif cmd == "I":
                    await show_about_license_screen(cfg)
                    continue
                elif cmd == "O":
                    save_dir = os.path.abspath(
                        os.path.join(
                            APP_DIR,
                            cfg.get(
                                "Output",
                                "save_directory",
                                fallback="./decrypted_links",
                            ),
                        )
                    )
                    open_folder_in_explorer(save_dir)
                    continue

                if not user_inp.startswith(("http://", "https://")):
                    user_inp = "https://" + user_inp

                parsed_inp = urlparse(user_inp)
                if "." not in parsed_inp.netloc or len(parsed_inp.netloc) < 4:
                    home_warning = f"[{ACCENT_AMBER}]! '{escape(user_inp)}' is not a valid Container URL or command (S, O, I, Q).[/{ACCENT_AMBER}]"
                    continue

                action = await process_single_container(
                    p, user_inp, cfg, session_holder
                )

                if action == "QUIT_APP":
                    break
        finally:
            await safe_close_context(session_holder["context"])


if __name__ == "__main__":
    multiprocessing.freeze_support()
    try:
        asyncio.run(main_app_loop())
    except KeyboardInterrupt:
        console.print(f"\n  [{ACCENT_AMBER}]Exited cleanly.[/{ACCENT_AMBER}]")
        sys.exit(0)
    except Exception as fatal_err:
        console.print(
            f"\n  [bold {ACCENT_CORAL}][FATAL ERROR] {escape(str(fatal_err))}[/bold {ACCENT_CORAL}]"
        )
        input("\n  Press Enter to exit...")
