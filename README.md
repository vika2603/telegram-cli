# tg

A command-line Telegram client. Script-friendly, JSON-first.

## Install

```sh
go install github.com/vika2603/telegram-cli/cmd/tg@latest
```

## Quick start

Get `api_id` and `api_hash` at <https://my.telegram.org>.

```sh
tg login work --api-id <ID> --api-hash <HASH>
# paste the login code from your Telegram app

tg auth switch work
tg auth list
```

Pass `--qr` to `auth login` for QR login instead.

## Command Map

### Frequent

| Command | Purpose |
| --- | --- |
| `tg login <name>` | Shortcut for `tg auth login <name>`. |
| `tg logout [name]` | Shortcut for `tg auth logout [name]`. |
| `tg send <ref> [text...]` | Shortcut for `tg msg send <ref> [text...]`. Repeat `--file` to attach media. |
| `tg reply <msg-ref> [text...]` | Reply to one message. Repeat `--file` to attach media. |
| `tg inbox` | Show recent dialogs with unread counts and last messages. |
| `tg read <ref>` | Shortcut for `tg msg list <ref>`. |
| `tg digest <ref>` | Compact history view for one chat. |
| `tg resolve <ref>` | Resolve one user, chat, bot, or channel. |

### Account, Session, Config

| Command | Purpose |
| --- | --- |
| `tg auth login <name>` | Create an account slot and log in. `--no-login` creates the slot only; `--force` re-runs auth. |
| `tg auth logout [name]` | Revoke this CLI session. `--purge` also removes local account files. |
| `tg auth list` | List local account slots. |
| `tg auth switch <name>` | Set `default_account` for later commands. |
| `tg auth status [name]` | Show local account health. `--probe` adds a live Telegram check. |
| `tg sessions list` | List remote Telegram authorizations. |
| `tg sessions revoke [hash]` | Revoke a remote session, or use `--all-others` to keep only this CLI session. |
| `tg password set` | Set or rotate the Telegram cloud password. |
| `tg password disable` | Remove the Telegram cloud password. |
| `tg config get <key>` | Read one config value. `--no-redact` shows `api_hash` verbatim. |
| `tg config set <key> <value>` | Write one config value. `--force` is required for `api_hash`. |
| `tg config unset <key>` | Remove one config value. |
| `tg config edit` | Open `$VISUAL` / `$EDITOR` on the config file with rollback on parse failure. |
| `tg config path` | Print the resolved config path. |
| `tg config show` | Print merged config with secrets redacted. |

### Chats, Messages, Search

| Command | Purpose |
| --- | --- |
| `tg chat list` | List dialogs. |
| `tg chat info <ref>` | Show one user, chat, bot, or channel. |
| `tg chat mark-read <ref>` | Mark a chat as read. `--max-id` limits the range. |
| `tg chat join <ref>` | Join a channel, group, or invite link. |
| `tg chat leave <ref>` | Leave a channel or supergroup. Prompts unless `--yes`. |
| `tg chat mute <ref>` | Mute notifications with `--duration`, `--until`, or `--forever`. |
| `tg chat unmute <ref>` | Restore notifications. |
| `tg chat archive <ref>` | Move a chat to the archive folder. |
| `tg chat unarchive <ref>` | Move a chat back to the main folder. |
| `tg chat members <ref>` | Command shape is present; Telegram participant loading is not wired yet. |
| `tg msg list <ref>` | List message history. |
| `tg msg send <ref> [text...]` | Send text. Repeat `--file <path>` to attach one or more files; text becomes the first media caption. Use `--name` to override upload filenames. |
| `tg msg download <msg-ref>` | Download photo, video, document, or other message media. Defaults to the media filename; use `-o/--output` for a file path or existing directory. |
| `tg msg edit <msg-ref>` | Edit a message. |
| `tg msg delete <msg-ref>...` | Delete messages. Add `--revoke` when deleting for everyone is required. |
| `tg msg forward <msg-ref>... --to <ref>` | Forward messages. |
| `tg msg react <msg-ref>` | Set or clear a reaction. |
| `tg msg pin <msg-ref>` / `tg msg unpin <msg-ref>` | Pin or unpin a message. |
| `tg msg schedule-list <ref>` | List scheduled messages. |
| `tg msg schedule-cancel <ref> <id>...` | Cancel scheduled messages. |
| `tg msg link <msg-ref>` | Print the public `t.me` link for a message when available. |
| `tg search msg <query>` | Search messages globally, or use `--in <ref>` for one chat. |
| `tg search chat <query>` | Search chats, users, channels, and bots. |

