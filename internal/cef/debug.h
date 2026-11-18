#ifndef DEBUG_H
#define DEBUG_H

#include <stdio.h>

void goLogMessage(char *file, int line, char *func, char *message);
void goLogMessageAndDie(char *file, int line, char *func, char *message);
void goLogPointer(char *file, int line, char *func, char *message, void *ptr);
void goLogFuncCall(char *file, int line, char *func);
void goLogFuncExit(char *file, int line, char *func);

#define DEBUG(x) { goLogMessage(__FILE__, __LINE__, (char*) __func__, x); }
#define DEBUG_FATAL(x) { goLogMessageAndDie(__FILE__, __LINE__, (char*) __func__, x); }
#define DEBUG_POINTER(x, p) { goLogPointer(__FILE__, __LINE__, (char*)  __func__, x, (void*) p); }

#define DEBUG_CALL { goLogFuncCall(__FILE__, __LINE__, (char*) __func__); }
#define DEBUG_EXIT { goLogFuncExit(__FILE__, __LINE__, (char*) __func__); }


#endif