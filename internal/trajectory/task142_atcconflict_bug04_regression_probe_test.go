package trajectory

import "testing"

func TestBug04_EastboundDatelineProjectionUsesSignedLongitude(t *testing.T) {
	p, _ := ProjectFromTrack(LatLon{Lat: 0, Lon: 179.9}, 90, 600, 350, 0, 3600)
	if p.Lon >= -170 || p.Lon <= -180 { t.Fatalf("dateline projection longitude = %v, want signed western longitude", p.Lon) }
}
