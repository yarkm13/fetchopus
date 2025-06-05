# Fetchopus

A multiprotocol multithread file downloader written in Go, designed to handle large downloads jobs with resume capability.

## Features

- Download files from various type of servers
- Parallel downloads with configurable thread count
- Resume capability through job files
- Recursive directory listing

## Usage

```
./fetchopus --url ftp://user@server.com/path --target-dir /local/path --threads 4
```

To resume a download:

```
./fetchopus --job myjob.dljob
```

## Building

```
go build -o fetchopus
```
