package limiter

type Rule struct {
	ClientID        string
	API             string
	RequestsAllowed int
	WindowSeconds   int
}

type Decision struct {
	Allowed    bool
	Remaining  int
	RetryAfter int
}
