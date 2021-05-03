package stoploss

// Notification wrapper
type Notify interface {
	Send(message string)
}
