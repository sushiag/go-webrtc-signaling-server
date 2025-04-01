#include <stdio.h>

typedef void (*MessageHandler)(const char* sourceID, const char* message);

// This function will call the Go callback.
void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message) {
    if (handler != NULL) {
        handler(sourceID, message);  // Call the Go function through the handler.
    }
}

