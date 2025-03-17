# FileSentry 🚀

[![Go Version](https://img.shields.io/badge/go-1.24+-blue)](https://golang.org/)

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Automated file management with Docker-powered workflows**

![Demo GIF](https://media.giphy.com/media/v1.Y2lkPTc5MGI3NjExZTBjY2Q4ZGNjYjU0YjM5YjYxYjI4Y2EzYjY5YjYyNDRkYzM5ZGM0NCZlcD12MV9pbnRlcm5hbF9naWZzX2dpZklkJmN0PWc/3ohzdIuqJCo7wMiew8/giphy.gif)

## Why FileSentry?

Solve these common file management problems:
- 🕵️ Manual file sorting/organization
- ⏳ Time-consuming repetitive tasks
- ☁️ Complex cloud sync setups
- 🚨 Missed file events/alerts

## Features

| Feature | Benefit |
|---------|---------|
| Real-time File Watching | Instant reaction to changes |
| YAML Rules Engine | No-code workflow configuration |
| Cross-Platform | Works anywhere Docker runs |
| Dry-Run Mode | Test rules safely |
| Regex-Based Renaming | Automatically clean up filenames (e.g. remove parentheses) |


## Quick Start

```bash
# Clone repository
git clone https://github.com/bettjesse/FileSentry.git
cd FileSentry

# Create directory structure
Create the unified data directory and its subdirectories on your host:
mkdir -p files/{watcher,torrents,documents,sorted/{images,docs,text_backups},media/movies,archive/documents}



# Configurations 
Create or update a rules.yaml file. For example:

```
rules:
  - name: "Sort Images"
    watch: "/app/files/watcher"
    filters:
      - extension: [".jpg", ".png"]
      - exclude: "(\\.tmp$|~$)"
    actions:
      - move: "/app/files/sorted/images"

  - name: "Organize PDFs"
    watch: "/app/files/watcher"
    filters:
      - extension: [".pdf"]
    actions:
      - move: "/app/files/sorted/docs"

  - name: "Backup Text Files"
    watch: "/app/files/watcher"
    filters:
      - extension: [".txt"]
    actions:
      - move: "/app/files/sorted/text_backups"

  - name: "Backup to S3"
    watch: "/app/files/watcher"
    filters:
      - extension: [".jpg", ".png"]
    actions:
      - move: "/app/files/sorted/images"
      - upload_to_s3: "s3://your-bucket-name/images/{{timestamp}}/{{filename}}"

  - name: "Clean Torrent Names"
    watch: "/app/files/torrents"
    filters:
      - extension: [".mkv", ".mp4"]
        operator: "AND"
        last_modified: "2h"   # Process files that are at least 2 hours old
    actions:
      - regex: '\([^\)]+\)'
        replace: ""         # Remove parentheses and their content
      - move: "/app/files/media/movies"

  - name: "Archive Old PDFs"
    watch: "/app/files/documents"
    filters:
      - extension: [".pdf"]
        last_modified: "720h"  # Files older than 30 days
        operator: "AND"
    actions:
      - move: "/app/files/archive/documents"

```
# Start container
docker-compose up --build

# Testing dry-run
``
1) DRY_RUN=true docker-compose up --build
```
# Testing functionality
For Torrents:
Place a file like movie (1080p).mp4 in files/torrents. After the file meets the last_modified condition, FileSentry should automatically rename it (removing the parentheses) and move it to files/media/movies.

For Other Rules:
Similarly, drop test files into files/watcher or files/documents to see them moved to their respective directories.

Dry-Run Mode:
To test without moving files, run:

```
DRY_RUN=true docker-compose up --build
```




