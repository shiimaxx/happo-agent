package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"

	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/stretchr/testify/assert"
)

func TestAutoScalingHealth(t *testing.T) {

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Get("/autoscaling/health/:alias", AutoScalingHealth)

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "OK")
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	urlPrefix := "/autoscaling/health"
	var cases = []struct {
		url  string
		want string
	}{
		{
			url:  fmt.Sprintf("%s/dummy-prod-ag-dummy-prod-app-1?port=%d", urlPrefix, port),
			want: "OK",
		},
		{
			url:  fmt.Sprintf("%s/dummy-prod-ag-dummy-prod-app-2", urlPrefix),
			want: "OK",
		},
		{
			url:  fmt.Sprintf("%s/dummy-prod-ag-dummy-prod-app-1?port=9999", urlPrefix),
			want: "error",
		},
		{
			url:  fmt.Sprintf("%s/dummy-prod-ag-dummy-prod-app-1?port=test", urlPrefix),
			want: "error",
		},
		{
			url:  fmt.Sprintf("%s/missing-ag-app-1", urlPrefix),
			want: "error",
		},
	}

	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			req, _ := http.NewRequest("GET", c.url, nil)
			res := httptest.NewRecorder()

			m.ServeHTTP(res, req)

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}

			var r halib.AutoScalingHealthResponse
			if err := json.Unmarshal(data, &r); err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, c.want, r.Status)
		})
	}
}
