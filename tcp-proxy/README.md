# TCP Proxy with Traffic Logging

This is a simple TCP proxy written in Go that listens on a specified local port and forwards all incoming connections to a target host and port.  In addition to basic proxy functionality, this version logs the traffic passing between the client and the server to the console. This can be invaluable for debugging network communication.

## Features

*   **TCP Proxy:** Forwards TCP connections from a local port to a specified target.
*   **Traffic Logging:** Logs the data transmitted between the client and server to the console, tagged with `[CLIENT]` or `[SERVER]` to indicate the direction of traffic.
*   **Concurrency:** Handles multiple connections concurrently using Go routines.

## Prerequisites

*   Go (version 1.16 or later) installed on your system.
*   Basic understanding of TCP networking.


## Installation

1. Clone the repository:
```bash
git clone https://github.com/pillaiharish/threat-detection-analysis-tools.git
cd tcp-proxy
```

2. Build and run the proxy: 
```bash
go run main.go <local-port> target-host:port
```

### Example Usage

To forward traffic from `localhost:8080` to `httpforever.com:80`, run:
```bash
go run main.go 8080 httpforever.com:80
```

**Parameters:**

*   `<local-port>`: The local port number on which the proxy will listen for incoming connections.
*   `<target-host:port>`: The hostname or IP address and port number of the target server.

## How It Works

1.  The proxy listens for incoming TCP connections on the specified local port.
2.  When a new connection is accepted, it establishes a connection to the target server.
3.  The proxy then spawns two goroutines:
    *   One goroutine reads data from the client, logs it with a `[CLIENT]` prefix, and forwards it to the server.
    *   The other goroutine reads data from the server, logs it with a `[SERVER]` prefix, and forwards it to the client.
4.  This bidirectional data transfer continues until one of the connections is closed.

## Logging

The application logs the following information:

*   New connection information:  The remote address of the client and the target address are logged when a new connection is established.
*   Data transferred: All data transferred between the client and server is logged to the console, prefixed with `[CLIENT]` or `[SERVER]` to indicate the direction.

**Important:** Be aware that logging all traffic can generate a large amount of output, especially with high-traffic connections.  This proxy is intended for debugging and development purposes.  For production use, consider implementing more sophisticated logging mechanisms or disabling traffic logging.

## Contributing

Contributions are welcome!  Please fork the repository and submit a pull request with your changes.
