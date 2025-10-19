# log-archive

A small Go CLI tool to archive logs by compressing them into a timestamped tar.gz and storing them in a new directory.

## Features
- Compresses regular files (non-recursive) in a provided directory into a tar.gz archive.
- Skips files that look already compressed (.gz, .tgz, .tar.gz).
- Stores archives in a subdirectory named `archives` (or a custom destination).
- Appends an entry to `archive_history.log` with timestamp, archive name, file count and total bytes.
- Optionally remove original files after successful archiving.

## Build

```bash
go build -o log-archive
```

## Usage

Basic usage:

```bash
# archive /var/log and place archives in /var/log/archives
./log-archive /var/log
```

Options:

- `-dest <dir>` : optional destination directory (default: `<log-dir>/archives`)
- `-remove` : remove original files after they are archived
- `-v` : verbose logging

Examples:

```bash
# archive /var/log into default /var/log/archives
sudo ./log-archive /var/log

# specify a different destination
sudo ./log-archive -dest /tmp/my-logs /var/log

# archive and remove original files
sudo ./log-archive -remove /var/log

# verbose mode
sudo ./log-archive -v /var/log
```

Note: Archiving system log files (e.g. /var/log) may require root privileges to read and remove files. Use with caution on production systems — ensure services that write to logs handle log rotation appropriately (e.g., via logrotate) and consider stopping services or truncating active files instead of deleting them if needed.

## Archive file naming

Created archives will look like:

```
logs_archive_20251019_170959.tar.gz
```

and appended into `<dest>/archive_history.log` as lines like:

```
2025-10-19 17:09:59 UTC	archive=logs_archive_20251019_170959.tar.gz	files=34	total_bytes=1234567
```

## Extending

Possible improvements:
- Make the tool recursive and preserve directory structure.
- Add filters (extensions, age, size).
- Integrate with systemd timers or cron for scheduled execution.
- Implement safe handling of active log files (copytruncate, send SIGHUP to services, or integrate with logrotate).
