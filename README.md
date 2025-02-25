# FileSentry 👀
Real-time file monitoring with Docker support

## Features
- Real-time file event tracking (create/modify/rename/delete)
- Docker-first design
- Lightweight Alpine-based image (~10MB)

## Quick Start
```bash
# Clone repo
https://github.com/bettjesse/FileSentry.git
cd FileSentry

# Build & run
docker build -t filesentry .
docker run -v $(pwd)/watcher:/app/watcher -it filesentry

# Test in another terminal:
touch watcher/test.txt
```
