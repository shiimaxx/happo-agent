package command

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
)

func TestCmdIsAddedFound(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusFound)
			}))
	defer ts.Close()

	mockCLI := buildBasicContext("is_added", ts.URL, "TEST", "192.168.0.1")
	err := CmdIsAdded(mockCLI)
	assert.Nil(t, err)
}

func TestCmdIsAddedNotFound(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
	defer ts.Close()

	mockCLI := buildBasicContext("is_added", ts.URL, "TEST", "192.168.0.1")
	err := CmdIsAdded(mockCLI)
	assert.NotNil(t, err)
	assert.EqualError(t, err, "Not found.")
}

func TestCmdIsAddedError(t *testing.T) {
	statuses := []int{
		http.StatusBadRequest,
		http.StatusInternalServerError,
	}

	for _, status := range statuses {
		t.Run("status_"+strconv.Itoa(status), func(t *testing.T) {
			ts := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(status)
						w.Write([]byte("dummy message"))
					}))
			defer ts.Close()

			mockCLI := buildBasicContext("is_added", ts.URL, "TEST", "192.168.0.1")
			err := CmdIsAdded(mockCLI).(*cli.ExitError)
			assert.NotNil(t, err)
			assert.EqualError(t, err, fmt.Sprintf("Failed! [%d] dummy message", status))
			assert.Equal(t, 2, err.ExitCode())
		})
	}
}

func TestCmdIsAddedValidation(t *testing.T) {
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
			mockCLI := buildBasicContext("is_added", dummyEndpoint, c.group, c.ip)
			err := CmdIsAdded(mockCLI).(*cli.ExitError)
			assert.NotNil(t, err)
			assert.Equal(t, 1, err.ExitCode())
			assert.EqualError(t, err, c.expectedMsg)
		})
	}
}
