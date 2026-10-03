package link

import "testing"

func TestUnknownAndServerCodesAreRefetchable(t *testing.T) {
	codes := []string{"400", "403", "500", "502", "504", "some_unrecognised_code"}
	for _, code := range codes {
		e := ErrorCodeToLinkError(code)
		if e.IsPermanent() {
			t.Errorf("code %q classified as permanent: the file would stay unreadable until restart", code)
		}
		if !e.ShouldRefetch() {
			t.Errorf("code %q: ShouldRefetch() is false, so the memoised failure is never cleared", code)
		}
	}
}

func TestThrottleCodesAreNotPermanent(t *testing.T) {
	for _, code := range []string{"429", "503", "read_pxy_timeout"} {
		e := ErrorCodeToLinkError(code)
		if e.IsPermanent() {
			t.Errorf("code %q classified as permanent", code)
		}
		if !e.ShouldRefetch() && !e.ShouldRetry() {
			t.Errorf("code %q is neither refetchable nor retryable", code)
		}
	}
}

func TestGenuinelyPermanentCodesStayPermanent(t *testing.T) {
	for _, code := range []string{"401", "unauthorized", "404", "link_not_found", "file_not_available"} {
		e := ErrorCodeToLinkError(code)
		if !e.IsPermanent() {
			t.Errorf("code %q should remain permanent", code)
		}
		if e.ShouldRefetch() {
			t.Errorf("code %q should not trigger a refetch", code)
		}
	}
}
