package flow

import "testing"

func TestBug08_MITNormalizesUnorderedRadarSnapshot(t *testing.T) {
	v := MilesInTrailCheck([]SpacingEntry{{PlanID:"A",DistanceToFix:120},{PlanID:"C",DistanceToFix:100},{PlanID:"B",DistanceToFix:110}}, 15)
	if len(v)!=2 || v[0].Leader!="A" || v[0].Trailer!="B" || v[1].Leader!="B" || v[1].Trailer!="C" { t.Fatalf("unordered snapshot violations = %+v, want A/B and B/C",v) }
}
