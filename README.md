WebRTC Signaling Server & Client in Go (For Rust Integration)

Objectives:
- A signaling server in Go using WebSockets to help clients find and connect with each other.
 A WebRTC client in Go that talks to the server and handles ICE candidates and session details.
 An FFI (Foreign Function Interface) so Rust can use the Go WebRTC client.

Main components of the code are as follows:
Imports: Necessary packages for HTTP handling, WebSocket communication, synchronization, and logging.
Client Struct: Represents a connected client.
Room Struct: Represents a chat room containing clients.
Global Variables: Includes the WebSocket upgrader and a map of rooms.
Functions:
authenticate: Validates incoming requests.
handler: Manages WebSocket connections.
readMessages: Reads messages from connected clients.
main: Sets up the HTTP server and registers the handler.
