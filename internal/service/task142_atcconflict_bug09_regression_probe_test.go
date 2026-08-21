package service

import (
	"context"
	"testing"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/store"
)

func TestBug09_HandoffMustStartWithCurrentController(t *testing.T) {
	st,err:=store.Open(":memory:"); if err!=nil {t.Fatal(err)}; defer st.Close(); s:=New(st,nil)
	if err:=st.PutFlight(context.Background(),&model.FlightPlan{ID:"p",Callsign:"OWN1",State:model.FlightActive,ControllingSector:"WEST"});err!=nil {t.Fatal(err)}
	if _,err:=s.InitiateHandoff(context.Background(),"p","EAST","NORTH");err==nil {t.Fatal("non-controlling sector was allowed to initiate handoff")}
}
