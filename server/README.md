# WEBRTC SERVER 

## Packages

- server
- register
- db
- main

# sever Package 

## REST API End Points

| POST   | /register         | Register new user                      | registerNewUser()       |
| POST   | /updatepassword   | Change password                        | updatePassword()        |
| POST   | /regenerate       | Regenerate API key                     | regenerateNewApiKeys    |
| GET    | /ws               | Upgrade to WebSocket (auth via header) | with API-key to auth    |

## POST requests accept `application/json`

Example: Register/Regenerate API KEY

{
  "Username": "spongebob",
  "Password": "mysecret99"
}

Example: Messages Type

{
  "msg_type": 0,
  "from": 1,
  "payload": {}
}

## message Types

| Type         | int | Description						   |
| CreateRoom   |  0  | Create a new room                   |
| JoinRoom     |  1  | Join existing room                  |
| SDP          |  2  | Send SDP offer/answer               |
| ICECandidate |  3  | Exchange ICE candidates             |
| LeaveRoom    |  4  | Leave the room                      |
| RoomCreated  |  5  | Server response after room creation |


## parameters

| Param 	| Description																		 |
| coon    	| a *Connection struct that represent the newly authenticated Websocket connection.  |
| msg     	| a  pointer to smsg.MessageRawJSONPayload containing the type, sender, receiver and |
|         	| raw JSON payload.                                                                  |
| post    	| a string that represent the TCP port                                               |
| queries	| a db.Queries instances (sqlc-generated) for the database                           |
| w, r	  	| Standard HTTP request/response													 |
| newConnCH | Channel used to pass new connection into the system.		   						 |

## Room Cycle and Connection Management

- Room Creation 
- - createRoom(hostiD uint64) : Allocates new room with a unique room ID and Host (first user who joins the room/created it)

User: Host user that created
ReadMap: Tracks ready users
JoinOrder: The sequences of users joined

- Joining Room
- - addUserToRoom(roomID, joiningUserID): adds a user to an existing room, sends a RoomJoined messages to the joining users with the list of the current members.

Users: Maps the eixsting users in the room
ReadyMap: Tracks ready users
JoinOrder: The squences of users joined

- Are in the Same Room
- - AreInSameRoom(roomId, usersIDs []uint64) bool: This check wether the provided user ID is in the same room, only returns if true.

- Leave Room / Disconnect
- - Cleans up user's entry in the `Room.Users` and aassigns new Host.

## Main Functions

| Function 				| Description																					|
| handleWSEndpoint()	| This handles incoming /ws upgrade request from the client and authenticates from via API-Key. |
| StartSSever()			| This launches the HTTP server that authenticates and handles websocket signaling.			    |
| NewWebsockerManager()	| This manages the Websocket connections.														|

# register Package

This package contains the database function and api-key generator for the SQLite database during user registration and regeneration. 

The following package contains:

- GenerateAPIKey.go: This generates API key for each user and renegerate it.
- dtabase.go: This handles the SQLite database steup for the signaling server.

# db Package

This package contains the schema and query for the SQLite database generated db, modoels and queries.

The following package contains:

- schema.sql: table for users
- query.sql: query for the users.
- db.go, query.sql.go, models.go: generated using sqlc.yaml.

## Notes 

- to start server, go to server/cmd/main.go
- if post is "0", the os will automatically asignn an avaiable port.
- SERVER_HOST: host to this binds to the server
- Messages are routed using the WebsocketManger's internet connection map.
- SafeWriteJson is used to ensure thread-safe writes to connections.
- Addicational room onwership and Validation Logic (same-room logic for users)
- readLoop() pushes incoming messages into the WebsocketManager's `messageChabn`, writeLoop() handles outgoing messages from the internal `writeChan`
