package service

import (
	"context"
	"testing"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/store"
)

func TestBug10_TerminatedFlightLeavesConflictPicture(t *testing.T) {
	st,err:=store.Open(":memory:");if err!=nil {t.Fatal(err)};defer st.Close();s:=New(st,nil);ctx:=context.Background()
	for _,id:=range []string{"a","b"} { if err:=st.PutFlight(ctx,&model.FlightPlan{ID:id,Callsign:id,State:model.FlightActive});err!=nil {t.Fatal(err)} }
	if _,err:=s.Report(ctx,&model.TrackReport{PlanID:"a",Ts:"2026-01-01T00:00:00Z",FL:350,Heading:90,Groundspeed:480});err!=nil {t.Fatal(err)}
	if _,err:=s.Report(ctx,&model.TrackReport{PlanID:"b",Ts:"2026-01-01T00:00:00Z",FL:350,Heading:270,Groundspeed:480});err!=nil {t.Fatal(err)}
	if cs,_:=s.Conflicts(ctx);len(cs)!=1 {t.Fatalf("setup conflicts = %d",len(cs))}
	if _,err:=s.Move(ctx,"a",model.FlightTerminated);err!=nil {t.Fatal(err)}
	if cs,_:=s.Conflicts(ctx);len(cs)!=0 {t.Fatalf("terminated flight still leaves %d conflicts",len(cs))}
}
