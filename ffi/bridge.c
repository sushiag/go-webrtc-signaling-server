#include "bridge.h"

void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message) {
    if (handler != NULL) {
        handler(sourceID, message);  // Call the Go function through the handler
    }
}
