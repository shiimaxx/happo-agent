package command

import (
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
			expected:      `Failed! [500] {"status":"NG","message":"some error has occured!!"}` + "\n",
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

			mockCLI := buildBasicContext("remove", ts.URL, dummyGroupName, dummyIP)
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

func TestCmdRemoveValidation(t *testing.T) {
	cases := []struct {
		caseName    string
		group       string
		ip          string
		expectedMsg string
	}{
		{
			caseName:    "blank group",
			group:       "",
			ip:          "192.168.0.1",
			expectedMsg: "group_name is null",
		},
		{
			caseName:    "blank ip",
			group:       "TEST",
			ip:          "",
			expectedMsg: "ip is null",
		},
	}

	dummyEndpoint := "http://localhost:6776"
	for _, c := range cases {
		t.Run(c.caseName, func(t *testing.T) {
			mockCLI := buildBasicContext("remove", dummyEndpoint, c.group, c.ip)
			err := CmdRemove(mockCLI).(*cli.ExitError)
			assert.NotNil(t, err)
			assert.Equal(t, 1, err.ExitCode())
			assert.EqualError(t, err, c.expectedMsg)
		})
	}
}
