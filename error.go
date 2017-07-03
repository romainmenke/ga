package ga

// Error is the ga Error type
type Error string

func (e Error) Error() string { return string(e) }

// ErrClientClosed occurs after a Client was closed and a new action was attempted.
// It is also returned by Client.Start()
const ErrClientClosed = Error("ga client closed")

// ErrAlreadyStarted occurs when multiple Client.Start() calls are made.
const ErrAlreadyStarted = Error("ga client already started")

// ErrGoogleAnalytics occurs when POST calls to Google Analytics fail.
const ErrGoogleAnalytics = Error("google analytics api error")
