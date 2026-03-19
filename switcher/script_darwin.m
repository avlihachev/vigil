#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <ApplicationServices/ApplicationServices.h>

char* run_applescript_sync(const char* source) {
    @autoreleasepool {
        NSString *script = [NSString stringWithUTF8String:source];
        NSAppleScript *as = [[NSAppleScript alloc] initWithSource:script];
        NSDictionary *error = nil;
        NSAppleEventDescriptor *desc = [as executeAndReturnError:&error];

        if (error) {
            NSString *errMsg = [NSString stringWithFormat:@"error:%@",
                [error objectForKey:NSAppleScriptErrorMessage] ?: @"unknown"];
            return strdup([errMsg UTF8String]);
        }
        NSString *str = [desc stringValue] ?: @"";
        return strdup([str UTF8String]);
    }
}

int check_accessibility_trusted(void) {
    return AXIsProcessTrusted() ? 1 : 0;
}

void prompt_accessibility_if_needed(void) {
    NSDictionary *opts = @{(__bridge NSString *)kAXTrustedCheckOptionPrompt: @YES};
    AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)opts);
}

// raise a specific window of appPID whose AXDocument matches docMatch
// or whose title contains titleMatch. Uses AX C API directly.
// returns: 1 = raised by doc, 2 = raised by title, 0 = not found, -1 = not trusted
int raise_window(pid_t appPID, const char* docMatch, const char* titleMatch) {
    if (!AXIsProcessTrusted()) {
        return -1;
    }

    AXUIElementRef app = AXUIElementCreateApplication(appPID);
    CFArrayRef windows = NULL;
    AXError err = AXUIElementCopyAttributeValue(app, kAXWindowsAttribute, (CFTypeRef*)&windows);
    if (err != kAXErrorSuccess || windows == NULL) {
        CFRelease(app);
        return 0;
    }

    NSString *docStr = docMatch ? [NSString stringWithUTF8String:docMatch] : nil;
    NSString *titleStr = titleMatch ? [NSString stringWithUTF8String:titleMatch] : nil;
    AXUIElementRef foundWindow = NULL;
    int matchType = 0;

    CFIndex count = CFArrayGetCount(windows);
    for (CFIndex i = 0; i < count; i++) {
        AXUIElementRef win = (AXUIElementRef)CFArrayGetValueAtIndex(windows, i);

        // try AXDocument match (CWD file URL)
        if (!foundWindow && docStr && [docStr length] > 0) {
            CFTypeRef docRef = NULL;
            if (AXUIElementCopyAttributeValue(win, kAXDocumentAttribute, &docRef) == kAXErrorSuccess && docRef) {
                NSString *winDoc = (__bridge NSString*)docRef;
                if ([winDoc isEqualToString:docStr] || [winDoc hasPrefix:docStr]) {
                    foundWindow = win;
                    matchType = 1;
                }
                CFRelease(docRef);
            }
        }

        // try title match
        if (!foundWindow && titleStr && [titleStr length] > 0) {
            CFTypeRef titleRef = NULL;
            if (AXUIElementCopyAttributeValue(win, kAXTitleAttribute, &titleRef) == kAXErrorSuccess && titleRef) {
                NSString *winTitle = (__bridge NSString*)titleRef;
                if ([winTitle rangeOfString:titleStr].location != NSNotFound) {
                    foundWindow = win;
                    matchType = 2;
                }
                CFRelease(titleRef);
            }
        }
    }

    int result = 0;
    if (foundWindow) {
        AXUIElementPerformAction(foundWindow, kAXRaiseAction);
        NSRunningApplication *runningApp = [NSRunningApplication runningApplicationWithProcessIdentifier:appPID];
        #pragma clang diagnostic push
        #pragma clang diagnostic ignored "-Wdeprecated-declarations"
        [runningApp activateWithOptions:NSApplicationActivateIgnoringOtherApps];
        #pragma clang diagnostic pop
        result = matchType;
    }

    CFRelease(windows);
    CFRelease(app);
    return result;
}

pid_t find_app_pid(const char* appName) {
    NSString *name = [NSString stringWithUTF8String:appName];
    for (NSRunningApplication *app in [[NSWorkspace sharedWorkspace] runningApplications]) {
        if ([app.localizedName isEqualToString:name]) {
            return app.processIdentifier;
        }
    }
    return 0;
}
