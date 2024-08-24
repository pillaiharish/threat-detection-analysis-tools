# Go IP Address Scanner

A simple IP address scanner built with Go to scan a local network (such as your home Wi-Fi) and identify active devices by pinging each IP address in a specified range.


## Introduction

This Go program allows you to scan a range of IP addresses on your local network and identify which devices are active by sending ICMP (ping) requests. It's a useful tool for monitoring your network and detecting any unauthorized devices.

## How It Works

The IP address scanner works by:

1. Identifying the network range.
2. Pinging each IP address within the range.
3. Reporting which IP addresses are active (respond to the ping).

When an IP address is active, the program will print `IP <ip_address> is up`, indicating that a device on your network is using that IP address.

## Prerequisites

- **Go (Golang)**: Ensure that Go is installed on your machine. You can download it from the [official Go website](https://golang.org/dl/).
- **Superuser Permissions**: On most systems, sending ICMP packets (ping) requires elevated privileges.

## Usage

### Cloning the Repository

First, clone the repository to your local machine:

```bash
harish $ git clone git@github.com:pillaiharish/threat-detection-analysis-tools.git
harish $ cd threat-detection-analysis-tools/ip-scanner
harish $ sudo go run main.go 
Password:
IP 192.168.1.1 is up
IP 192.168.1.5 is up
IP 192.168.1.75 is up
IP 192.168.1.10 is up
IP 192.168.1.76 is up
Scan complete
```

## What "IP is up" Means

- **Device is Active**: It indicates that there is a device on the network that is actively using that IP address and is responding to ping requests.
- **Network Reachability**: The IP address is reachable, meaning that the network infrastructure (routers, switches, etc.) can route traffic to and from this IP address correctly.

## Clarifying the Concept

### IP Address
An IP address is a unique identifier assigned to devices on a network. Each device (like your phone, computer, smart TV, etc.) connected to the network has an IP address.

### Ping Request
A "ping" is a network utility used to test the reachability of a host on an IP network. It sends ICMP Echo Request packets to the target IP address and waits for an Echo Reply. If the target device responds, it means the device is active and reachable on the network.

### "IP is up"
This phrase is shorthand for saying that the device associated with that IP address is "alive" or "reachable" because it responded to the ping request.

## Real-World Meaning
If you see that "IP 192.168.1.5 is up" in the program output, it implies:

- **Active Device**: A device (such as a computer, phone, IoT device, etc.) is connected to the network using the IP address 192.168.1.5.
- **Network Presence**: The device is responding to network requests, which suggests it’s currently active and communicating on the network.

If an IP address does not respond (i.e., the program does not print "IP <ip_address> is up"), it could mean:

- The IP address is not currently assigned to any device.
- The device using that IP address is turned off or disconnected from the network.
- The device or network firewall is blocking ICMP (ping) requests.