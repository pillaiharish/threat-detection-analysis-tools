# Network Speed Monitor

A real-time network monitoring tool built using **Golang** for the backend and **HTML/CSS/JavaScript** for the frontend. This tool continuously tracks your internet connectivity, upload/download speeds, and logs downtime and bandwidth issues. The frontend provides a visually appealing dashboard to display the results.

## Features
- **Real-Time Monitoring**: Check internet connectivity, upload/download speeds, and latency.
- **Downtime Logs**: Record timestamps for downtimes and bandwidth issues.
- **Frontend Dashboard**: View real-time stats on a responsive web UI.
- **Detailed Logs**: 
  - `full_logs.txt`: Comprehensive log of all events.
  - `filtered_logs.txt`: Logs filtered by low bandwidth or connectivity loss.
- **Backend Efficiency**: Uses a caching mechanism to reduce redundant speed tests.

## Prerequisites
- **Golang** (v1.19+ recommended)
- Internet connectivity for speed tests
- **`speedtest-go`** library: Install using `go get github.com/showwin/speedtest-go`

## Installation
After Clone the repository:
   ```bash
   cd network-speed-monitor
   go mod tidy
   go run cmd/main.go
   ```

## Access the frontend:
http://localhost:8080/static/index.html

## Access the backend logs:
http://localhost:8080/api/logs/filtered

## API Endpoints
GET /api/stats: Retrieve real-time stats (connectivity, upload/download speeds, latency).
GET /api/logs/full: Access the full logs.
GET /api/logs/filtered: Access the filtered logs.

# Medium Blog
https://medium.com/@harishpillai1994/network-speed-monitor-using-golang-2cac56a6910c 