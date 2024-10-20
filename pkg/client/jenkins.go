package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
)

var (
	errorNotFound = errors.New("404")
)

// Jenkins defines Jenkins API.
type Jenkins interface {
	GenerateToken(userName, tokenName string) (*UserToken, error)
	Info(ctx context.Context) (*gojenkins.ExecutorResponse, error)
	SafeRestart(ctx context.Context) error
	CreateNode(ctx context.Context, name string, numExecutors int, description string, remoteFS string, label string, options ...interface{}) (*gojenkins.Node, error)
	DeleteNode(ctx context.Context, name string) (bool, error)
	CreateFolder(ctx context.Context, name string, parents ...string) (*gojenkins.Folder, error)
	CreateJobInFolder(ctx context.Context, config string, jobName string, parentIDs ...string) (*gojenkins.Job, error)
	CreateJob(ctx context.Context, config string, options ...interface{}) (*gojenkins.Job, error)
	CreateOrUpdateJob(config, jobName string) (*gojenkins.Job, bool, error)
	RenameJob(ctx context.Context, job string, name string) *gojenkins.Job
	CopyJob(ctx context.Context, copyFrom string, newName string) (*gojenkins.Job, error)
	CreateView(ctx context.Context, name string, viewType string) (*gojenkins.View, error)
	DeleteJob(ctx context.Context, name string) (bool, error)
	BuildJob(ctx context.Context, name string, options map[string]string) (int64, error)
	GetNode(ctx context.Context, name string) (*gojenkins.Node, error)
	GetLabel(ctx context.Context, name string) (*gojenkins.Label, error)
	GetBuild(jobName string, number int64) (*gojenkins.Build, error)
	GetJob(ctx context.Context, id string, parentIDs ...string) (*gojenkins.Job, error)
	GetSubJob(ctx context.Context, parentID string, childID string) (*gojenkins.Job, error)
	GetFolder(ctx context.Context, id string, parents ...string) (*gojenkins.Folder, error)
	GetAllNodes(ctx context.Context) ([]*gojenkins.Node, error)
	GetAllBuildIds(ctx context.Context, job string) ([]gojenkins.JobBuild, error)
	GetAllJobNames(context.Context) ([]gojenkins.InnerJob, error)
	GetAllJobs(context.Context) ([]*gojenkins.Job, error)
	GetQueue(context.Context) (*gojenkins.Queue, error)
	GetQueueUrl() string
	GetQueueItem(ctx context.Context, id int64) (*gojenkins.Task, error)
	GetArtifactData(ctx context.Context, id string) (*gojenkins.FingerPrintResponse, error)
	GetPlugins(depth int) (*gojenkins.Plugins, error)
	UninstallPlugin(ctx context.Context, name string) error
	HasPlugin(ctx context.Context, name string) (*gojenkins.Plugin, error)
	InstallPlugin(ctx context.Context, name string, version string) error
	ValidateFingerPrint(ctx context.Context, id string) (bool, error)
	GetView(ctx context.Context, name string) (*gojenkins.View, error)
	GetAllViews(context.Context) ([]*gojenkins.View, error)
	Poll(ctx context.Context) (int, error)
	ExecuteScript(groovyScript string) (logs string, err error)
	GetNodeSecret(name string) (string, error)
}

type jenkins struct {
	gojenkins.Jenkins
}

// JenkinsAPIConnectionSettings is struct that handle information about Jenkins API connection.
type JenkinsAPIConnectionSettings struct {
	Hostname    string
	Port        int
	UseNodePort bool
}

type setBearerToken struct {
	rt    http.RoundTripper
	token string
}

func (t *setBearerToken) transport() http.RoundTripper {
	if t.rt != nil {
		return t.rt
	}
	return http.DefaultTransport
}

func (t *setBearerToken) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.token))
	return t.transport().RoundTrip(r)
}

// CreateOrUpdateJob creates or updates a job from config.
func (jenkins *jenkins) CreateOrUpdateJob(config, jobName string) (job *gojenkins.Job, created bool, err error) {
	// create or update
	job, err = jenkins.GetJob(context.TODO(), jobName)
	if isNotFoundError(err) {
		job, err = jenkins.CreateJob(context.TODO(), config, jobName)
		return job, true, errors.WithStack(err)
	} else if err != nil {
		return job, false, errors.WithStack(err)
	}

	err = job.UpdateConfig(context.TODO(), config)
	return job, false, errors.WithStack(err)
}

