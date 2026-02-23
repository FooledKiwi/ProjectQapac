package service

import (
	"context"
	"time"
)

// ETAProvider computes the estimated time of arrival (in seconds) of a bus to
// the given transit stop.  The source string identifies the calculation
// strategy used (e.g. "simple", "gps") and is intended for telemetry.
//
// Planned implementations:
//
//   - SimpleETAProvider  (MVP v1) — deterministic simulation based on
//     time-of-day and stop ID; no external dependencies.
//
//   - GPSETAProvider     (MVP v2) — reads real vehicle positions from the
//     vehicle_positions table, routes from vehicle to stop via RoutingService,
//     and returns the minimum duration across all active vehicles on routes
//     that serve the stop.  Requires table:
//
//     CREATE TABLE vehicle_positions (
//     id          SERIAL PRIMARY KEY,
//     route_id    INT NOT NULL REFERENCES routes(id),
//     geom        GEOMETRY(POINT, 4326) NOT NULL,
//     reported_at TIMESTAMP NOT NULL DEFAULT NOW()
//     );
//
//     If no vehicle has reported a position within the last 5 minutes,
//     GPSETAProvider returns ErrNoVehicleData so that ETAService can invoke
//     its configured fallback provider.
type ETAProvider interface {
	GetETA(ctx context.Context, stopID int32) (seconds int, source string, err error)
}

// ErrNoVehicleData is returned by a GPS-based ETAProvider when there is no
// recent position data available for any vehicle serving the requested stop.
// ETAService treats this as a signal to invoke the fallback provider.
var ErrNoVehicleData = errNoVehicleData("no recent vehicle position data")

type errNoVehicleData string

func (e errNoVehicleData) Error() string { return string(e) }

// SimpleETAProvider simulates the ETA of a bus arriving at a transit stop.
// It uses two inputs available without any external data source:
//
//  1. Time of day — peak hours (07-09, 17-19 by default) yield a higher base
//     ETA because buses move slower in heavy traffic.
//  2. Stop ID — used as a deterministic offset (stopID % 60) so that
//     different stops return slightly different ETAs, simulating vehicles at
//     various points along their routes.
//
// This provider is the MVP v1 placeholder.  When real GPS data becomes
// available, replace it with GPSETAProvider; ETAService and all call sites
// remain unchanged because both satisfy ETAProvider.
type SimpleETAProvider struct {
	// baseNormalS is the ETA base (seconds) during off-peak hours. Default: 180s.
	baseNormalS int

	// basePeakS is the ETA base (seconds) during peak hours. Default: 360s.
	basePeakS int

	// peakHours holds the local hours (0-23) considered peak traffic.
	// Default: {7, 8, 17, 18}.
	peakHours map[int]struct{}

	// now is a clock function; overridable in tests.
	now func() time.Time
}

// SimpleETAOption configures a SimpleETAProvider.
type SimpleETAOption func(*SimpleETAProvider)

// WithBaseETAs overrides the off-peak and peak base ETA values (in seconds).
func WithBaseETAs(normalS, peakS int) SimpleETAOption {
	return func(p *SimpleETAProvider) {
		p.baseNormalS = normalS
		p.basePeakS = peakS
	}
}

// WithPeakHours overrides which local hours (0-23) are considered peak.
func WithPeakHours(hours ...int) SimpleETAOption {
	return func(p *SimpleETAProvider) {
		m := make(map[int]struct{}, len(hours))
		for _, h := range hours {
			m[h] = struct{}{}
		}
		p.peakHours = m
	}
}

// withClock injects a fake clock for unit testing.
func withClock(fn func() time.Time) SimpleETAOption {
	return func(p *SimpleETAProvider) { p.now = fn }
}

// NewSimpleETAProvider creates a SimpleETAProvider with sensible defaults.
func NewSimpleETAProvider(opts ...SimpleETAOption) *SimpleETAProvider {
	p := &SimpleETAProvider{
		baseNormalS: 180,
		basePeakS:   360,
		peakHours:   map[int]struct{}{7: {}, 8: {}, 17: {}, 18: {}},
		now:         time.Now,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

// GetETA implements ETAProvider.
//
// Returns a deterministic ETA simulating bus arrival time at the stop:
//   - Base seconds depend on whether the current hour is peak or off-peak.
//   - A per-stop offset of (stopID % 60) is added to simulate vehicles at
//     different positions along their routes.
//   - Source is always "simple".
func (p *SimpleETAProvider) GetETA(_ context.Context, stopID int32) (seconds int, source string, err error) {
	hour := p.now().Local().Hour()

	base := p.baseNormalS
	if _, isPeak := p.peakHours[hour]; isPeak {
		base = p.basePeakS
	}

	// Per-stop deterministic offset: spreads ETAs across [0, 59] extra seconds,
	// simulating buses at different distances from each stop.
	offset := int(stopID % 60)

	return base + offset, "simple", nil
}
