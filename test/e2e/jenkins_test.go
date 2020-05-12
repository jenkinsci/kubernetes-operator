package e2e

import (
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	corev1 "k8s.io/api/core/v1"
	"reflect"
	"testing"
)

func Test_createJenkinsAPIClientFromSecret(t *testing.T) {
	type args struct {
		t             *testing.T
		jenkins       *v1alpha2.Jenkins
		jenkinsAPIURL string
	}
	tests := []struct {
		name    string
		args    args
		want    client.Jenkins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createJenkinsAPIClientFromSecret(tt.args.t, tt.args.jenkins, tt.args.jenkinsAPIURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("createJenkinsAPIClientFromSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createJenkinsAPIClientFromSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createJenkinsAPIClientFromServiceAccount(t *testing.T) {
	type args struct {
		t             *testing.T
		jenkins       *v1alpha2.Jenkins
		jenkinsAPIURL string
	}
	tests := []struct {
		name    string
		args    args
		want    client.Jenkins
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createJenkinsAPIClientFromServiceAccount(tt.args.t, tt.args.jenkins, tt.args.jenkinsAPIURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("createJenkinsAPIClientFromServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createJenkinsAPIClientFromServiceAccount() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createJenkinsCR(t *testing.T) {
	type args struct {
		t                 *testing.T
		name              string
		namespace         string
		seedJob           *[]v1alpha2.SeedJob
		groovyScripts     v1alpha2.GroovyScripts
		casc              v1alpha2.ConfigurationAsCode
		priorityClassName string
	}
	tests := []struct {
		name string
		args args
		want *v1alpha2.Jenkins
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createJenkinsCR(tt.args.t, tt.args.name, tt.args.namespace, tt.args.seedJob, tt.args.groovyScripts, tt.args.casc, tt.args.priorityClassName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createJenkinsCR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getJenkins(t *testing.T) {
	type args struct {
		t         *testing.T
		namespace string
		name      string
	}
	tests := []struct {
		name string
		args args
		want *v1alpha2.Jenkins
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJenkins(tt.args.t, tt.args.namespace, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJenkins() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getJenkinsMasterPod(t *testing.T) {
	type args struct {
		t       *testing.T
		jenkins *v1alpha2.Jenkins
	}
	tests := []struct {
		name string
		args args
		want *corev1.Pod
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getJenkinsMasterPod(tt.args.t, tt.args.jenkins); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJenkinsMasterPod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_restartJenkinsMasterPod(t *testing.T) {
	type args struct {
		t       *testing.T
		jenkins *v1alpha2.Jenkins
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_verifyJenkinsAPIConnection(t *testing.T) {
	type args struct {
		t         *testing.T
		jenkins   *v1alpha2.Jenkins
		namespace string
	}
	tests := []struct {
		name  string
		args  args
		want  client.Jenkins
		want1 func()
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := verifyJenkinsAPIConnection(tt.args.t, tt.args.jenkins, tt.args.namespace)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("verifyJenkinsAPIConnection() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("verifyJenkinsAPIConnection() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}