### Contacts And Profile

| Command | Purpose |
| --- | --- |
| `tg contact list` | List contacts. `--blocked` lists blocked users instead. |
| `tg contact add <phone>` | Add a contact by phone number. |
| `tg contact delete <ref>` | Delete a contact. |
| `tg contact block <ref>` | Block a user, bot, or channel. |
| `tg contact unblock <ref>` | Unblock a user, bot, or channel. |
| `tg me` | Print the current Telegram identity. |
| `tg profile set-name <first>` | Set first name; `--last` sets or clears last name. |
| `tg profile set-username <username>` | Set or clear the public username. |
| `tg profile set-bio <text>` | Set bio; `-` reads stdin. |
| `tg profile set-photo <path>` | Set profile photo; `-` reads stdin bytes. |
| `tg profile delete-photo` | Remove the current profile photo. Prompts unless `--yes`. |
| `tg profile set-status <online|offline>` | Set online visibility status. |

Peer refs accept usernames (`@name` or `name`), phone numbers, `me`, `saved`,
supported `t.me` / `tg://resolve` links, and the copy-paste refs printed by
`tg chat list`: `u:<id>:<access_hash>` for users, `g:<id>` for legacy groups,
and `c:<id>:<access_hash>` for channels or supergroups. Numeric IDs still work
when the local peer cache is warm, but the printed refs are the stable CLI form.
Message refs are `<peer-ref>:<message-id>` and are printed by `tg read` /
`tg msg list`, for example `@news:42` or `c:100:555:42`.

The old `tg account` group has been removed. Use `tg auth login`, `tg auth list`,
and `tg auth switch` instead.

## Output

Human output by default. Add `--json=<fields>` to get structured output.
List commands emit ndjson (one JSON object per line); scalar commands
emit a single object.

```sh
tg auth list --json=name,state,api_id
# {"name":"work","state":"AUTHED","api_id":12345}

tg read @news --limit 1 --json
# {"ref":"@news:42","id":42,"date":"...","from":{"ref":"@alice","id":123,"type":"user","title":"Alice","username":"alice","link":"https://t.me/alice"},"text":"hello"}

tg auth list --json=name,state --jq='.name'
# work

tg auth list --json=name,state --template='{{range .}}{{.name}}{{"\n"}}{{end}}'
# work
```

`--jq` and `--template` refine JSON output and are only valid with
`--json`. `--output=json` is equivalent to `--json` with all fields.

## Configuration

`~/.config/tg/config.toml`, `0600`. Everything is optional except `version`.

```toml
version = 1
default_account = "work"
api_id = 12345
api_hash = "..."

[output]
format = "human"   # or "json"

[log]
level = "warn"     # error | warn | info | debug

[flood_wait]
mode = "fail"      # or "wait"
max_seconds = 30
```

Values can also come from `TG_*` environment variables or command flags.
Precedence: **flag > env > config file > default**.

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | Success. Empty successful results also exit 0. |
| 1 | Unknown error, or an error that was already printed by the command. |
| 2 | Usage or configuration error. |
| 3 | Authentication is required, expired, or revoked. |
| 4 | Peer or message could not be found, or a ref was ambiguous. |
| 5 | Telegram says the peer/message/action is forbidden. |
| 6 | Telegram returned `FLOOD_WAIT` or rate/quota exhaustion. |
| 7 | Network or transport failure. |
| 8 | Conflict-like operation state: current session, invalid invite, revoke required, or bad cloud password. |
| 9 | Internal precondition, numeric peer cache miss, unsupported operation, missing media, or unavailable message link. |
| 72 | Another `tg` process holds the account lock. |
| 73 | User declined a confirmation prompt. |
| 130 | Interrupted by SIGINT / Ctrl+C. |

Under `--json`, errors are emitted on stderr as a single JSON object with
`error.code`, `error.message`, and `exit_code` fields. stdout stays data-only.

## Shell completion

```sh
tg completion bash   > /usr/local/etc/bash_completion.d/tg
tg completion zsh    > "${fpath[1]}/_tg"
tg completion fish   > ~/.config/fish/completions/tg.fish
```

Account names, enum flags, and recently used peer/message refs autocomplete.

## License

TBD.
