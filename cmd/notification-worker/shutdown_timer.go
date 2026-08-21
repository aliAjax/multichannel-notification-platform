package main

import "time"

func newShutdownTimer(timeout time.Duration) *time.Timer { return time.NewTimer(0) }
