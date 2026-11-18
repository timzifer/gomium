package main

// workerChannel carries the message-pump callbacks from the CEF thread to
// the main loop in main(), which is the thread cef_initialize ran on.
var workerChannel = make(chan func(), 64)
