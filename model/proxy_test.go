package model

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/codegangsta/martini-contrib/render"
	"github.com/go-martini/martini"
	"github.com/heartbeatsjp/happo-agent/autoscaling"
	"github.com/heartbeatsjp/happo-agent/db"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/martini-contrib/binding"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
	leveldbUtil "github.com/syndtr/goleveldb/leveldb/util"
)

func setup() {
	//Mock
	DB, err := leveldb.Open(storage.NewMemStorage(), nil)
	if err != nil {
		os.Exit(1)
	}
	db.DB = DB

	saveInstanceData := func(alias, instanceID, ip string) {
		var instanceData halib.InstanceData
		instanceData.InstanceID = instanceID
		instanceData.IP = ip
		instanceData.MetricConfig = halib.MetricConfig{}

		var b bytes.Buffer
		enc := gob.NewEncoder(&b)
		enc.Encode(instanceData)

		db.DB.Put([]byte(fmt.Sprintf("ag-%s", alias)), b.Bytes(), nil)
	}

	saveInstanceData("dummy-prod-ag-dummy-prod-app-1", "i-aaaaaa", "127.0.0.1")
	saveInstanceData("dummy-prod-ag-dummy-prod-app-2", "", "")
	saveInstanceData("dummy-stg-ag-dummy-stg-app-1", "", "")
	saveInstanceData("dummy-stg-ag-dummy-stg-app-2", "", "")
}

func teardown() {
	iter := db.DB.NewIterator(
		leveldbUtil.BytesPrefix(
			[]byte("ag-"),
		),
		nil,
	)
	for iter.Next() {
		key := iter.Key()
		db.DB.Delete(key, nil)
	}
	iter.Release()
	db.DB.Close()
}

func TestPostToAgent1(t *testing.T) {
	const stubResponse = "OK"

	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, stubResponse)
			}))
	defer ts.Close()
	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	jsonData := []byte("{}")
	statusCode, response, err := postToAgent(host, port, "test", jsonData)
	assert.EqualValues(t, http.StatusOK, statusCode)
	assert.Contains(t, response, stubResponse)
	assert.Nil(t, err)
}

func TestPostToAgent2(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(10 * time.Millisecond)
				fmt.Fprintln(w, "will ignore(return will be blank)")
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	timeout := _httpClient.Timeout
	_httpClient.Timeout = 1 * time.Millisecond
	statusCode, response, err := postToAgent(host, port, "test", []byte("{}"))
	_httpClient.Timeout = timeout

	assert.EqualValues(t, http.StatusGatewayTimeout, statusCode)
	assert.Contains(t, response, "")
	assert.True(t, err.(net.Error).Timeout())
}

func TestPostToAgent3(t *testing.T) {
	/*
		// FIXME cannot test err != nil and err is NOT timeout.
		ts := httptest.NewTLSServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTemporaryRedirect)
					w.Header().Set("Location", "the/broken:location:header/")
					fmt.Fprintln(w, "will ignore(return will be blank)")
				}))
		defer ts.Close()

		re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
		found := re.FindStringSubmatch(ts.URL)
		host := found[2]
		port, _ := strconv.Atoi(found[3])
		status_code, response, err := postToAgent(host, port, "test", []byte("{}"))

		assert.EqualValues(t, status_code, http.StatusBadGateway)
		assert.Contains(t, response, "")
		assert.NotNil(t, err)
	*/
}

func TestPostToAgent4(t *testing.T) {
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				fmt.Fprint(w, "error response")
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])
	statusCode, response, err := postToAgent(host, port, "test", []byte("{}"))

	assert.EqualValues(t, http.StatusServiceUnavailable, statusCode)
	assert.Contains(t, response, "error response")
	assert.Nil(t, err)
}

