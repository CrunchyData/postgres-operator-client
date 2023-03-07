package cmd

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"testing"
)

func getRunningPgCluster() unstructured.Unstructured {
	var c unstructured.Unstructured
	c.Object = make(map[string]interface{})
	m := make(map[string]interface{})
	m["shutdown"] = false
	c.Object["spec"] = m
	return c
}

func getStoppedPgCluster() unstructured.Unstructured {
	var c unstructured.Unstructured
	c.Object = make(map[string]interface{})
	m := make(map[string]interface{})
	m["shutdown"] = true
	c.Object["spec"] = m
	return c
}

func Test_changePowerOnUnstructured(t *testing.T) {
	type args struct {
		item   unstructured.Unstructured
		status ClusterStatus
	}
	tests := []struct {
		name        string
		args        args
		want        unstructured.Unstructured
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Power off on already shut downed cluster reports an error",
			args: args{
				item:   getStoppedPgCluster(),
				status: SHUTDOWN,
			},
			want:        unstructured.Unstructured{},
			wantErr:     true,
			wantErrType: AlreadyShutdown{},
		},
		{
			name: "Power on on already running cluster report an error",
			args: args{
				item:   getRunningPgCluster(),
				status: RUNNING,
			},
			want:        unstructured.Unstructured{},
			wantErr:     true,
			wantErrType: AlreadyRunning{},
		},
		{
			name: "Power off on running cluster is successful",
			args: args{
				item:   getRunningPgCluster(),
				status: SHUTDOWN,
			},
			want:        getStoppedPgCluster(),
			wantErr:     false,
			wantErrType: nil,
		},
		{
			name: "Power on on stopped cluster is successful",
			args: args{
				item:   getStoppedPgCluster(),
				status: RUNNING,
			},
			want:        getRunningPgCluster(),
			wantErr:     false,
			wantErrType: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := changePowerOnUnstructured(tt.args.item, tt.args.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("changePowerOnUnstructured() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) == tt.wantErr && err != tt.wantErrType {
				t.Errorf("changePowerOnUnstructured() received error if wrong type. Expected %t but received %t", tt.wantErrType, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("changePowerOnUnstructured() got = %v, want %v", got, tt.want)
			}
		})
	}
}
