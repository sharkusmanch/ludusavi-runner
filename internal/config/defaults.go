// Package config handles application configuration loading and validation.
package config

import "time"

// Default configuration values.
const (
	DefaultInterval        = 20 * time.Minute
	DefaultBackupOnStartup = true

	DefaultMetricsEnabled    = false
	DefaultMetricsPushgatewayURL = ""

	DefaultRetryMaxAttempts  = 3
	DefaultRetryInitialDelay = 5 * time.Second
	DefaultRetryMaxDelay     = 30 * time.Second

	DefaultAppriseEnabled = false
	DefaultAppriseURL     = ""
	DefaultAppriseKey     = ""
	DefaultAppriseNotify  = NotifyError

	DefaultLogLevel     = "info"
	DefaultLogMaxSizeMB = 10
)

// NotifyLevel represents when to send notifications.
type NotifyLevel string

const (
	// NotifyError sends notifications only on errors.
	NotifyError NotifyLevel = "error"
	// NotifyWarning sends notifications on errors and warnings.
	NotifyWarning NotifyLevel = "warning"
	// NotifyAlways sends notifications on every backup.
	NotifyAlways NotifyLevel = "always"
)

// IsValid returns true if the notify level is valid.
func (n NotifyLevel) IsValid() bool {
	switch n {
	case NotifyError, NotifyWarning, NotifyAlways:
		return true
	default:
		return false
	}
}

// String returns the string representation of the notify level.
func (n NotifyLevel) String() string {
	return string(n)
}
