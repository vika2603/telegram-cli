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
| `tg session list` | List remote Telegram authorizations. |
| `tg session revoke [hash]` | Revoke a remote session, or use `--all-others` to keep only this CLI session. |
| `tg password set` | Set or rotate the Telegram cloud password. |
| `tg password disable` | Remove the Telegram cloud password. |
| `tg config get <key>` | Read one config value. `--no-redact` shows `api_hash` verbatim. |
| `tg config set <key> <value>` | Write one config value. `--force` is required for `api_hash`. |
| `tg config unset <key>` | Remove one config value. |
| `tg config edit` | Open `$VISUAL` / `$EDITOR` on the config file with rollback on parse failure. |
| `tg config path` | Print the resolved config path. |
| `tg config show` | Print merged config with secrets redacted. |

### Daemon

A per-account background service that holds a long-lived MTProto session
so subsequent commands skip the dial / auth-resume cost (~1 s → ~0.25 s)
and `tg watch` can stream real-time updates over a Unix socket instead
of opening its own connection. Registration uses the host's native
service manager: launchd on macOS, systemd-user on Linux. The daemon is
optional — every command still works without one. Add `--no-daemon` to
any command to force a fresh MTProto session even when a daemon is
reachable.

| Command | Purpose |
| --- | --- |
| `tg daemon install` | Register the per-account daemon with the host service manager. `--force` reinstalls; `--log-file` / `--log-max-mb` tune logging. |
| `tg daemon uninstall` | Remove the service registration. Keeps `daemon.log` and `updates.ndjson` so a re-install can pick them up. |
| `tg daemon start` / `tg daemon stop` | Ask the OS to start or stop the service. |
| `tg daemon status` | JSON-emitting status (installed / running / pid / log + updates paths). |
| `tg daemon logs [-f] [-n N]` | Tail the daemon log; `-f` follows. |

Daemon artifacts live under `~/.config/tg/accounts/<account>/daemon/`:
`daemon.log` (service stdout+stderr, rotated at 10 MB by default),
`updates.ndjson` (append-only stream of every MTProto update — tailable
even without a connected subscriber), and `daemon.sock` (Unix socket
client commands route through when present).

### Chats, Messages, Search

