<div align="center">
  <img src="icon.png" alt="Filecrypt Container Decryptor Icon" width="110" />
  <h1>Filecrypt Container Decryptor</h1>
  <p>A fast, zero-flash Filecrypt container decryptor and link extractor with a modern Rich TUI, ad/popup blocking, session persistence, and multi-host batch extraction.</p>
  <p>
    <a href="https://github.com/ArvindSaini978/Filecrypt-Decryptor/releases/latest"><b>⬇ Download Standalone .EXE (v1.0)</b></a>
    &nbsp;•&nbsp;
    <a href="https://github.com/ArvindSaini978/Filecrypt-Decryptor/blob/master/filecrypt_decryptor.py"><b>View Python Source</b></a>
    &nbsp;•&nbsp;
    <a href="https://github.com/ArvindSaini978/Filecrypt-Decryptor/blob/master/LICENSE"><b>GPL-3.0 License</b></a>
    &nbsp;•&nbsp;
    <a href="https://github.com/ArvindSaini978/Filecrypt-Decryptor/issues"><b>Report Issue</b></a>
  </p>
</div>

---

## ✨ Key Features

- **⚡ Zero-Flash In-Memory Link Resolver:** Follows Filecrypt `/Link/` ➔ `/Go/` redirect chains directly in memory with **zero screen flashing**, with an automatic single-tab dark fallback for client-side hash hosts like `mega.nz`.
- **🛡️ Multi-Layer Ad & Popup Blocker:** Automatically strips Filecrypt/CutCaptcha ad overlays, shadow-DOM click-jackers, and hijacked popup tabs.
- **🔑 Captcha Session Persistence (`KEEP ALIVE`):** Keeps the browser session alive across multiple containers (`[N]`) so solved Captchas (`PHPSESSID`) stay remembered while the browser remains open.
- **📊 Smart Host Selection & Availability Detection:** Groups container links by host provider (`mega.nz`, `pixeldrain.com`, `1fichier.com`, etc.), detects online/offline status, heals false-offline status glitches, and saves clean `.txt` link files.
- **📋 Persistent Link Viewer & Quick Copy:** Displays extracted links right inside the terminal after decryption with one-key `[C]` clipboard copy.
- **🎨 Soft Nord / Catppuccin Mocha TUI:** Easy-on-the-eyes terminal interface built with `Rich`, featuring an interactive Settings menu (`[S]`) and self-healing `config.ini` validator.

---

## 🚀 Quick Start

### Option 1: Standalone Windows Executable (No Python Required — Recommended)
1. Go to the [**Releases Page**](https://github.com/ArvindSaini978/Filecrypt-Decryptor/releases/latest) and download **`FilecryptDecryptor.exe`**.
2. Double-click **`FilecryptDecryptor.exe`** to launch the app.
3. Paste any Filecrypt Container URL and press **Enter**!
> **Note:** Requires an installed web browser (**Google Chrome**, **Microsoft Edge**, or **Brave Browser**). You can switch browsers anytime inside the app by pressing **`[S]` (Settings)**.

### Option 2: Run from Python Source Code
```bash
git clone https://github.com/ArvindSaini978/Filecrypt-Decryptor.git
cd Filecrypt-Decryptor
pip install -r requirements.txt
python filecrypt_decryptor.py
```

---

## ⌨️ In-App Controls & Shortcuts

| Key | Action | Description |
| :---: | :--- | :--- |
| **`[1-9]`** | **Decrypt Selected Host** | Extracts and saves all download links for a specific host provider |
| **`[A]`** | **Decrypt All Hosts** | Batch-extracts links across every available host in the container |
| **`[C]`** | **Copy Shown URLs** | Copies the most recently decrypted URLs directly to Windows Clipboard |
| **`[N]`** | **New Container URL** | Loads another Filecrypt container while keeping your solved Captcha session alive |
| **`[O]`** | **Open Output Folder** | Opens `./decrypted_links` in Windows Explorer to view saved `.txt` files |
| **`[S]`** | **Application Settings** | Customize Browser, Resolve Mode, Auto-Copy, Session Persistence, and Debug Mode |
| **`[I]`** | **About & License** | View author info, GitHub repository link, and GPL-3.0 license details |
| **`[Q]`** | **Quit Application** | Cleanly closes the browser session and exits |

---

## 🛠️ Building the Standalone `.exe` Yourself

If you want to compile `FilecryptDecryptor.exe` from the source code yourself:

```powershell
pip install -r requirements.txt
pip install pyinstaller pillow

pyinstaller --noconfirm --clean --onefile --console --name "FilecryptDecryptor" --icon "icon.png" --version-file version_info.txt --collect-all playwright --collect-submodules rich --exclude-module tkinter --exclude-module unittest --exclude-module pydoc --exclude-module sqlite3 --exclude-module test --exclude-module xmlrpc filecrypt_decryptor.py
```

---

## 📜 License

Copyright © 2026 **[ArvindSaini978](https://github.com/ArvindSaini978)**.  
This program is free software licensed under the **[GNU General Public License v3.0 (GPL-3.0)](https://github.com/ArvindSaini978/Filecrypt-Decryptor/blob/master/LICENSE)**.
