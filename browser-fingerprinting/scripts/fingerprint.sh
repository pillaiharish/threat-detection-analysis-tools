#!/bin/bash
#
# fingerprint.sh — OS fingerprint scan for authorized intranet assessment.
# Invoked by the Go server with a single IP as argv[1]. Output goes to stdout
# only; the server captures it and files it. Do NOT write to client_info.txt
# from here (the server owns that file to avoid races).
#
# Usage: ./fingerprint.sh <IP_ADDRESS>
#
set -u

if [ $# -lt 1 ] || [ -z "$1" ]; then
  echo "Usage: $0 <IP_ADDRESS>"
  exit 2
fi

IP_ADDRESS="$1"

# Validate that the argument looks like an IPv4/IPv6 literal or hostname.
# Reject anything containing whitespace, shell metacharacters, or path
# separators — defense in depth against injection from client-controlled
# input (the server also sanitizes before calling).
case "$IP_ADDRESS" in
  *[![:alnum:]:.\-]*)
    echo "Rejected: invalid IP/host literal"
    exit 2
    ;;
esac

echo "Running OS fingerprinting on ${IP_ADDRESS}..."
echo "command: nmap -O -T3 --osscan-guess ${IP_ADDRESS}"
echo "----"

# -T3: polite, default-safe timing (per operator choice).
# --osscan-guess: report best-effort OS guesses for the triage dashboard.
# Exit code is captured by the caller; do not treat nmap nonzero as fatal
# (e.g. "0 hosts up" is a valid negative result meaning the boundary blocks us).
nmap -O -T3 --osscan-guess "$IP_ADDRESS" 2>&1
nmap_rc=$?

echo "----"
echo "nmap_exit_code: ${nmap_rc}"
exit 0