| Command | Purpose |
| --- | --- |
| `tg chat list` | List dialogs. |
| `tg chat info <ref>` | Show one user, chat, bot, or channel. `--full` adds members/admins/online counts, about, linked discussion group, pinned message, and slow mode (supergroups/channels only). |
| `tg chat create <title>` | Create a supergroup. `--forum` enables topics; `--about` sets the description. |
| `tg chat delete <ref>` | Delete a supergroup or channel (irreversible). Prompts unless `--yes`. |
| `tg chat edit <ref>` | Edit a supergroup: `--title`, `--about`, `--public <name>` / `--private`; toggles: `--forum`/`--no-forum`, `--hide-members`/`--show-members`, `--hide-history`/`--show-history`, `--slow-mode <s>`, `--no-forwards`/`--allow-forwards`, `--need-approval`/`--no-need-approval` (public only). |
| `tg chat topic list <ref>` | List a forum supergroup's topics. `--search` filters by title; `--limit` caps results (single page only, ~100 max — pagination not implemented). |
| `tg chat topic create <ref> <title>` | Create a topic. `--icon-color`, `--icon-emoji`, and `--random-id` (idempotent retry). |
| `tg chat topic info <ref> <topic-id>` | Show details for one topic. |
| `tg chat topic mute <ref> <topic-id>` / `tg chat topic unmute <ref> <topic-id>` | Mute or unmute a single topic. Mute takes `--duration`, `--until`, or `--forever` (default forever). |
| `tg chat topic read <ref> <topic-id>` | Mark a topic as read. |
| `tg chat topic edit <ref> <topic-id>` | Rename (`--title`), close/reopen (`--close`/`--reopen`), hide/unhide (`--hide`/`--unhide`). |
| `tg chat topic pin <ref> <topic-id>` / `tg chat topic unpin <ref> <topic-id>` | Pin or unpin a topic. |
| `tg chat topic delete <ref> <topic-id>` | Delete a topic and its history. Prompts unless `--yes`. |
| `tg channel create <title>` | Create a broadcast channel. `--about` sets the description. |
| `tg channel delete <ref>` | Delete a channel (irreversible). Prompts unless `--yes`. |
| `tg channel edit <ref>` | Edit a channel: `--title`, `--about`, `--public <name>` / `--private`; toggles: `--no-forwards`/`--allow-forwards`, `--signatures`/`--no-signatures`. |
| `tg channel discussion link <channel> <group>` | Link a supergroup as the channel's discussion group (comments). |
| `tg channel discussion unlink <channel>` | Unlink the channel's discussion group. |
| `tg channel discussion candidates` | List supergroups eligible to be a discussion group. |
| `tg chat mark-read <ref>` | Mark a chat as read. `--max-id` limits the range. |
| `tg chat join <ref>` | Join a channel, group, or invite link (request-needed links report `requested`). |
| `tg chat join list <ref>` | List pending join requests. `--link` limits to one invite link. |
| `tg chat join approve <ref> <user>...` / `deny ...` | Approve/reject specific users, or `--all` (optionally `--link`) for every pending request. |
| `tg chat leave <ref>` | Leave a channel or supergroup. Prompts unless `--yes`. |
| `tg chat mute <ref>` | Mute notifications with `--duration`, `--until`, or `--forever`. |
| `tg chat unmute <ref>` | Restore notifications. |
| `tg chat archive <ref>` | Move a chat to the archive folder. |
| `tg chat unarchive <ref>` | Move a chat back to the main folder. |
| `tg chat pin <ref>` | Pin a chat to the top of the chat list. |
| `tg chat unpin <ref>` | Unpin a chat from the top of the chat list. |
| `tg chat photo set <ref> <path>` | Set a group/channel photo (`-` reads stdin). Also available as `tg channel photo set`. |
| `tg chat photo clear <ref>` | Remove a group/channel photo. Also `tg channel photo clear`. |
| `tg chat member list <ref>` | List members (admins show their custom `rank`/title). `--filter recent\|admins\|bots\|kicked\|banned\|contacts`, `--search`, `--limit`; `--via-link <link>` lists users who joined via that invite link. |
| `tg chat invite <ref> <user>...` | Add users to a group or channel. Each row reports `invited` true/false plus a `skip_reason` (e.g. `privacy_restricted`) when Telegram declined to add a user. |
| `tg chat invite create <ref>` | Create an invite link. `--title`, `--expire <RFC3339\|dur>`, `--usage-limit <n>`, `--request-needed`. |
| `tg chat invite list <ref>` | List invite links. `--revoked`, `--admin <user>`, `--limit`. |
| `tg chat invite revoke <ref> <link>` / `delete <ref> <link>` | Revoke a link, or delete a revoked one. |
| `tg chat ban <ref> <user>` / `tg chat unban <ref> <user>` | Ban (remove + block) or unban a member. Ban prompts unless `--yes`. |
| `tg chat admin promote <ref> <user>` / `tg chat admin demote <ref> <user>` | Grant or revoke admin rights. Promote takes `--title <rank>` (custom admin title, ≤16 chars); omit `--title` to keep the current title, pass `--title ""` to clear it. |
| `tg chat member set-perms <ref> <user>` | Set a member's permissions with `--deny`/`--allow` keywords (`send,media,stickers,bots,polls,links,invite,pin,info,topics`) and optional `--until <RFC3339\|dur>` (default permanent). |
| `tg chat member unset-perms <ref> <user>` | Clear all permission restrictions on a member. |
| `tg chat perms <ref>` | Set the group's default member permissions with `--deny`/`--allow` (same keywords). |
| `tg msg list <ref>` | List message history. Album members (same `grouped_id`) merge into one row with an `album` array (`--limit` counts an album as one). |
| `tg msg info <msg-ref>` | Show one message's details: media info (file name/size/mime, dimensions, duration, sticker emoji, web-page title/url), album members (each with media detail), and poll content (question, numbered options, tallies). |
| `tg msg send <ref> [text...]` | Send text. Repeat `--file <path>` to attach one or more files; text becomes the first media caption. Use `--name` to override upload filenames. `--sticker <ref>` / `--gif <ref>` send a sticker/gif — either a `msg sticker list`/`msg gif list` ref or a `<msg-ref>` of an existing one (no text/`--file`). |
| `tg msg sticker list` | List your stickers to get a sendable `ref`. `--recent` (default), `--faved`, `--installed` (sets), `--all` (recent+faved+all sets expanded; slow). |
| `tg msg sticker fave <ref>` / `tg msg sticker unfave <ref>` | Add or remove a sticker from favorites (`<ref>` is a list ref or `<msg-ref>`). |
| `tg msg gif list` | List your saved GIFs, each with a sendable `ref`. |
| `tg msg poll <ref> <question> <option>...` | Send a poll (≥2 options). `--multiple`, `--public` (default anonymous); `--correct <n>` makes it a quiz with optional `--explanation`. |
| `tg msg vote <msg-ref> <option-number>...` | Vote on a poll by 1-based option number (multiple numbers for multiple-choice polls); `--retract` to take back your vote. |
| `tg msg download <msg-ref>` | Download photo, video, document, or other message media. Defaults to the media filename; use `-o/--output` for a file path or existing directory. |
| `tg msg edit <msg-ref>` | Edit a message. |
| `tg msg delete <msg-ref>...` | Delete messages. Add `--revoke` when deleting for everyone is required. |
| `tg msg forward <msg-ref>... --to <ref>` | Forward messages. |
| `tg msg react <msg-ref>` | Set or clear a reaction. |
| `tg msg pin <msg-ref>` / `tg msg unpin <msg-ref>` | Pin or unpin a message. |
| `tg msg schedule-list <ref>` | List scheduled messages. |
| `tg msg schedule-cancel <ref> <id>...` | Cancel scheduled messages. |
| `tg msg link <msg-ref>` | Print the public `t.me` link for a message when available. |
| `tg watch [<ref>...]` | Stream new messages, edits, and deletes as ndjson until cancelled. `--kind=message,edit,delete` filters event kinds; `--limit N` exits after N events. When a daemon is running, routes through its socket; otherwise opens a foreground MTProto session. |
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
When the failure is a Telegram RPC error, `error.rpc_error` carries the exact
Telegram enum (e.g. `CHAT_ADMIN_REQUIRED`) for programmatic matching — present
even for errors that aren't mapped to a friendly `code`/`message`.

## Shell completion

```sh
tg completion bash   > /usr/local/etc/bash_completion.d/tg
tg completion zsh    > "${fpath[1]}/_tg"
tg completion fish   > ~/.config/fish/completions/tg.fish
```

Account names, enum flags, and recently used peer/message refs autocomplete.

## License

TBD.
