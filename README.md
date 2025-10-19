# log-archive
A small Go CLI tool to archive log files from a directory into a timestamped tar.gz archive and record the operation in a history log.

## What it does
- Collects regular (non-directory, non-symlink) files in a single directory (non-recursive).
- Skips files that already look compressed: `.gz`, `.tgz`, `.tar.gz`.
- Skips the history file (`archive_history.log`) if it exists in the source directory.
- Writes a tar.gz archive containing only the filenames (no full paths) into a destination directory.
- Creates the archive atomically using a temporary file suffixed with `.tmp` and renames it into place.
- Appends a single line to `archive_history.log` in the destination directory recording the timestamp, archive name, number of files, and total bytes.
- Does NOT delete or modify original files.

## Features
- Non-recursive collection of regular files only.
- Skips common compressed file extensions and the history file.
- Default destination is a sibling directory named `<log-directory>_archive` (see Usage).
- Verbose logging option for progress and warnings.

## Build

```bash
go build -o log-archive
```

## Usage

Basic usage:

```bash
./log-archive [options] <path/to/input/logs/>
```

Options:
- `-dest <dir>` : optional destination directory. Default: sibling directory named `<log-directory>_archive` (e.g. `/var/log` -> `/var/log_archive`).
- `-v` : enable verbose logging (prints skipped files, archived file names, warnings).

The program prints a final summary line to stdout:

```
Archive complete: /path/to/dest/logs_archive_<timestamp>.tar.gz
```

The CLI usage string follows:
```
Usage: <program> [options] <log-directory>
```

## Archive file naming

Created archives follow this pattern:

```
logs_archive_20060102_150405.tar.gz
```

Example with real timestamp:

```
logs_archive_20251019_170959.tar.gz
```

The archive contains only the filename entries (no directory paths) and is created atomically from a `.tmp` file.

## History log

Each run appends a single line to `archive_history.log` in the destination directory. The line format:

```
<timestamp_including_zone>  archive=<archive_filename>  files=<count>  total_bytes=<bytes>
```

Example:

```
2025-10-19 17:09:59 UTC	archive=logs_archive_20251019_170959.tar.gz	files=34	total_bytes=1234567
```

Timestamps use the format `2006-01-02 15:04:05 MST` (includes time zone).

## Examples

Archive `/var/log` into the default sibling destination:

```bash
sudo ./log-archive /var/log
```

Specify a different destination directory:

```bash
sudo ./log-archive -dest /tmp/my-logs /var/log
```

Verbose mode:

```bash
sudo ./log-archive -v /var/log
```

## Notes and caveats
- The tool is intentionally simple: it is non-recursive and does not preserve directory trees.
- It does not perform log rotation semantics (copytruncate, signaling services, etc.). For active log files, consider using proper log rotation tools or stopping services before archiving.
- Running against system log directories may require elevated privileges to read files.
- The tool does not delete original files — it only creates archives and updates the history log.

## Extending
Possible improvements:
- Add recursive mode and preserve directory structure.
- Add filters (extensions, age, size).
- Add dry-run mode.
- Integrate with logrotate, systemd timers, or cron for scheduled execution.

https://roadmap.sh/projects/log-archive-tool