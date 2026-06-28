package main

import (
	"context"
)

type stubSink struct {
	entries []IPLog
}

func (s *stubSink) log(_ context.Context, entry IPLog) {
	s.entries = append(s.entries, entry)
}

func (s *stubSink) close() error { return nil }