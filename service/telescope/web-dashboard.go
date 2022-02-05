package telescope

import (
	_ "embed"
	"gitlab.com/enrico204/http-telescope/service/telescope/transport"
	"html/template"
	"math"
	"net/http"
	"time"
)

//go:embed "templates/dashboard.gohtml"
var dashboardTplFile []byte

var dashboardTpl = template.Must(template.New("dashboard").Parse(string(dashboardTplFile)))

func (t *Telescope) ServeWebDashboard(w http.ResponseWriter, _ *http.Request) {
	type httpTrackTpl struct {
		*transport.HTTPTrack

		StatusCodeGroup    int
		SecondsAgo         int
		Duration           time.Duration
		RequestString      string
		RequestBodyString  string
		ResponseString     string
		ResponseBodyString string
		InFlight           bool
		Class              string
	}

	var tracks = t.transport.GetTracks()
	var items = make([]httpTrackTpl, 0, len(tracks))
	for idx, v := range tracks {
		var group = 0
		var responseString = ""
		var responseBodyString = ""
		var class = ""
		if v.Response != nil {
			group = v.Response.StatusCode / 100
			responseString = v.Response.String()
			responseBodyString = string(v.Response.Body)
		}
		if idx%2 == 0 {
			class = "odd"
		}
		items = append(items, httpTrackTpl{
			HTTPTrack:          v,
			StatusCodeGroup:    group,
			SecondsAgo:         int(time.Since(v.When).Seconds()),
			Duration:           time.Duration(math.Round(float64(v.Duration()/time.Millisecond))) * time.Millisecond,
			RequestString:      v.String(),
			RequestBodyString:  string(v.Body),
			ResponseString:     responseString,
			ResponseBodyString: responseBodyString,
			Class:              class,
		})
	}

	err := dashboardTpl.Execute(w, struct {
		Tracks []httpTrackTpl
	}{items})
	if err != nil {
		panic(err)
	}
}
