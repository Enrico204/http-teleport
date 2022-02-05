package telescope

import (
	"net/http"
)

func (t *Telescope) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.embeddedUI && r.RequestURI == "/_telescope/" {
		t.ServeWebDashboard(w, r)
		return
	}

	r.Host = t.target.Host
	t.proxy.ServeHTTP(w, r)
}
