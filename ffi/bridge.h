#ifndef BRIDGE_H
#define BRIDGE_H

typedef void (*MessageHandler)(const char* sourceID, const char* message);

// Function to call the Go callback
void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message);

#endif
