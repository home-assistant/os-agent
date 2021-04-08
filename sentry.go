package main

import "github.com/getsentry/sentry-go"

func filterSentry(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if version == "dev" {
		return nil
	}
	return event
}
