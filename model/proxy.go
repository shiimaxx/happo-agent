package model

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codegangsta/martini-contrib/render"
	"github.com/heartbeatsjp/happo-agent/autoscaling"
	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/heartbeatsjp/happo-agent/util"
	"github.com/syndtr/goleveldb/leveldb"
)

// --- Global Variables
var (
	// See http://golang.org/pkg/net/http/#Client
	tr          = http.DefaultTransport.(*http.Transport)
	_httpClient = http.DefaultClient

	refreshAutoScalingChan = make(chan struct {
		config halib.AutoScalingConfigData
		client autoscaling.AWSClient
	})
	refreshAutoScalingMutex           = sync.Mutex{}
	lastRefreshAutoScaling            = make(map[string]int64)
	refreshAutoScalingIntervalSeconds = int64(halib.DefaultRefreshAutoScalingIntervalSeconds)
)

func init() {
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	_httpClient.Transport = tr

	go func() {
		for {
			select {
			case a := <-refreshAutoScalingChan:
				go func(autoScalingGroupName, hostPrefix string, autoScalingCount int) {
					if isPermitRefreshAutoScaling(autoScalingGroupName) {
						if err := autoscaling.RefreshAutoScalingInstances(&a.client, autoScalingGroupName, hostPrefix, autoScalingCount); err != nil {
							log := util.HappoAgentLogger()
							log.Error(err.Error())
						}
					}
				}(a.config.AutoScalingGroupName, a.config.HostPrefix, a.config.AutoScalingCount)
			}
		}
	}()
}

// Proxy do http reqest to next happo-agent
func Proxy(proxyRequest halib.ProxyRequest, r render.Render, client *autoscaling.AWSClient) (int, string) {
	var nextHostport string
	var requestType string
	var requestJSON []byte
	var err error

	nextHostport = proxyRequest.ProxyHostPort[0]

	if len(proxyRequest.ProxyHostPort) == 1 {
		// last proxy
		requestType = proxyRequest.RequestType
		requestJSON = proxyRequest.RequestJSON
	} else {
		// more proxies
		proxyRequest.ProxyHostPort = proxyRequest.ProxyHostPort[1:]
		requestType = "proxy"
		requestJSON, _ = json.Marshal(proxyRequest) // ここではエラーは出ない(出るとしたら上位でずっこけている
	}
	nextHostdata := strings.Split(nextHostport, ":")
	nextHost := nextHostdata[0]
	nextPort := halib.DefaultAgentPort
	if len(nextHostdata) == 2 {
		nextPort, err = strconv.Atoi(nextHostdata[1])
		if err != nil {
			nextPort = halib.DefaultAgentPort
		}
	}

	config := getAutoScalingInfo(nextHost)

	var respCode int
	var response string
	if config.AutoScalingGroupName == "" {
		respCode, response, err = postToAgent(nextHost, nextPort, requestType, requestJSON)
		if err != nil {
			response = makeMonitorResponse(halib.MonitorUnknown, err.Error())
		}
	} else {
		respCode, response, err = postToAutoScalingAgent(nextHost, nextPort, requestType, requestJSON, config.AutoScalingGroupName)
		if err != nil {
			response = makeMonitorResponse(halib.MonitorUnknown, err.Error())
		}
		if requestType == "monitor" && respCode != http.StatusOK {
			refreshAutoScalingChan <- struct {
				config halib.AutoScalingConfigData
				client autoscaling.AWSClient
			}{
				config: config,
				client: *client,
			}
		}
	}

	return respCode, response
}

func getAutoScalingInfo(nextHost string) halib.AutoScalingConfigData {
	log := util.HappoAgentLogger()

	if _, err := os.Stat(AutoScalingConfigFile); os.IsNotExist(err) {
		log.Info(err)
		return halib.AutoScalingConfigData{}
	}

	var autoScalingConfigData halib.AutoScalingConfigData
	autoScalingList, err := autoscaling.GetAutoScalingConfig(AutoScalingConfigFile)
	if err != nil {
		log.Errorf("failed to get autoscaling config: %s", err.Error())
		return autoScalingConfigData
	}

	for _, a := range autoScalingList.AutoScalings {
		if strings.HasPrefix(nextHost, a.AutoScalingGroupName) {
			autoScalingConfigData.AutoScalingGroupName = a.AutoScalingGroupName
			autoScalingConfigData.HostPrefix = a.HostPrefix
			autoScalingConfigData.AutoScalingCount = a.AutoScalingCount
			return autoScalingConfigData
		}
	}
	return autoScalingConfigData
}

