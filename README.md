# WebRTC Signaling Server & Client in Go

## Objectives

- A signaling server in Go using WebSockets to help clients find and connect with each other.
- A WebRTC client in Go that talks to the server and handles ICE candidates and session details.
- REST API for user authenthication ('/ws', '/register', /updatepassword' and '/regenerate')
- Sqlite for lightweight and storage
- End-to-End test using tesify
- Channel-based conncurreny (no 'sync.' beside using in End-to-End testings)


## Overview

This project is a WebRTC signaling server & client implemented in Go using SQLite as the database.

## Features

- WebSocket-based signaling
- ICE candidate exchange and restart handling
- Room management and peer discovery
- DataChannel error handling
- STUN server configuration
- User registration, Authentication using API-Key to connect to the websocket,  update password and API-Key generation for websocket authentication.
- Lightweight SQLite

## Troubleshooting

### Common Issues

#### 1. WebSocket Connection Fails
- Ensure the server is running and accessible.
- Check if firewall rules allow WebSocket traffic.

#### 2. ICE Connection Fails
- Verify STUN server configuration.
- Check if network restrictions block WebRTC traffic.