// BuildJenkinsAPIUrl returns Jenkins API URL.
func (j JenkinsAPIConnectionSettings) BuildJenkinsAPIUrl(serviceName string, serviceNamespace string, servicePort int32, serviceNodePort int32) string {
	if j.Hostname == "" && j.Port == 0 {
		return fmt.Sprintf("http://%s.%s:%d", serviceName, serviceNamespace, servicePort)
	}

	if j.Hostname != "" && j.UseNodePort {
		return fmt.Sprintf("http://%s:%d", j.Hostname, serviceNodePort)
	}

	return fmt.Sprintf("http://%s:%d", j.Hostname, j.Port)
}

// Validate validates jenkins API connection settings.
func (j JenkinsAPIConnectionSettings) Validate() error {
	if j.Port > 0 && j.UseNodePort {
		return errors.New("can't use service port and nodePort both. Please use port or nodePort")
	}

	if j.Port < 0 {
		return errors.New("service port cannot be lower than 0")
	}

	if (j.Hostname == "" && j.Port > 0) || (j.Hostname == "" && j.UseNodePort) {
		return errors.New("empty hostname is now allowed. Please provide hostname")
	}

	return nil
}

// NewUserAndPasswordAuthorization creates Jenkins API client with user and password authorization.
func NewUserAndPasswordAuthorization(url, userName, passwordOrToken string) (Jenkins, error) {
	return newClient(url, userName, passwordOrToken)
}

// NewBearerTokenAuthorization creates Jenkins API client with bearer token authorization.
func NewBearerTokenAuthorization(url, token string) (Jenkins, error) {
	return newClient(url, "", token)
}

func newClient(url, userName, passwordOrToken string) (Jenkins, error) {
	// if strings.HasSuffix(url, "/") {
	// url = url[:len(url)-1]
	url = strings.TrimSuffix(url, "/")
	// }

	jenkinsClient := &jenkins{}
	jenkinsClient.Server = url

	var basicAuth *gojenkins.BasicAuth
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create a cookie jar")
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: 20 * time.Second,
	}

	if len(userName) > 0 && len(passwordOrToken) > 0 {
		basicAuth = &gojenkins.BasicAuth{Username: userName, Password: passwordOrToken}
	} else {
		httpClient.Transport = &setBearerToken{token: passwordOrToken, rt: httpClient.Transport}
	}

	jenkinsClient.Requester = &gojenkins.Requester{
		Base:      url,
		SslVerify: true,
		Client:    httpClient,
		BasicAuth: basicAuth,
	}
	if _, err := jenkinsClient.Init(context.TODO()); err != nil {
		return nil, errors.Wrap(err, "couldn't init Jenkins API client")
	}

	status, err := jenkinsClient.Poll(context.TODO())
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
	url := fmt.Sprintf("%s/scriptText", jenkins.Server)
	script := fmt.Sprintf(`println(jenkins.model.Jenkins.getInstance().getComputer("%s").getJnlpMac())`, name)
	payload := bytes.NewBufferString(fmt.Sprintf("script=%s", script))

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		return "", errors.WithStack(err)
	}
	req.SetBasicAuth(jenkins.Requester.BasicAuth.Username, jenkins.Requester.BasicAuth.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get node secret: %s", string(body))
	}

	return string(body), nil
}

// Returns the list of all plugins installed on the Jenkins server.
// You can supply depth parameter, to limit how much data is returned.
func (jenkins *jenkins) GetPlugins(depth int) (*gojenkins.Plugins, error) {
	p := gojenkins.Plugins{Jenkins: &jenkins.Jenkins, Raw: new(gojenkins.PluginResponse), Base: "/pluginManager", Depth: depth}
	statusCode, err := p.Poll(context.TODO())
	if err != nil {
		return nil, err
	}
	if statusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code returned: %d", statusCode)
	}
	return &p, nil
}
