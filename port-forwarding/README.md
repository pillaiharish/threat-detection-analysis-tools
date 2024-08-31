# TCP Port Forwarder in Golang

This repository contains a simple and efficient TCP port forwarder written in Golang. The port forwarder listens on a specified local port and forwards incoming connections to a different port on a remote server. This project also includes a basic TCP server for testing purposes.

## Features

- **Concurrent Connections:** Handles multiple connections simultaneously using Goroutines.
- **Easy Configuration:** Modify the ports and IP addresses in the code to suit your needs.
- **Simple Logging:** Basic logging to monitor connections and errors.

## Getting Started

### Prerequisites

- Go 1.18+ installed on your system.

### Clone the Repository

```bash
harish $ git clone git@github.com:pillaiharish/threat-detection-analysis-tools.git
```

### Running Test Server
First run the test TCP server on port 8081 this will be our target to connect to using the port forwarder
```bash
harish $ cd threat-detection-analysis-tools/port-forwarding/tests
harish $ sudo go run test_port_forwarding.go
```

### Running the Port Forwarder
To run the port forwarder, use the following command:
```bash
harish $ cd threat-detection-analysis-tools/port-forwarding
harish $ go run main.go
```
This will start the server, listening on port 8080 and forwarding connections to the remote server on port 8081.

### Testing with Telnet or Netcat
You can test the port forwarder using telnet or nc (Netcat):


- Using Telnet:
```bash
harish $ telnet localhost 8080
```


- Using Netcat:
```bash
harish $ echo "Hello, Server!" | nc localhost 8080
```


- Using curl:
```bash
harish $ curl -X POST http://localhost:8080 -d "Hello, Server from curl"
curl: (1) Received HTTP/0.9 when not allowed
```

All methods should forward your message to the test server, which will respond with "Message received."


## Screeshot for Test Server run
![Screeshot for test server](https://github.com/pillaiharish/threat-detection-analysis-tools/port-forwarding/screen-captures/blob/main/test_port_forwarding.png)


## Screeshot for Port Forwarder
![Screeshot for port forwarder](https://github.com/pillaiharish/threat-detection-analysis-tools/port-forwarding/screen-captures/blob/main/main_tcp_port_forwarder.png)


## Screeshot for Telnet
![Screeshot for telent](https://github.com/pillaiharish/threat-detection-analysis-tools/port-forwarding/screen-captures/blob/main/telnet_on_8080.png)


## Screeshot for Curl
![Screeshot for curl](https://github.com/pillaiharish/threat-detection-analysis-tools/port-forwarding/screen-captures/blob/main/curl_on_8080.png)