func TestProxy1(t *testing.T) {
	//monitor ok

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				body, err := ioutil.ReadAll(r.Body)
				if err != nil {
					assert.FailNow(t, err.Error())
				}
				expected := `{"apikey": "", "plugin_name": "monitor_test_plugin", "plugin_option": "0"}`
				assert.Equal(t, string(body), expected)
				fmt.Fprint(w, `{"return_value":0,"message":"ok"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	var content string
	rawContent := `{"apikey": "", "plugin_name": "monitor_test_plugin", "plugin_option": "0"}`
	content = base64.StdEncoding.EncodeToString([]byte(rawContent))

	// This line lets the test fail because `/proxy` only accepts base64 encoded string as request_json
	// content = `{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}`

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"%s"
	}`, host, port, content)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	lastRunned = time.Now().Unix() //avoid saveMachineState
	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":0,"message":"ok"}`,
		res.Body.String(),
	)
}

func TestProxy2(t *testing.T) {
	//gateway timeout

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(1 * time.Second)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"xxx\", \"plugin_option\": \"\"}"
	}`, host, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	timeout := _httpClient.Timeout
	_httpClient.Timeout = 1 * time.Millisecond

	lastRunned = time.Now().Unix() //avoid saveMachineState
	m.ServeHTTP(res, req)

	_httpClient.Timeout = timeout

	assert.Equal(t, http.StatusGatewayTimeout, res.Code)
	assert.Regexp(t,
		regexp.MustCompile(
			fmt.Sprintf(`"return_value":3,"message":"Post https://%s:%d/monitor: net/http: request canceled .*(Client.Timeout exceeded while awaiting headers)`, host, port)),
		res.Body.String(),
	)
}

func TestProxy3(t *testing.T) {
	//monitor ok (multi proxy)

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d","127.0.0.1:6777"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}"
	}`, host, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	lastRunned = time.Now().Unix() //avoid saveMachineState
	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":0,"message":"ok"}`,
		res.Body.String(),
	)
}

func TestProxy4(t *testing.T) {
	//gateway timeout(multi proxy)

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(1 * time.Second)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d","127.0.0.1:6777"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"xxx\", \"plugin_option\": \"\"}"
	}`, host, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	timeout := _httpClient.Timeout
	_httpClient.Timeout = 1 * time.Millisecond

	lastRunned = time.Now().Unix() //avoid saveMachineState
	m.ServeHTTP(res, req)

	_httpClient.Timeout = timeout

	assert.Equal(t, http.StatusGatewayTimeout, res.Code)
	assert.Equal(t,
		fmt.Sprintf(`{"return_value":3,"message":"Post https://%s:%d/proxy: net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)"}`, host, port),
		res.Body.String(),
	)
}

func TestProxy5(t *testing.T) {
	//proxy monitor

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok\n"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-1"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":0,"message":"ok\nAutoScaling Group Name: dummy-prod-ag\nAutoScaling Instance PrivateIP: 127.0.0.1\n"}`,
		res.Body.String(),
	)
}

func TestProxy6(t *testing.T) {
	//proxy monitor when alias not assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok\n"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-2"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":0,"message":"dummy-prod-ag-dummy-prod-app-2 has not been assigned instance\n"}`,
		res.Body.String(),
	)
}

func TestProxy7(t *testing.T) {
	//proxy monitor when alias not found

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok\n"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-99"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":3,"message":"alias not found: dummy-prod-ag-dummy-prod-app-99\n"}`,
		res.Body.String(),
	)
}

func TestProxy8(t *testing.T) {
	//monitor ok when autoscaling config is not found

	AutoScalingConfigFile = "./not_found"
	defer func() { AutoScalingConfigFile = "../autoscaling/testdata/autoscaling_test_multi.yaml" }()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	host := found[2]
	port, _ := strconv.Atoi(found[3])

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "monitor",
		"request_json":
			"{\"apikey\": \"\", \"plugin_name\": \"monitor_test_plugin\", \"plugin_option\": \"0\"}"
	}`, host, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	lastRunned = time.Now().Unix() //avoid saveMachineState
	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_value":0,"message":"ok"}`,
		res.Body.String(),
	)
}

