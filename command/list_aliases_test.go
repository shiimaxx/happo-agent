package command

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heartbeatsjp/happo-agent/halib"
	"github.com/stretchr/testify/assert"
)

func Test_listAliases(t *testing.T) {
	var res halib.AutoScalingResponse
	res.AutoScaling = []halib.AutoScalingData{
		{
			AutoScalingGroupName: "dummy-prod-ag",
			Instances: []struct {
				Alias        string             `json:"alias"`
				InstanceData halib.InstanceData `json:"instance_data"`
			}{
				{
					Alias: "dummy-prod-ag-web-01",
					InstanceData: halib.InstanceData{
						InstanceID: "i-0123456789abcded1",
						IP:         "192.0.2.1",
					},
				},
				{
					Alias: "dummy-prod-ag-web-02",
					InstanceData: halib.InstanceData{
						InstanceID: "i-0123456789abcded2",
						IP:         "192.0.2.2",
					},
				},
				{
					Alias: "dummy-prod-ag-web-03",
					InstanceData: halib.InstanceData{
						InstanceID: "",
						IP:         "",
					},
				},
				{
					Alias: "dummy-prod-ag-web-04",
					InstanceData: halib.InstanceData{
						InstanceID: "",
						IP:         "",
					},
				},
			},
		},
		{
			AutoScalingGroupName: "dummy-stg-ag",
			Instances: []struct {
				Alias        string             `json:"alias"`
				InstanceData halib.InstanceData `json:"instance_data"`
			}{
				{
					Alias: "dummy-stg-ag-web-01",
					InstanceData: halib.InstanceData{
						InstanceID: "i-1123456789abcded1",
						IP:         "192.0.2.11",
					},
				},
				{
					Alias: "dummy-stg-ag-web-02",
					InstanceData: halib.InstanceData{
						InstanceID: "i-1123456789abcded2",
						IP:         "192.0.2.12",
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, string(b))
			}))
	defer ts.Close()

	type args struct {
		bastionEndpoint string
		agName          string
		listAll         bool
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "",
			args: args{
				bastionEndpoint: ts.URL,
				agName:          "",
				listAll:         false,
			},
			want: `dummy-prod-ag-web-01,192.0.2.1,i-0123456789abcded1
dummy-prod-ag-web-02,192.0.2.2,i-0123456789abcded2
dummy-stg-ag-web-01,192.0.2.11,i-1123456789abcded1
dummy-stg-ag-web-02,192.0.2.12,i-1123456789abcded2`,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				bastionEndpoint: ts.URL,
				agName:          "dummy-prod-ag",
				listAll:         false,
			},
			want: `dummy-prod-ag-web-01,192.0.2.1,i-0123456789abcded1
dummy-prod-ag-web-02,192.0.2.2,i-0123456789abcded2`,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				bastionEndpoint: ts.URL,
				agName:          "",
				listAll:         true,
			},
			want: `dummy-prod-ag-web-01,192.0.2.1,i-0123456789abcded1
dummy-prod-ag-web-02,192.0.2.2,i-0123456789abcded2
dummy-prod-ag-web-03,,
dummy-prod-ag-web-04,,
dummy-stg-ag-web-01,192.0.2.11,i-1123456789abcded1
dummy-stg-ag-web-02,192.0.2.12,i-1123456789abcded2`,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				bastionEndpoint: ts.URL,
				agName:          "missing-ag",
				listAll:         false,
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listAliases(tt.args.bastionEndpoint, tt.args.agName, tt.args.listAll)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			assert.Equal(t, got, tt.want)
		})
	}
}

func Test_listAlias_emptyResponse(t *testing.T) {
	var res halib.AutoScalingResponse
	res.AutoScaling = []halib.AutoScalingData{}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, string(b))
			}))
	defer ts.Close()

	got, err := listAliases(ts.URL, "", false)
	assert.Equal(t, got, "")
	assert.NotNil(t, err)
}
