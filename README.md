# FileSentry 👀
Real-time file monitoring with Docker support

## Features
- Real-time file event tracking (create/modify/rename/delete)
- Docker-first design
- Lightweight Alpine-based image (~10MB)

## Quick Start

### Using Docker 
```bash
# Clone repo
https://github.com/bettjesse/FileSentry.git
cd FileSentry

# Build & run
docker-compose up --build

## create a rules.yaml file a the root folder 
rules:
  - name: "Sort Images"
    watch: "/app/watcher"
    filters:
      - extension: ".jpg"
    actions:
      - move: "/app/sorted/images"

# Test in another terminal:
touch watcher/test.txt
```
