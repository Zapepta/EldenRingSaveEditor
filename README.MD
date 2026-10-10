[English] | [Русский](README_RU.md)

# Elden Ring Save Editor (CLI)

A lightweight (~2.5 MB), standalone command-line tool written in pure Go for editing PC Elden Ring save files (`ER0000.sl2`) and Seamless Co-op saves (`ER0000.co2`).

**No installation required. No dependencies. Just download the `.exe` and run.**

---

## Quick Start (Download)

1. Go to the **[Releases](../../releases)** tab (or look at the right sidebar).
2. Download the latest `EldenRingSaveEditor.exe`.
3. Run the executable.

---

## Features

- **Automatic Save Discovery:** Automatically detects active SteamID profiles in `%APPDATA%\EldenRing` and sorts saves by last modified date.
- **Seamless Co-op Ready:** Native support for both Vanilla (`.sl2`) and Seamless Co-op (`.co2`) save files.
- **DLC Safe (Shadow of the Erdtree):** Dynamic version detection handles offset shifts between pre-DLC and Patch 1.12+ saves automatically (`0x13FE` vs `0x1400` item pool).
- **Checksum Verification:** Automatically recalculates slot MD5 checksums on save — completely eliminates "Save Data Corrupted" warnings.
- **Weapon Swapping & Upgrading:** Replace any weapon using the built-in search with automatic Ash of War / infusion sanitization to prevent inventory crashes.
- **Stat & Level Validation:** Edit attributes safely with built-in class base floor checks to prevent impossible stat flags.
- **Automatic Backups:** Creates an instant `.bak` backup copy of your save before any changes are written.

---

## How It Works

1. Launch `EldenRingSaveEditor.exe`.
2. Select your save file from the detected list (or pass path as an argument).
3. Select your character slot.
4. Modify stats, runes, starting class, or weapons.
5. Choose **Save to Disk (Recalculate MD5)**.
6. Launch Elden Ring and enjoy.

---

## Disclaimer

- The tool creates a `.bak` backup automatically, but it is always recommended to keep your own copy of your save files.
- Modifying save files carries inherent risks on official servers. Recommended for offline play or Seamless Co-op.