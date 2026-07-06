package sysutil

import "runtime"

func OS() string   { return runtime.GOOS }
func Arch() string { return runtime.GOARCH }
