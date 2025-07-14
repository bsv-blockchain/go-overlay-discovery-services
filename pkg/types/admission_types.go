package types

// AdmissionMode defines how a lookup service receives output admissions
type AdmissionMode string

const (
	AdmissionModeLockingScript AdmissionMode = "locking-script"
	AdmissionModeWholeTx       AdmissionMode = "whole-tx"
)

// SpendNotificationMode defines how a lookup service is notified of spends
type SpendNotificationMode string

const (
	SpendNotificationModeNone     SpendNotificationMode = "none"
	SpendNotificationModeTxid     SpendNotificationMode = "txid"
	SpendNotificationModeScript   SpendNotificationMode = "script"
	SpendNotificationModeWholeTx  SpendNotificationMode = "whole-tx"
)