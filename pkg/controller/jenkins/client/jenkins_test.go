package client

import (
	"github.com/bndr/gojenkins"
	"net/http"
	"reflect"
	"testing"
)

func TestJenkinsAPIConnectionSettings_BuildJenkinsAPIUrl(t *testing.T) {
	type fields struct {
		Hostname    string
		Port        int
		UseNodePort bool
	}
	type args struct {
		serviceName      string
		serviceNamespace string
		servicePort      int32
		serviceNodePort  int32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := JenkinsAPIConnectionSettings{
				Hostname:    tt.fields.Hostname,
				Port:        tt.fields.Port,
				UseNodePort: tt.fields.UseNodePort,
			}
			if got := j.BuildJenkinsAPIUrl(tt.args.serviceName, tt.args.serviceNamespace, tt.args.servicePort, tt.args.serviceNodePort); got != tt.want {
				t.Errorf("BuildJenkinsAPIUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJenkinsAPIConnectionSettings_Validate(t *testing.T) {
	type fields struct {
		Hostname    string
		Port        int
		UseNodePort bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := JenkinsAPIConnectionSettings{
				Hostname:    tt.fields.Hostname,
				Port:        tt.fields.Port,
				UseNodePort: tt.fields.UseNodePort,
			}
			if err := j.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewBearerTokenAuthorization(t *testing.T) {
	type args struct {
		url   string
		token string
	}
	tests := []struct {
		name    string
		args    args
		want    Jenkins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewBearerTokenAuthorization(tt.args.url, tt.args.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBearerTokenAuthorization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBearerTokenAuthorization() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUserAndPasswordAuthorization(t *testing.T) {
	type args struct {
		url             string
		userName        string
		passwordOrToken string
	}
	tests := []struct {
		name    string
		args    args
		want    Jenkins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUserAndPasswordAuthorization(tt.args.url, tt.args.userName, tt.args.passwordOrToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewUserAndPasswordAuthorization() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUserAndPasswordAuthorization() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isNotFoundError(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundError(tt.args.err); got != tt.want {
				t.Errorf("isNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_jenkins_CreateOrUpdateJob(t *testing.T) {
	type fields struct {
		Jenkins gojenkins.Jenkins
	}
	type args struct {
		config  string
		jobName string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantJob     *gojenkins.Job
		wantCreated bool
		wantErr     bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jenkins := &jenkins{
				Jenkins: tt.fields.Jenkins,
			}
			gotJob, gotCreated, err := jenkins.CreateOrUpdateJob(tt.args.config, tt.args.jobName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdateJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotJob, tt.wantJob) {
				t.Errorf("CreateOrUpdateJob() gotJob = %v, want %v", gotJob, tt.wantJob)
			}
			if gotCreated != tt.wantCreated {
				t.Errorf("CreateOrUpdateJob() gotCreated = %v, want %v", gotCreated, tt.wantCreated)
			}
		})
	}
}

func Test_jenkins_GetNodeSecret(t *testing.T) {
	type fields struct {
		Jenkins gojenkins.Jenkins
	}
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jenkins := &jenkins{
				Jenkins: tt.fields.Jenkins,
			}
			got, err := jenkins.GetNodeSecret(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetNodeSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_jenkins_GetPlugins(t *testing.T) {
	type fields struct {
		Jenkins gojenkins.Jenkins
	}
	type args struct {
		depth int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *gojenkins.Plugins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jenkins := &jenkins{
				Jenkins: tt.fields.Jenkins,
			}
			got, err := jenkins.GetPlugins(tt.args.depth)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPlugins() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPlugins() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newClient(t *testing.T) {
	type args struct {
		url             string
		userName        string
		passwordOrToken string
	}
	tests := []struct {
		name    string
		args    args
		want    Jenkins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newClient(tt.args.url, tt.args.userName, tt.args.passwordOrToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("newClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newClient() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setBearerToken_RoundTrip(t1 *testing.T) {
	type fields struct {
		rt    http.RoundTripper
		token string
	}
	type args struct {
		r *http.Request
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *http.Response
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &setBearerToken{
				rt:    tt.fields.rt,
				token: tt.fields.token,
			}
			got, err := t.RoundTrip(tt.args.r)
			if (err != nil) != tt.wantErr {
				t1.Errorf("RoundTrip() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("RoundTrip() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setBearerToken_transport(t1 *testing.T) {
	type fields struct {
		rt    http.RoundTripper
		token string
	}
	tests := []struct {
		name   string
		fields fields
		want   http.RoundTripper
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &setBearerToken{
				rt:    tt.fields.rt,
				token: tt.fields.token,
			}
			if got := t.transport(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("transport() = %v, want %v", got, tt.want)
			}
		})
	}
}