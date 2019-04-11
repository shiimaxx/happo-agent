package command

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
)

func buildContext(command, endpoint, groupname, ip string) *cli.Context {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("endpoint", endpoint, "")
	set.String("group_name", groupname, "")
	set.String("ip", ip, "")
	mockCLI := cli.NewContext(app, set, nil)
	mockCLI.Command.Name = command
	return mockCLI
}

func TestCmdIsAddedFound(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusFound)
			}))
	defer ts.Close()

	mockCLI := buildContext("is_added", ts.URL, "TEST", "192.168.0.1")
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

	mockCLI := buildContext("is_added", ts.URL, "TEST", "192.168.0.1")
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

			mockCLI := buildContext("is_added", ts.URL, "TEST", "192.168.0.1")
			err := CmdIsAdded(mockCLI)
			assert.NotNil(t, err)
			assert.EqualError(t, err, fmt.Sprintf("Failed! [%d] dummy message", status))
		})
	}
}
