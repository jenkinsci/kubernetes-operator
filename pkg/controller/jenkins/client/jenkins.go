package client

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

var (
	errorNotFound = errors.New("404")
	regex         = regexp.MustCompile("(<application-desc main-class=\"hudson.remoting.jnlp.Main\"><argument>)(?P<secret>[a-z0-9]*)")
)

// Jenkins defines Jenkins API
type Jenkins interface {
	GenerateToken(userName, tokenName string) (*UserToken, error)
	Info() (*gojenkins.ExecutorResponse, error)
	SafeRestart() error
	CreateNode(name string, numExecutors int, description string, remoteFS string, label string, options ...interface{}) (*gojenkins.Node, error)
	DeleteNode(name string) (bool, error)
	CreateFolder(name string, parents ...string) (*gojenkins.Folder, error)
	CreateJobInFolder(config string, jobName string, parentIDs ...string) (*gojenkins.Job, error)
	CreateJob(config string, options ...interface{}) (*gojenkins.Job, error)
	CreateOrUpdateJob(config, jobName string) (*gojenkins.Job, bool, error)
	RenameJob(job string, name string) *gojenkins.Job
	CopyJob(copyFrom string, newName string) (*gojenkins.Job, error)
	DeleteJob(name string) (bool, error)
	BuildJob(name string, options ...interface{}) (int64, error)
	GetNode(name string) (*gojenkins.Node, error)
	GetLabel(name string) (*gojenkins.Label, error)
	GetBuild(jobName string, number int64) (*gojenkins.Build, error)
	GetJob(id string, parentIDs ...string) (*gojenkins.Job, error)
	GetSubJob(parentID string, childID string) (*gojenkins.Job, error)
	GetFolder(id string, parents ...string) (*gojenkins.Folder, error)
	GetAllNodes() ([]*gojenkins.Node, error)
	GetAllBuildIds(job string) ([]gojenkins.JobBuild, error)
	GetAllJobNames() ([]gojenkins.InnerJob, error)
	GetAllJobs() ([]*gojenkins.Job, error)
	GetQueue() (*gojenkins.Queue, error)
	GetQueueUrl() string
	GetQueueItem(id int64) (*gojenkins.Task, error)
	GetArtifactData(id string) (*gojenkins.FingerPrintResponse, error)
	GetPlugins(depth int) (*gojenkins.Plugins, error)
	UninstallPlugin(name string) error
	HasPlugin(name string) (*gojenkins.Plugin, error)
	InstallPlugin(name string, version string) error
	ValidateFingerPrint(id string) (bool, error)
	GetView(name string) (*gojenkins.View, error)
	GetAllViews() ([]*gojenkins.View, error)
	CreateView(name string, viewType string) (*gojenkins.View, error)
	Poll() (int, error)
	ExecuteScript(groovyScript string) (logs string, err error)
	GetNodeSecret(name string) (string, error)
}

type jenkins struct {
	gojenkins.Jenkins
}

// CreateOrUpdateJob creates or updates a job from config
func (jenkins *jenkins) CreateOrUpdateJob(config, jobName string) (job *gojenkins.Job, created bool, err error) {
	// create or update
	job, err = jenkins.GetJob(jobName)
	if isNotFoundError(err) {
		job, err = jenkins.CreateJob(config, jobName)
		created = true
		return job, true, errors.WithStack(err)
	} else if err != nil {
		return job, false, errors.WithStack(err)
	}

	err = job.UpdateConfig(config)
	return job, false, errors.WithStack(err)
}

// BuildJenkinsAPIUrl returns Jenkins API URL
func BuildJenkinsAPIUrl(service v1.Service, hostname string, port int, useNodePort bool) string {
	if hostname == "" && port == -1 {
		return fmt.Sprintf("http://%s.%s:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
	}

	if hostname != "" && useNodePort {
		return fmt.Sprintf("http://%s:%d", hostname, service.Spec.Ports[0].NodePort)
	}

	return fmt.Sprintf("http://%s:%d", hostname, port)
}

// New creates Jenkins API client
func New(url, user, passwordOrToken string) (Jenkins, error) {
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}

	jenkinsClient := &jenkins{}
	jenkinsClient.Server = url
	jenkinsClient.Requester = &gojenkins.Requester{
		Base:      url,
		SslVerify: true,
		Client:    http.DefaultClient,
		BasicAuth: &gojenkins.BasicAuth{Username: user, Password: passwordOrToken},
	}
	if _, err := jenkinsClient.Init(); err != nil {
		return nil, errors.Wrap(err, "couldn't init Jenkins API client")
	}

	status, err := jenkinsClient.Poll()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't poll data from Jenkins API")
	}
	if status != http.StatusOK {
		return nil, errors.Errorf("couldn't poll data from Jenkins API, invalid status code returned: %d", status)
	}

	return jenkinsClient, nil
}

func isNotFoundError(err error) bool {
	if err != nil {
		return err.Error() == errorNotFound.Error()
	}
	return false
}

func (jenkins *jenkins) GetNodeSecret(name string) (string, error) {
	var content string
	_, err := jenkins.Requester.GetXML(fmt.Sprintf("/computer/%s/slave-agent.jnlp", name), &content, nil)
	if err != nil {
		return "", errors.WithStack(err)
	}

	match := regex.FindStringSubmatch(content)
	if match == nil {
		return "", errors.New("Node secret cannot be parsed")
	}

	result := make(map[string]string)

	for i, name := range regex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	return result["secret"], nil
}
