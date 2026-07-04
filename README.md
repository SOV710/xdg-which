# xdg-which

`xdg-which` answers which `.desktop` file is found for a desktop file ID under the
XDG application search path.

It is useful when you edit desktop entries by hand and need to know whether the
desktop environment can discover the entry you expect, or whether another file
with the same desktop file ID shadows it.

## Feasibility

The tool is feasible because desktop application lookup is specified by the
freedesktop.org Desktop Entry, MIME Apps, and XDG Base Directory specifications.
The important lookup rules are local filesystem rules:

- application desktop entries live below `applications/` in `$XDG_DATA_HOME` and
  each directory in `$XDG_DATA_DIRS`;
- `$XDG_DATA_HOME` has higher priority than `$XDG_DATA_DIRS`, and earlier
  `$XDG_DATA_DIRS` entries have higher priority than later entries;
- a desktop file ID is the relative path below `applications/` with `/` replaced
  by `-`, for example `kde/org.example.App.desktop` becomes
  `kde-org.example.App.desktop`;
- `Hidden`, `OnlyShowIn`, `NotShowIn`, `NoDisplay`, and `TryExec` can affect
  whether the found entry is useful in a real desktop environment.

Existing tools cover adjacent pieces, but not this exact question:

- `xdg-mime query default <mime>` returns a desktop file ID for a MIME type, but
  does not explain where that ID resolves or whether it is shadowed.
- `desktop-file-validate` checks desktop entry syntax, but does not perform XDG
  lookup.
- `gtk-launch <desktop-id>` launches an application by ID, but does not show the
  resolved file or competing candidates.

## Install

```sh
go install github.com/sov710/xdg-which/cmd/xdg-which@latest
```

From this checkout:

```sh
go build ./cmd/xdg-which
```

## Usage

```sh
xdg-which org.example.App
xdg-which org.example.App.desktop
xdg-which --desktop KDE org.example.App
xdg-which -q org.example.App
```

Exit codes:

- `0`: found and no visibility problems were detected;
- `1`: no candidate was found;
- `2`: invalid usage or input;
- `3`: found, but the selected entry has visibility problems.
