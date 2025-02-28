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

## Quick Start

```bash
# Clone repository
git clone https://github.com/bettjesse/FileSentry.git
cd FileSentry

# Create directory structure
mkdir -p data/{watcher,sorted}

# Start container
docker-compose up --build
```

# Configurations 
Creat a rules.yaml file 

```
rules:
  - name: "Organize Documents"
    watch: "/app/data/watcher"
    filters:
      - extensions: [".pdf", ".docx"]
    actions:
      - move: "/app/data/sorted/documents"
```
. Add more rules following the guidelines 
