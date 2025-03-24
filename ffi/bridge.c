#include <stdlib.h>

typedef void (*MessageHandler)(const char* sourceID, const char* message);

void CallMessageHandlerBridge(MessageHandler handler, const char* sourceID, const char* message) {
    if (handler != NULL) {
        handler(sourceID, message);
    }
}