func postToAgent(host string, port int, requestType string, jsonData []byte) (int, string, error) {
	log := util.HappoAgentLogger()
	uri := fmt.Sprintf("https://%s:%d/%s", host, port, requestType)
	log.Printf("Proxy to: %s", uri)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := _httpClient.Do(req)
	if err != nil {
		if errTimeout, ok := err.(net.Error); ok && errTimeout.Timeout() {
			return http.StatusGatewayTimeout, "", errTimeout
		}
		if resp != nil {
			if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == 0 {
				return http.StatusServiceUnavailable, "", err
			}
			return resp.StatusCode, "", err
		}
		return http.StatusInternalServerError, "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return http.StatusInternalServerError, "", err
	}
	return resp.StatusCode, string(body[:]), nil
}

func makeMonitorResponse(returnValue int, message string) string {
	var monitorResponse halib.MonitorResponse
	monitorResponse.ReturnValue = returnValue
	monitorResponse.Message = message

	jsonData, err := json.Marshal(&monitorResponse)
	if err != nil {
		return err.Error()
	}
	return string(jsonData)
}

func monitorAutoScaling(host string, port int, requestType string, jsonData []byte, autoScalingGroupName string) (int, string, error) {
	ip, err := autoscaling.AliasToIP(host)
	if err != nil {
		var message string
		if err == leveldb.ErrNotFound {
			message = fmt.Sprintf("alias not found: %s\n", host)
		} else {
			message = err.Error()
		}
		return http.StatusOK, makeMonitorResponse(halib.MonitorUnknown, message), nil
	}

	if ip == "" {
		message := fmt.Sprintf("%s has not been assigned instance\n", host)
		return http.StatusOK, makeMonitorResponse(halib.MonitorOK, message), nil
	}

	statusCode, jsonStr, perr := postToAgent(ip, port, requestType, jsonData)

	var m halib.MonitorResponse
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return http.StatusInternalServerError, "", err
	}

	message := fmt.Sprintf("%sAutoScaling Group Name: %s\nAutoScaling Instance PrivateIP: %s\n", m.Message, autoScalingGroupName, ip)
	return statusCode, makeMonitorResponse(m.ReturnValue, message), perr
}

func metricAutoScaling(host string, port int, requestType string, jsonData []byte) (int, string, error) {
	ip, err := autoscaling.AliasToIP(host)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return http.StatusNotFound, fmt.Sprintf("alias not found: %s\n", host), nil
		}
		return http.StatusInternalServerError, err.Error(), nil
	}

	if ip == "" {
		return http.StatusServiceUnavailable, fmt.Sprintf("%s has not been assigned instance\n", host), nil
	}

	return postToAgent(ip, port, requestType, jsonData)
}

func makeMetricConfigUpdateResponse(status, message string) string {
	var metricConfigUpdateResponse halib.MetricConfigUpdateResponse
	metricConfigUpdateResponse.Status = status
	metricConfigUpdateResponse.Message = message

	jsonData, err := json.Marshal(&metricConfigUpdateResponse)
	if err != nil {
		return err.Error()
	}
	return string(jsonData)
}

