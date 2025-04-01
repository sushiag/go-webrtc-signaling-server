# WebRTC Signaling Server & Client in Go (For Rust Integration)

## Objectives:
- A signaling server in Go using WebSockets to help clients find and connect with each other.
- A WebRTC client in Go that talks to the server and handles ICE candidates and session details.
- An FFI (Foreign Function Interface) so Rust can use the Go WebRTC client.

## Overview
This project is a WebRTC signaling server implemented in Go with FFI bindings for Rust. It handles peer discovery, WebSocket-based signaling, API key authentication, and session management. The client-side WebRTC logic is also managed in Go, while the UI is handled separately using Svelte or Rust.

## Features
- WebSocket-based signaling
- API key authentication
- ICE candidate exchange and restart handling
- Room management and peer discovery
- DataChannel error handling
- STUN server configuration

## Installation
### Prerequisites
- Go 1.18+
- Rust (latest stable version)
- Node.js (for frontend development)
- WebRTC-compatible browsers
- Visual Studio Build Tools (for Windows users compiling Rust with MSVC)

### Steps
1. Clone the repository:
   ```sh
   git clone https://github.com/sushiag/go-webrtc-signaling-server.git
   cd go-webrtc-signaling-server
   ```
2. Install dependencies:
   ```sh
   go mod tidy
   ```
3. Build and run the server:
   ```sh
   go run main.go
   ```

## API Reference
### WebSocket Messages
- `offer` – Sent when a peer creates an SDP offer.
- `answer` – Sent when a peer responds with an SDP answer.
- `ice-candidate` – Sent when a new ICE candidate is discovered.
- `join` – Used to join a signaling room.
- `leave` – Used to leave a signaling room.

## Troubleshooting
### Common Issues
#### 1. WebSocket Connection Fails
- Ensure the server is running and accessible.
- Check if firewall rules allow WebSocket traffic.

#### 2. ICE Connection Fails
- Verify STUN server configuration.
- Check if network restrictions block WebRTC traffic.

#### 3. Rust Compilation Issues (Windows)
- Ensure Visual Studio Build Tools are installed.
- Run:
  ```sh
  rustup show
  ```
  and confirm `msvc` toolchain is used.

## Contribution Guidelines
1. Fork the repository and create a new branch.
2. Make your changes and run tests.
3. Submit a pull request with a clear description.

## License
This project is licensed under the MIT License.

