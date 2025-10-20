# log-archive
A small shell script tool that packages regular files from a single directory into a timestamped tar.gz archive and records the operation in a history log.

## Summary
- Collects regular (non-directory, non-symlink) files from a single directory (non-recursive).
- Skips files that appear already compressed: `.gz`, `.tgz`, `.tar.gz`.
- Skips the history file (`archive_history.log`) if present.
- Creates a tar.gz archive containing only the filenames (no directory paths).
- Writes the archive atomically: first to a `.tmp` file, then renames into place.
- Appends a single-line entry to `archive_history.log` in the destination directory with timestamp, archive name, file count, and total bytes.
- Does not delete or modify source files.

## Usage
```
chmod +x log-archive.sh
./log-archive [options] <path/to/input/dir>
```
Options:
- `-dest <dir>`  : destination directory for the archive and history log. Default: sibling directory named `<input_dir>_archive` (e.g. `/some/logs` → `/some/logs_archive`).
- `-v`           : verbose logging (prints skipped files, archived entries, warnings).

The program prints a short success line on completion, for example:
```
Archive complete: /path/to/dest/logs_archive_20251019_170959.tar.gz
```


## Archive file naming
Archives are named with a fixed prefix and timestamp:
```
logs_archive_20060102_150405.tar.gz
```
Notes:
- The archive entries are stored with only their base filenames (no directory entries).
- A temporary file with a `.tmp` suffix is used during creation and then renamed atomically into the final name.

## History log
Each run appends one line to `archive_history.log` in the destination directory. Format:
```
2006-01-02 15:04:05 PDT archive=<archive_filename> files=<count> total_bytes=<bytes>
```

Example:
```
2025-10-19 17:09:59 UTC	archive=logs_archive_20251019_170959.tar.gz files=34 total_bytes=1234567
```
Timestamps include the time zone.


## Examples
Archive `/some/logs` into the default sibling destination:
```bash
./log-archive.sh /some/logs
```

Specify a different destination directory:
```bash
./log-archive.sh -dest /a/different/dir /some/logs
```

Verbose mode:
```bash
./log-archive.sh -v /some/logs
```


## Notes and caveats
- Non-recursive: subdirectories are not traversed and directory trees are not preserved.
- The tool does not perform log rotation semantics (no copytruncate, no signaling). For active logs, consider stopping the service or using a rotation tool.
- Reading some system log directories may require elevated privileges.
- Originals are left intact; the tool only creates archives and updates the history log.

## Suggested extensions
- Add recursive mode and path-preserving archive option.
- Add filters by extension, age, or size, and a dry-run mode.
- Integrate with cron/systemd timers for scheduled runs.
