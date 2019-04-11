package command

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
)

func TestCmdRemove(t *testing.T) {
	var cases = []struct {
		name          string
		resStatusCode int
		resBody       string
		isNormalTest  bool
		expected      string
	}{
		{
			name:          "When server returns 200",
			resStatusCode: http.StatusOK,
			resBody:       `{"status":"OK","message":""}`,
			isNormalTest:  true,
			expected:      "",
		},
		{
			name:          "When server returns 404",
			resStatusCode: http.StatusNotFound,
			resBody:       `{"status":"OK","message":""}`,
			isNormalTest:  false,
			expected:      "Not found.",
		},
		{
			name:          "When server returns 500",
			resStatusCode: http.StatusInternalServerError,
			resBody:       `{"status":"NG","message":"some error has occured!!"}`,
			isNormalTest:  false,
			expected:      "Failed! [500] {\"status\":\"NG\",\"message\":\"some error has occured!!\"}\n",
		},
	}

	dummyGroupName := "HB_TEST"
	dummyIP := "192.0.2.1"
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(c.resStatusCode)
						fmt.Fprintln(w, c.resBody)
					}))
			defer ts.Close()

			app := cli.NewApp()
			set := flag.NewFlagSet("test", 0)
			set.String("endpoint", ts.URL, "")
			set.String("group_name", dummyGroupName, "")
			set.String("ip", dummyIP, "")
			mockCLI := cli.NewContext(app, set, nil)
			mockCLI.Command.Name = "remove"

			if err := CmdRemove(mockCLI); err != nil {
				if c.isNormalTest {
					assert.Nil(t, err)
				} else {
					assert.EqualError(t, err, c.expected)
				}
			}
		})
	}
}
