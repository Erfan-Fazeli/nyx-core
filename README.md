# nyx-core

**nyx-core** is a versatile and extensible system designed for tunneling data over networks using custom protocol definitions. It enables secure and obfuscated data transfer by wrapping payloads in user-defined protocol formats, making it suitable for bypassing Deep Packet Inspection (DPI) systems, such as those employed by advanced firewalls. The system is highly configurable, allowing users to define custom protocols via a JSON configuration file (`pattern.json`) without modifying the core codebase.

## Features

- **Custom Protocol Simulation**: Define protocols for layers 3–7 (e.g., HTTP, DNS, or custom formats) using `pattern.json`.
- **Flexible Tunneling Modes**: Supports both **client** and **server** modes for versatile deployment scenarios.
- **Protocol Rotation**: Implements random, round-robin, or time-based protocol selection to avoid predictable traffic patterns.
- **Configurable Packet Building**: Supports dynamic packet construction with fields, sequences, computations (e.g., checksums, CRC), and randomization.
- **Obfuscation**: Designed to evade DPI and machine learning-based traffic analysis through protocol mimicry and randomization.
- **Extensible Architecture**: Add new protocol behaviors by editing `pattern.json`, minimizing code changes.
- **Verbose Logging**: Optional detailed logging for debugging and monitoring.

## How It Works

The nyx-core operates in two modes:
- **Client Mode**: Listens on a local port (default: 2020), wraps incoming data in a user-defined protocol, and forwards it to a remote server.
- **Server Mode**: Receives wrapped data, unwraps it, and forwards it to a VPN server (default: 127.0.0.1:4040).

Data is wrapped using protocol definitions in `pattern.json`, which can include headers, fields, and payloads (e.g., HTTP-like requests). The system supports dynamic values like connection IDs, timestamps, and computed fields (e.g., checksums).

## Installation

### Prerequisites
- Go 1.16 or higher
- A JSON configuration file (e.g., `pattern.json`) defining the protocols

### Steps
1. Clone the repository:
   ```bash
   git clone https://github.com/Erfan-Fazeli/nyx-core/
   cd nyx-core

   Build the project:
go build


Create or modify the pattern.json file to define your protocols (see Configuration).

Run the tunnel in client or server mode:
# Client mode
./nyx-core-tunnel -mode client -server example.com:443 -port 2020 -pattern pattern.json

# Server mode
./nyx-core-tunnel -mode server -vpn-server 127.0.0.1:4040 -port 2020 -pattern pattern.json



Configuration
The tunnel uses two main configuration files:

config.json: Defines general settings like mode, timeouts, and security options.
pattern.json: Defines protocol structures, including headers, fields, and payloads.

Example pattern.json
{
  "protocol_engine": {
    "name": "nyx-core https",
    "version": "3.2"
  },
  "protocols": [
    {
      "identifier": "fake_https",
      "transport": "tcp",
      "frame_structure": {
        "request_format": [
          "POST /api/data HTTP/1.1\r\n",
          {
            "Host": "api.example.com",
            "Content-Type": "application/octet-stream",
            "Content-Length": "${DATA_SIZE}"
          },
          "\r\n",
          "<<VPN_DATA>>"
        ],
        "line_ending": ""
      },
      "state_machine": {
        "initial_state": "connected",
        "states": [{"name": "connected", "transitions": []}]
      }
    }
  ]
}

This example defines a fake_https protocol that mimics an HTTP POST request, embedding VPN data in the body.
Key Configuration Options

Protocol Rotation: Set protocol_rotation in config.json to random, round_robin, or time_based.
Timeouts: Configure connection, read, and write timeouts in config.json.
Security: Enable/disable features like connection_encryption or timing_obfuscation.

Usage
Running in Client Mode
./nyx-core-tunnel -mode client -server example.com:443 -port 2020 -verbose


Listens on port 2020 and forwards data to example.com:443 using the defined protocol.

Running in Server Mode
./nyx-core -mode server -vpn-server 127.0.0.1:4040 -port 2020 -verbose


Listens on port 2020, unwraps data, and forwards it to the VPN server at 127.0.0.1:4040.

Verbose Logging
Use the -verbose flag to enable detailed debug logs:
./nyx-core -mode client -server example.com:443 -port 2020 -verbose

Bypassing DPI and Machine Learning
The tunnel evades DPI and machine learning-based firewalls by:

Mimicking Legitimate Protocols: Wrapping data in formats like HTTP or DNS to appear as normal traffic.
Protocol Randomization: Rotating protocols to avoid predictable patterns.
Dynamic Packet Construction: Using computed fields (e.g., checksums) and randomization to make traffic unique.

Limitations

Complex Protocols: Protocols requiring complex handshakes (e.g., WebSocket, QUIC) may need code modifications beyond pattern.json.
Lower Layers: Full simulation of layer 2 (e.g., Ethernet) or advanced layer 3 features may require additional code.

Extending Protocols
To create a new protocol (e.g., DNS or ICMP), edit pattern.json to define the structure:

Use frame_structure for application-layer protocols (e.g., HTTP).
Use layer_stack for lower-layer protocols (e.g., IPv4, TCP).
Define fields with types (uint8, string, etc.), computations (checksum, crc32), or sequences.

Example for a DNS-like protocol:
{
  "identifier": "fake_dns",
  "transport": "udp",
  "frame_structure": {
    "request_format": [
      "${RANDOM_ID}",
      "<<VPN_DATA>>"
    ]
  }
}

License
This project is licensed under the MIT License. See the LICENSE file for details.
Contact
For questions or issues, please open an issue on GitHub or contact your.email@example.com.

Last Updated: July 19, 2025
