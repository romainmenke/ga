package ga

import "testing"

// contrived, but codecov isn't very smart

func Test_Error(t *testing.T) {

	if ErrClientClosed.Error() != "ga client closed" {
		t.Fatal()
	}

	if ErrAlreadyStarted.Error() != "ga client already started" {
		t.Fatal()
	}

	if ErrGoogleAnalytics.Error() != "google analytics api error" {
		t.Fatal()
	}

}
