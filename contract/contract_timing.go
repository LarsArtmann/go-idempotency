package contract

import "time"

// timings holds the durations the TTL-based subtests sleep on. Defaults are
// intentionally short to keep the suite fast while leaving enough margin for
// scheduler jitter; TimingScale stretches them on slow or busy machines.
type timings struct {
	shortTTL  time.Duration // expires quickly; used to test expiry
	mediumTTL time.Duration // slightly longer than shortTTL
	waitAfter time.Duration // sleep past shortTTL/mediumTTL
	longTTL   time.Duration // does not expire during the test
}

func defaultTimings() timings {
	//nolint:mnd // the tuned suite timings; named by their struct fields
	return timings{
		shortTTL:  20 * time.Millisecond,
		mediumTTL: 30 * time.Millisecond,
		waitAfter: 50 * time.Millisecond,
		longTTL:   time.Minute,
	}
}

func scaledTimings(scale float64) timings {
	tm := defaultTimings()

	if scale <= 0 {
		return tm
	}

	stretch := func(d time.Duration) time.Duration {
		return time.Duration(float64(d) * scale)
	}

	return timings{
		shortTTL:  stretch(tm.shortTTL),
		mediumTTL: stretch(tm.mediumTTL),
		waitAfter: stretch(tm.waitAfter),
		longTTL:   stretch(tm.longTTL),
	}
}