func metricConfigUpdateAutoScaling(autoScalingGroupName string, port int, requestType string, jsonData []byte) (int, string, error) {
	log := util.HappoAgentLogger()

	autoScaling, err := autoscaling.AutoScaling(AutoScalingConfigFile)
	if err != nil {
		message := fmt.Sprintf("failed to fetch autoscaling instances: %s", err.Error())
		log.Error(message)
		return http.StatusInternalServerError, makeMetricConfigUpdateResponse("NG", message), nil
	}

	var autoScalingData halib.AutoScalingData
	for _, a := range autoScaling {
		if a.AutoScalingGroupName == autoScalingGroupName {
			autoScalingData = a
		}
	}

	if autoScalingData.AutoScalingGroupName == "" {
		message := fmt.Sprintf("can't find autoscaling group: %s", autoScalingGroupName)
		log.Error(message)
		return http.StatusNotFound, makeMetricConfigUpdateResponse("NG", message), nil
	}

	var metricConfigUpdateRequest halib.MetricConfigUpdateRequest
	if err := json.Unmarshal(jsonData, &metricConfigUpdateRequest); err != nil {
		message := fmt.Sprintf("failed to parse metric config update request: %s", err.Error())
		log.Error(message)
		return http.StatusInternalServerError, makeMetricConfigUpdateResponse("NG", message), nil
	}

	if err := autoscaling.SaveAutoScalingMetricConfig(autoScalingGroupName, metricConfigUpdateRequest.Config); err != nil {
		message := fmt.Sprintf("failed to save metric config: %s", err.Error())
		log.Error(message)
		return http.StatusInternalServerError, makeMetricConfigUpdateResponse("NG", message), nil
	}

	var errStrings []string
	for _, instance := range autoScalingData.Instances {
		if instance.InstanceData.IP == "" {
			continue
		}

		m := metricConfigUpdateRequest
		for i := 0; i < len(m.Config.Metrics); i++ {
			m.Config.Metrics[i].Hostname = instance.Alias
		}

		jsonData, err := json.Marshal(m)
		if err != nil {
			message := fmt.Sprintf("failed to post request at %s: %s", instance.Alias, err.Error())
			log.Error(message)
			errStrings = append(errStrings, message)
			continue
		}

		_, _, err = postToAgent(instance.InstanceData.IP, port, requestType, jsonData)
		if err != nil {
			message := fmt.Sprintf("failed to post request at %s: %s", instance.Alias, err.Error())
			log.Error(message)
			errStrings = append(errStrings, message)
			continue
		}
	}

	if len(errStrings) > 0 {
		message := fmt.Sprintf("update metric config errors: %s", strings.Join(errStrings, ","))
		log.Error(message)
		return http.StatusServiceUnavailable, makeMetricConfigUpdateResponse("NG", message), nil
	}

	return http.StatusOK, makeMetricConfigUpdateResponse("OK", ""), nil
}

func inventoryAutoScaling(autoScalingGroupName string, port int, requestType string, jsonData []byte) (int, string, error) {
	log := util.HappoAgentLogger()

	autoScaling, err := autoscaling.AutoScaling(AutoScalingConfigFile)
	if err != nil {
		log.Error(fmt.Sprintf("failed to fetch autoscaling instances: %s", err.Error()))
		return http.StatusInternalServerError, "", nil
	}

	var autoScalingData halib.AutoScalingData
	for _, a := range autoScaling {
		if a.AutoScalingGroupName == autoScalingGroupName {
			autoScalingData = a
		}
	}

	if autoScalingData.AutoScalingGroupName == "" {
		log.Error(fmt.Sprintf("can't find autoscaling group: %s", autoScalingGroupName))
		return http.StatusNotFound, "", nil
	}

	ip, err := autoscaling.GetAssignedInstance(autoScalingGroupName)
	if err != nil {
		log.Error(fmt.Sprintf("failed to get assigned instance: %s", err.Error()))
		return http.StatusInternalServerError, "", err
	}
	if ip == "" {
		log.Error(fmt.Sprintf("can't find assigned instance: %s", autoScalingGroupName))
		return http.StatusServiceUnavailable, "", err
	}

	return postToAgent(ip, port, requestType, jsonData)
}

func postToAutoScalingAgent(host string, port int, requestType string, jsonData []byte, autoScalingGroupName string) (int, string, error) {
	switch requestType {
	case "monitor":
		return monitorAutoScaling(host, port, requestType, jsonData, autoScalingGroupName)
	case "metric":
		return metricAutoScaling(host, port, requestType, jsonData)
	case "metric/config/update":
		return metricConfigUpdateAutoScaling(host, port, requestType, jsonData)
	case "inventory":
		return inventoryAutoScaling(host, port, requestType, jsonData)
	default:
		return http.StatusBadRequest, "request_type unsupported", nil
	}
}

func isPermitRefreshAutoScaling(autoScalingGroupName string) bool {
	log := util.HappoAgentLogger()

	refreshAutoScalingMutex.Lock()
	defer refreshAutoScalingMutex.Unlock()

	if _, ok := lastRefreshAutoScaling[autoScalingGroupName]; !ok {
		lastRefreshAutoScaling[autoScalingGroupName] = 0
	}
	duration := time.Now().Unix() - lastRefreshAutoScaling[autoScalingGroupName]
	if duration < refreshAutoScalingIntervalSeconds {
		log.Debug(fmt.Sprintf("Duration of after last refresh autoscaling: %d < %d", duration, refreshAutoScalingIntervalSeconds))
		return false
	}
	lastRefreshAutoScaling[autoScalingGroupName] = time.Now().Unix()
	return true
}

// SetProxyTimeout set timeout of _httpClient
func SetProxyTimeout(timeoutSeconds int64) {
	_httpClient.Timeout = time.Duration(timeoutSeconds) * time.Second
}
