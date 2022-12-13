package midcobra

import (
	"time"

	"github.com/getsentry/sentry-go"
)

func Sentry(opts *SentryOpts) Middleware {
	return &sentryMiddleware{
		opts:     *opts,
		disabled: doNotTrack() || opts.SentryDSN == "",
	}
}

type SentryOpts struct {
	AppName    string
	AppVersion string
	SentryDSN  string // used by error reporting
}

type sentryMiddleware struct {
	// Setup:
	opts     SentryOpts
	disabled bool

	executionID string
}

// sentryMiddleware implements interface Middleware (compile-time check)
var _ Middleware = (*sentryMiddleware)(nil)

func (m *sentryMiddleware) preRun(cmd Command, args []string) {

}

func (m *sentryMiddleware) postRun(cmd Command, args []string, runErr error) {
	if m.disabled {
		return
	}
	initSentry(m.opts, m.executionID)
}

func (m *sentryMiddleware) withExecutionID(execID string) Middleware {
	m.executionID = execID
	return m
}

func initSentry(opts SentryOpts, executionID string) {
	sentrySyncTransport := sentry.NewHTTPSyncTransport()
	sentrySyncTransport.Timeout = time.Second * 2
	release := opts.AppName + "@" + opts.AppVersion
	environment := "production"
	if opts.AppVersion == "0.0.0-dev" {
		environment = "development"
	}

	_ = sentry.Init(sentry.ClientOptions{
		Dsn:              opts.SentryDSN,
		Environment:      environment,
		Release:          release,
		Transport:        sentrySyncTransport,
		TracesSampleRate: 1,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			for i := range event.Exception {
				// edit in place and remove error message from tracking
				event.Exception[i].Value = ""
			}
			event.EventID = sentry.EventID(executionID)
			return event
		},
	})
}
