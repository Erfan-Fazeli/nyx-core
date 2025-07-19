# nyx-core 🌌

**nyx-core** is a powerful and flexible system crafted for tunneling data across networks with custom protocol definitions. It ensures secure and stealthy data transfer by wrapping payloads in user-defined protocol formats, making it ideal for bypassing advanced Deep Packet Inspection (DPI) systems, such as those used in sophisticated firewalls. With a highly configurable architecture, users can define custom protocols through a simple JSON file (`pattern.json`) without touching the core codebase.

## ✨ Features

- **Custom Protocol Simulation**: Craft protocols for layers 3–7 (e.g., HTTP, DNS, or bespoke formats) using `pattern.json`.
- **Flexible Tunneling Modes**: Seamlessly operates in **client** or **server** modes to suit diverse deployment needs.
- **Protocol Rotation**: Supports random, round-robin, or time-based protocol switching to evade predictable traffic patterns.
- **Dynamic Packet Building**: Enables intricate packet construction with fields, sequences, computations (e.g., checksums, CRC), and randomization.
- **Stealth Obfuscation**: Engineered to outsmart DPI and machine learning-based traffic analysis through protocol mimicry and randomization.
- **Extensible Design**: Easily add new protocol behaviors by editing `pattern.json`, keeping code changes minimal.
- **Verbose Logging**: Optional detailed logs for in-depth debugging and monitoring.

## 🚀 How It Works

nyx-core operates in two intuitive modes:

- **Client Mode**: Listens on a local port, wraps incoming data in a custom protocol, and forwards it to a remote server.
- **Server Mode**: Receives wrapped data, unwraps it, and routes it to a VPN server.

Data is encapsulated using protocol definitions in `pattern.json`, supporting headers, fields, and payloads (e.g., HTTP-like requests). The system handles dynamic values like connection IDs, timestamps, and computed fields (e.g., checksums) for maximum flexibility.

## 🛡️ Bypassing DPI and Machine Learning

nyx-core excels at evading DPI and machine learning-based firewalls through:

- **Mimicking Legitimate Protocols**: Wraps data in formats like HTTP or DNS to blend in as normal traffic.
- **Protocol Randomization**: Rotates protocols to eliminate predictable patterns.
- **Dynamic Packet Construction**: Leverages computed fields (e.g., checksums) and randomization to ensure unique traffic signatures.

## ⚠️ Limitations

- **Complex Protocols**: Protocols with intricate handshakes (e.g., WebSocket, QUIC) may require code modifications beyond `pattern.json`.
- **Lower Layers**: Full simulation of layer 2 (e.g., Ethernet) or advanced layer 3 features may need additional code.

---

