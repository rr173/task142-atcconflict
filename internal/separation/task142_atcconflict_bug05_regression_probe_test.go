package separation

import (
	"testing"
	"time"
	"task142-atcconflict/internal/trajectory"
)

func TestBug05_RVSMMinimumFollowsProjectedLevels(t *testing.T) {
	p := &Prober{LookaheadSec: 60, StepSec: 60}
	tracks := []TrackState{{PlanID:"A", Position:trajectory.LatLon{}, FL:410}, {PlanID:"B", Position:trajectory.LatLon{}, FL:400, VerticalRate:2000}}
	if got := p.Detect(time.Date(2026,1,1,0,0,0,0,time.UTC), tracks); len(got) != 1 { t.Fatalf("projected non-RVSM pair conflicts = %d, want 1", len(got)) }
}
