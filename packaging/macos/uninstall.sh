#!/usr/bin/env sh
set -eu
launchctl unload /Library/LaunchDaemons/com.simplefrp.client.plist 2>/dev/null || true
rm -f /Library/LaunchDaemons/com.simplefrp.client.plist
rm -rf "/Library/Application Support/SimpleFRP" "/Library/Logs/SimpleFRP"