func TestProxy9(t *testing.T) {
	//proxy monitor when request is not contain request_type

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_value":0,"message":"ok\n"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-1"

	requestJSON := fmt.Sprintf(`{"proxy_hostport": ["%s:%d"]}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
	assert.Equal(t, "request_type unsupported", res.Body.String())
}

func TestProxy10(t *testing.T) {
	//proxy metric when alias assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"metric_data":null,"message":""}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-1"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "metric",
		"request_json": "{\"apikey\": \"\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"metric_data":null,"message":""}`, res.Body.String())
}

func TestProxy11(t *testing.T) {
	//proxy metric when alias not assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"metric_data":null,"message":""}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-2"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "metric",
		"request_json": "{\"apikey\": \"\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
	assert.Equal(t, "dummy-prod-ag-dummy-prod-app-2 has not been assigned instance\n", res.Body.String())
}

func TestProxy12(t *testing.T) {
	//proxy metric when alias not found

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"metric_data":null,"message":""}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-99"

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "metric",
		"request_json": "{\"apikey\": \"\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "alias not found: dummy-prod-ag-dummy-prod-app-99\n", res.Body.String())
}

func TestProxy13(t *testing.T) {
	//proxy metric config update when alias assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"status":"OK","message":""}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-1"

	var request halib.MetricConfigUpdateRequest
	request.APIKey = ""
	request.Config.Metrics = append(request.Config.Metrics, struct {
		Hostname string `yaml:"hostname" json:"Hostname"`
		Plugins  []struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		} `yaml:"plugins" json:"Plugins"`
	}{
		"dummy-prod-ag",
		[]struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		}{
			{PluginName: "metric_test_plugin", PluginOption: "0"},
		},
	})
	b, _ := json.Marshal(request)

	var proxyRequest halib.ProxyRequest
	proxyRequest.ProxyHostPort = append(proxyRequest.ProxyHostPort, fmt.Sprintf("%s:%d", "dummy-prod-ag", port))
	proxyRequest.RequestType = "metric/config/update"
	proxyRequest.RequestJSON = b
	requestJSON, _ := json.Marshal(proxyRequest)

	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"status":"OK","message":""}`, res.Body.String())

	v, _ := db.DB.Get([]byte(fmt.Sprintf("ag-%s", alias)), nil)
	var instanceData halib.InstanceData
	dec := gob.NewDecoder(bytes.NewReader(v))
	dec.Decode(&instanceData)
	assert.Equal(t, "dummy-prod-ag-dummy-prod-app-1", instanceData.MetricConfig.Metrics[0].Hostname)
	assert.Equal(t, "metric_test_plugin", instanceData.MetricConfig.Metrics[0].Plugins[0].PluginName)
	assert.Equal(t, "0", instanceData.MetricConfig.Metrics[0].Plugins[0].PluginOption)
}

func TestProxy14(t *testing.T) {
	//proxy metric config update when alias not assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"status":"OK","message":""}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-Za-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	alias := "dummy-prod-ag-dummy-prod-app-2"

	var request halib.MetricConfigUpdateRequest
	request.APIKey = ""
	request.Config.Metrics = append(request.Config.Metrics, struct {
		Hostname string `yaml:"hostname" json:"Hostname"`
		Plugins  []struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		} `yaml:"plugins" json:"Plugins"`
	}{
		"dummy-prod-ag",
		[]struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		}{
			{PluginName: "metric_test_plugin", PluginOption: "0"},
		},
	})
	b, _ := json.Marshal(request)

	var proxyRequest halib.ProxyRequest
	proxyRequest.ProxyHostPort = append(proxyRequest.ProxyHostPort, fmt.Sprintf("%s:%d", "dummy-prod-ag", port))
	proxyRequest.RequestType = "metric/config/update"
	proxyRequest.RequestJSON = b
	requestJSON, _ := json.Marshal(proxyRequest)

	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t, `{"status":"OK","message":""}`, res.Body.String())

	v, _ := db.DB.Get([]byte(fmt.Sprintf("ag-%s", alias)), nil)
	var instanceData halib.InstanceData
	dec := gob.NewDecoder(bytes.NewReader(v))
	dec.Decode(&instanceData)
	assert.Equal(t, "dummy-prod-ag-dummy-prod-app-2", instanceData.MetricConfig.Metrics[0].Hostname)
	assert.Equal(t, "metric_test_plugin", instanceData.MetricConfig.Metrics[0].Plugins[0].PluginName)
	assert.Equal(t, "0", instanceData.MetricConfig.Metrics[0].Plugins[0].PluginOption)
}

func TestProxy15(t *testing.T) {
	//proxy metric config update when autoscaling group not found

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	alias := "dummy-prod-ag-dummy-prod-app-1"
	port := 6777

	var request halib.MetricConfigUpdateRequest
	request.APIKey = ""
	request.Config.Metrics = append(request.Config.Metrics, struct {
		Hostname string `yaml:"hostname" json:"Hostname"`
		Plugins  []struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		} `yaml:"plugins" json:"Plugins"`
	}{
		"dummy-prod-ag-dummy-prod-app-1",
		[]struct {
			PluginName   string `yaml:"plugin_name" json:"Plugin_Name"`
			PluginOption string `yaml:"plugin_option" json:"Plugin_Option"`
		}{
			{PluginName: "metric_test_plugin", PluginOption: "0"},
		},
	})
	b, _ := json.Marshal(request)

	var proxyRequest halib.ProxyRequest
	proxyRequest.ProxyHostPort = append(proxyRequest.ProxyHostPort, fmt.Sprintf("%s:%d", alias, port))
	proxyRequest.RequestType = "metric/config/update"
	proxyRequest.RequestJSON = b
	requestJSON, _ := json.Marshal(proxyRequest)

	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, `{"status":"NG","message":"can't find autoscaling group: dummy-prod-ag-dummy-prod-app-1"}`, res.Body.String())
}

func TestProxy16(t *testing.T) {
	//proxy inventory when alias assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	//edge
	ts := httptest.NewTLSServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"return_code":0, "return_value":"dummy-prod-ag-dummy-prod-app-1\n"}`)
			}))
	defer ts.Close()

	re, _ := regexp.Compile("([a-z]+)://([A-nZa-z0-9.]+):([0-9]+)(.*)")
	found := re.FindStringSubmatch(ts.URL)
	port, _ := strconv.Atoi(found[3])

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "inventory",
		"request_json":
			"{\"apikey\": \"\", \"command\": \"uname\", \"command_option\": \"-n\"}"
	}`, "dummy-prod-ag", port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
	assert.Equal(t,
		`{"return_code":0, "return_value":"dummy-prod-ag-dummy-prod-app-1\n"}`,
		res.Body.String(),
	)
}

func TestProxy17(t *testing.T) {
	//proxy inventory when alias not assigned instance

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "inventory",
		"request_json":
			"{\"apikey\": \"\", \"command\": \"uname\", \"command_option\": \"-n\"}"
	}`, "dummy-stg-ag", 6777)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusServiceUnavailable, res.Code)
	assert.Equal(t, "", res.Body.String())
}

func TestProxy18(t *testing.T) {
	//proxy inventory when autoscaling group not found

	setup()
	defer teardown()

	//bastion
	m := martini.Classic()
	m.Use(render.Renderer())
	m.Post("/proxy", binding.Json(halib.ProxyRequest{}), Proxy)
	m.Map(&autoscaling.AWSClient{})

	alias := "dummy-prod-ag-dummy-prod-app-1"
	port := 6777

	requestJSON := fmt.Sprintf(`{
		"proxy_hostport": ["%s:%d"],
		"request_type": "inventory",
		"request_json":
			"{\"apikey\": \"\", \"command\": \"uname\", \"command_option\": \"-n\"}"
	}`, alias, port)
	reader := bytes.NewReader([]byte(requestJSON))

	req, _ := http.NewRequest("POST", "/proxy", reader)
	req.Header.Set("Content-Type", "application/json")

	res := httptest.NewRecorder()

	m.ServeHTTP(res, req)

	assert.Equal(t, http.StatusNotFound, res.Code)
	assert.Equal(t, "", res.Body.String())
}

func TestMain(m *testing.M) {
	AutoScalingConfigFile = "../autoscaling/testdata/autoscaling_test_multi.yaml"
	os.Exit(m.Run())
}
