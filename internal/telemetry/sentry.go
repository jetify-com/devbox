package telemetry

import (
	"time"

	"github.com/getsentry/sentry-go"
)

type Sentry struct {
	disabled bool

	dsn string
}

func NewSentry(dsn string) *Sentry {
	return &Sentry{
		disabled: DoNotTrack() || dsn == "",
		dsn:      dsn,
	}
}

func (s *Sentry) Init(appName, appVersion, executionID string) {
	if s.disabled {
		return
	}

	sentrySyncTransport := sentry.NewHTTPSyncTransport()
	sentrySyncTransport.Timeout = time.Second * 2
	release := appName + "@" + appVersion
	environment := "production"
	if appVersion == "0.0.0-dev" {
		environment = "development"
	}

	_ = sentry.Init(sentry.ClientOptions{
		AttachStacktrace: true,
		EnableTracing:    true,
		Dsn:              s.dsn,
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
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{ID: DeviceID()})
		scope.SetContext("os", map[string]interface{}{
			"name": OS(),
		})
	})
}

// CaptureException
func (s *Sentry) CaptureException(runErr error) string {
	if s.disabled || runErr == nil {
		return ""
	}
	defer sentry.Flush(2 * time.Second)

	eventIDPointer := sentry.CaptureException(runErr)
	if eventIDPointer == nil {
		return ""
	}
	return string(*eventIDPointer)
}
