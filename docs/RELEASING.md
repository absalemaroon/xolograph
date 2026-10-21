# Releasing Xolograph

Everything below is **optional and secret-driven**: with no secrets set, a tagged release still builds
unsigned installers. Add secrets in GitHub → Settings → Secrets and variables → Actions.

## 1. Cut a release
1. Bump the version in `package.json`, `frontend/package.json`, `src-tauri/Cargo.toml`, `src-tauri/tauri.conf.json`.
2. Tag and push: `git tag v1.3.0 && git push origin v1.3.0`.
3. `Release` workflow builds Windows, macOS (arm64 + Intel) and Linux and creates a **draft** GitHub release.
4. Review the draft, then **publish** it (auto-update reads `releases/latest/download/latest.json`, which only
   exists for a *published* release).

## 2. Auto-update (signed update packages)
Update packages are verified with a minisign-style key pair created by Tauri. **Generate the key on a machine you
trust** (a PC, or a private GitHub Codespace — *never* in a public workflow, its logs/artifacts are readable):
```
npx @tauri-apps/cli signer generate -w xolograph-updater.key
```
Add these repository secrets:
| Secret | Value |
|---|---|
| `TAURI_SIGNING_PRIVATE_KEY` | contents of `xolograph-updater.key` |
| `TAURI_SIGNING_PRIVATE_KEY_PASSWORD` | the password you chose (may be empty) |
| `TAURI_UPDATER_PUBKEY` | contents of `xolograph-updater.key.pub` |

The workflow then builds with `--features updater`, signs the packages and uploads `latest.json`. Keep a backup
of the private key: **if it is lost, existing installs cannot be updated**. Updates are checked only when the user
presses **Check for updates** in Settings (no background network calls).

## 3. Windows code signing
Obtain an Authenticode code-signing certificate (OV/EV, from a CA). Export it as `.pfx`, then:
```
base64 -w0 certificate.pfx        # on macOS: base64 -i certificate.pfx
```
Secrets: `WINDOWS_CERTIFICATE` (that base64 string), `WINDOWS_CERTIFICATE_PASSWORD`.
Without it, Windows SmartScreen shows an "unknown publisher" warning.

## 4. macOS signing and notarization
Requires an Apple Developer account (Developer ID Application certificate). Export the certificate as `.p12`, base64 it,
and create an app-specific password at appleid.apple.com. Secrets:
`APPLE_CERTIFICATE`, `APPLE_CERTIFICATE_PASSWORD`, `APPLE_SIGNING_IDENTITY` (e.g. `Developer ID Application: Name (TEAMID)`),
`APPLE_ID`, `APPLE_PASSWORD` (app-specific), `APPLE_TEAM_ID`.

## 5. Verifying a download
Compare the file's SHA-256 (`sha256sum file`) with the value published alongside the release.
