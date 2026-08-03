# cli-cheat-sheets (ccs)
Write your own cheat sheets in cli.

## Why

With so many tools in cli, custom scripts and functions, I forgot some commands or how to use some tool.

Have an entire list of cheat-sheets also does not help me to find what I need quickly and I spend a lot of time navigating the cheat-sheet instead of remember my favourite commands.

So, why not register all my favorite commands in files simple to edit (yaml or json), one file per cheat sheet, and every time I want to remember something just run `ccs ???` and it prints a table with the commands?

Well... here we are!


## Install

```
go install github.com/bhrott/ccs@latest
```

Or build it from a clone:

```sh
./build.sh              # builds ./bin/ccs
./build.sh --install    # builds and copies it to ~/.local/bin
```


## Usage

```sh
# Print a cheat sheet as a table
ccs tmux

# Print only the items matching a term (command or description)
ccs tmux pane

# List all your cheat sheets
ccs ls
```

Output:

```
tmux
Terminal multiplexer

┌───────────────────────┬──────────────────────────┐
│ COMMAND               │ DESCRIPTION              │
├───────────────────────┴──────────────────────────┤
│ Sessions                                         │
├───────────────────────┬──────────────────────────┤
│ tmux new -s mysession │ New session              │
│ tmux ls               │ List all sessions        │
│ tmux a -t mysession   │ Attach to session        │
│ ctrl+b s              │ View and switch sessions │
├───────────────────────┴──────────────────────────┤
│ Panes                                            │
├───────────────────────┬──────────────────────────┤
│ ctrl+b z              │ Zoom in/out of pane      │
│ ctrl+b x              │ Kill pane                │
└───────────────────────┴──────────────────────────┘
```

### Options

| Option | Description |
| --- | --- |
| `--plain` | Render without table borders, only aligned columns |
| `--no-color` | Render without colors (`NO_COLOR` is respected too) |
| `--path` | Print the cheat sheets path being used |
| `-h`, `--help` | Show the help |
| `-v`, `--version` | Show the version |

The table wraps the description column to the terminal width, so it also reads
fine inside a narrow tmux pane. Set `CCS_PLAIN=1` to always use the plain style.


## How to create cheat-sheets

Each cheat sheet is one file inside `~/.cheat-sheets`, and the folder is created
with an example on the first run:

```
~/.cheat-sheets
├── config.yaml     # colors, shared by every sheet (optional)
├── tmux.yaml       # ccs tmux
├── docker.yaml     # ccs docker
└── git.json        # ccs git
```

The **file name is the sheet id**, so adding a cheat sheet is just adding a
file, and removing one is just deleting it. Point `CHEAT_SHEETS_FILE_PATH` to
another folder to keep the sheets with your dotfiles:

```sh
export CHEAT_SHEETS_FILE_PATH=~/dotfiles/cheat-sheets
```

Run `ccs --path` to see which folder is being read.

Every file can be YAML or JSON (`.yaml`, `.yml` or `.json`). This is
`~/.cheat-sheets/tmux.yaml`:

```yaml
description: Terminal multiplexer

groups:
  - name: Sessions
    items:
      - command: tmux new -s mysession
        description: New session
      - command: tmux ls
        description: List all sessions

  - name: Panes
    items:
      - command: ctrl+b z
        description: Zoom in/out of pane
```

Explaining:

- the file name (`tmux.yaml`) is the id of your sheet, you will use this to print in cli
- `description:`: optional, shown under the title and on `ccs ls`
- `groups:`: split the sheet in smaller blocks
    - `- name:`: the group title
    - `  items:`: the list of your commands
        - `- command:`: the command, snippet, etc
        - `  description:`: any help text you want to add
- `id:`: optional, only when you want an id different from the file name

Groups are optional. Items can also be declared directly on the sheet:

```yaml
items:
  - command: my-command
    description: This is a description of my command
```

The same sheet in JSON (`~/.cheat-sheets/tmux.json`):

```json
{
  "description": "Terminal multiplexer",
  "groups": [
    {
      "name": "Sessions",
      "items": [
        { "command": "tmux ls", "description": "List all sessions" }
      ]
    }
  ]
}
```

### Colors

`config.yaml` holds the settings shared by every sheet, and it is optional:

```yaml
config:
  colors:
    title: "#5fd7ff"
    group: "#ff87d7"
    command: "#ffd75f"
    description: "#dadada"
    border: "#5f5f5f"
```

- `title`: the sheet id printed on top
- `group`: the group names
- `command`: the command column
- `description`: the description column
- `border`: the table borders

### Coming from a single file

Files with every sheet inside keep working, both in the folder and when
`CHEAT_SHEETS_FILE_PATH` points straight at the file:

```yaml
config:
  colors:
    title: "#5fd7ff"

sheets:
  - id: tmux
    description: Terminal multiplexer
    items:
      - command: tmux ls
        description: List all sessions
```

To split it, move each `- id: <name>` block to its own `<name>.yaml`, drop the
`sheets:` and `- id:` lines, unindent the rest, and move `config:` to
`config.yaml`.

Old files using `-- GROUP NAME --` items as separators keep working too: those
items are turned into groups automatically.


## Development

```sh
./test.sh            # gofmt + go vet + unit tests
./test.sh --cover    # same, with a coverage report
./build.sh -o /tmp   # build somewhere else

CHEAT_SHEETS_FILE_PATH=./cheat-sheets go run . tmux
```

The `cheat-sheets/` folder of this repo is a working example: one file per
sheet, plus `config.yaml`.

On every push and pull request the `ci` workflow runs `./test.sh` and only then
builds the binary on linux and macos, so a broken test blocks the build.

The code is split in:

- `internal/cheatsheet`: the model, the path resolution and the YAML/JSON loading, one file per sheet
- `internal/render`: the terminal table rendering
- `internal/cli`: argument parsing and commands
