# Golang Port Scanner

A simple and efficient port scanner written in Golang. This tool scans a specified range of ports on a given IP address concurrently using goroutines, making it fast and effective.

## Features

- Concurrent port scanning using goroutines
- Configurable IP address and port range
- Limits the number of concurrent goroutines to avoid system overload
- Displays open ports


## Usage

1. Clone the repository:

```bash
harish $ git clone https://github.com/pillaiharish/threat-detection-analysis-tools.git
harish $ cd port-scanner
harish $ sudo go run main.go 
Open port 5000 
Open port 7000 
Open port 62135 
2024/07/28 12:50:25 Port scanning completed
```

