#ifndef BRIDGE_H
#define BRIDGE_H

#include <stdio.h> // Move it here

typedef void (*MessageHandler)(const char* sourceID, const char* message);

void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message);

#endif  // Correct `#endif`
