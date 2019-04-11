package command

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"flag"

	"github.com/codegangsta/cli"
	"github.com/stretchr/testify/assert"
)

func TestCmdAdd(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, `{"stauts":"OK","message":""}`)
			}))
	defer ts.Close()
	var cases = []struct {
		name         string
		endpoint     string
		group        string
		ip           string
		expected     string
		isNormalTest bool
	}{
		{
			name:         "happo-agent add --endpoint <endpoint url> -group_name HB_TEST -ip 192.0.2.1",
			endpoint:     ts.URL,
			group:        "HB_TEST",
			ip:           "192.0.2.1",
			expected:     "",
			isNormalTest: true,
		},
		{
			name:         "happo-agent add --endpoint <endpoint url> -ip 192.0.2.1",
			endpoint:     ts.URL,
			group:        "",
			ip:           "192.0.2.1",
			expected:     "group_name is null",
			isNormalTest: false,
		},
		{
			name:         "happo-agent add --endpoint <endpoint url> -group_name HB_TEST",
			endpoint:     ts.URL,
			group:        "HB_TEST",
			ip:           "",
			expected:     "ip is null",
			isNormalTest: false,
		},
		{
			name:         "happo-agent add -group_name HB_TEST -ip 192.0.2.1",
			endpoint:     "",
			group:        "HB_TEST",
			ip:           "192.0.2.1",
			expected:     "ERROR: endpoint must set with args or environment variable",
			isNormalTest: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := cli.NewApp()
			set := flag.NewFlagSet("test", 0)
			set.String("endpoint", c.endpoint, "")
			set.String("group_name", c.group, "")
			set.String("ip", c.ip, "")
			mockCLI := cli.NewContext(app, set, nil)
			mockCLI.Command.Name = "add"

			if err := CmdAdd(mockCLI); err != nil {
				if c.isNormalTest {
					assert.Nil(t, err)
				} else {
					assert.NotNil(t, err)
				}
			}
		})
	}
}

func TestCmdAdd_AutoScaling(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, `{"stauts":"OK","message":""}`)
			}))
	defer ts.Close()
	var cases = []struct {
		name                 string
		endpoint             string
		group                string
		proxy                cli.StringSlice
		autoScalingGroupName string
		autoScalingCount     int
		hostPrefix           string
		expected             string
		isNormalTest         bool
	}{
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --group_name HB_TEST --proxy 192.0.2.1 \
							--autoscaling_group_name hb-autoscaling --autoscaling_count 10 --host_prefix app`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     10,
			hostPrefix:           "app",
			expected:             "",
			isNormalTest:         true,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --proxy 192.0.2.1 \
							--autoscaling_group_name hb-autoscaling --autoscaling_count 10 --host_prefix app`,
			endpoint:             ts.URL,
			group:                "",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     10,
			hostPrefix:           "app",
			expected:             "group_name is null",
			isNormalTest:         false,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url>  --group_name HB_TEST \
							--autoscaling_group_name hb-autoscaling --autoscaling_count 10 --host_prefix app`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     10,
			hostPrefix:           "app",
			expected:             "proxy is null",
			isNormalTest:         false,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --group_name HB_TEST --proxy 192.0.2.1 \
							--autoscaling_count 10 --host_prefix app`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "",
			autoScalingCount:     10,
			hostPrefix:           "app",
			expected:             "autoscaling_group_name is null",
			isNormalTest:         false,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --group_name HB_TEST --proxy 192.0.2.1 \
							--autoscaling_group_name hb-autoscaling --host_prefix app`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     0,
			hostPrefix:           "app",
			expected:             "autoscaling_count is lower than 1",
			isNormalTest:         false,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --group_name HB_TEST --proxy 192.0.2.1 \
							--autoscaling_group_name hb-autoscaling --autoscaling_count -5 --host_prefix app`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     -5,
			hostPrefix:           "app",
			expected:             "autoscaling_count is lower than 1",
			isNormalTest:         false,
		},
		{
			name: `happo-agent add_ag --endpoint <endpoint url> --group_name HB_TEST --proxy 192.0.2.1 \
							--autoscaling_group_name hb-autoscaling --autoscaling_count 10`,
			endpoint:             ts.URL,
			group:                "HB_TEST",
			proxy:                cli.StringSlice{"192.0.2.1"},
			autoScalingGroupName: "hb-autoscaling",
			autoScalingCount:     10,
			hostPrefix:           "",
			expected:             "host_prefix is null",
			isNormalTest:         false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := cli.NewApp()
			set := flag.NewFlagSet("test", 0)
			set.String("endpoint", c.endpoint, "")
			set.String("group_name", c.group, "")
			set.Var(&c.proxy, "proxy", "")
			set.String("autoscaling_group_name", c.autoScalingGroupName, "")
			set.Int("autoscaling_count", c.autoScalingCount, "")
			set.String("host_prefix", c.hostPrefix, "")
			mockCLI := cli.NewContext(app, set, nil)
			mockCLI.Command.Name = "add_ag"

			err := CmdAdd(mockCLI)
			if c.isNormalTest {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
				assert.Equal(t, c.expected, err.Error())
			}
		})
	}
}

func TestCmdAddError(t *testing.T) {
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

			app := cli.NewApp()
			set := flag.NewFlagSet("test", 0)
			set.String("endpoint", ts.URL, "")
			set.String("group_name", "TEST", "")
			set.String("ip", "192.168.0.1", "")
			mockCLI := cli.NewContext(app, set, nil)
			mockCLI.Command.Name = "add"

			err := CmdAdd(mockCLI)
			assert.NotNil(t, err)
			assert.EqualError(t, err, fmt.Sprintf("Failed! [%d] dummy message", status))
		})
	}
}
