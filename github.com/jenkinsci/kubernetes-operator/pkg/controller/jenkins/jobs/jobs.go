package jobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrorUnexpectedBuildStatus - this is custom error returned when jenkins build has unexpected status
	ErrorUnexpectedBuildStatus = fmt.Errorf("unexpected build status")
	// ErrorBuildFailed - this is custom error returned when jenkins build has failed
	ErrorBuildFailed = fmt.Errorf("build failed")
	// ErrorAbortBuildFailed - this is custom error returned when jenkins build couldn't be aborted
	ErrorAbortBuildFailed = fmt.Errorf("build abort failed")
	// ErrorUnrecoverableBuildFailed - this is custom error returned when jenkins build has failed and cannot be recovered
	ErrorUnrecoverableBuildFailed = fmt.Errorf("build failed and cannot be recovered")
	// ErrorNotFound - this is error returned when jenkins build couldn't be found
	ErrorNotFound = fmt.Errorf("404")
	// BuildRetires - determines max amount of retires for failed build
	BuildRetires = 3
)

// Jobs defines Jobs API tailored for operator sdk
type Jobs struct {
	jenkinsClient client.Jenkins
	logger        logr.Logger
	k8sClient     k8s.Client
}

// New creates jobs client
func New(jenkinsClient client.Jenkins, k8sClient k8s.Client, logger logr.Logger) *Jobs {
	return &Jobs{
		jenkinsClient: jenkinsClient,
		k8sClient:     k8sClient,
		logger:        logger,
	}
}

// EnsureBuildJob function takes care of jenkins build lifecycle according to the lifecycle of reconciliation loop
// implementation guarantees that jenkins build can be properly handled even after operator pod restart
// entire state is saved in Jenkins.Status.Builds section
// function return 'true' when build finished successfully or false when reconciliation loop should requeue this function
// preserveStatus determines that build won't be removed from Jenkins.Status.Builds section
func (jobs *Jobs) EnsureBuildJob(jobName, hash string, parameters map[string]string, jenkins *v1alpha2.Jenkins, preserveStatus bool) (done bool, err error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring build, name:'%s' hash:'%s'", jobName, hash))

	build := jobs.getBuildFromStatus(jobName, hash, jenkins)
	if build != nil {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Build exists in status, %+v", build))
		switch build.Status {
		case v1alpha2.BuildSuccessStatus:
			return jobs.ensureSuccessBuild(*build, jenkins, preserveStatus)
		case v1alpha2.BuildRunningStatus:
			return jobs.ensureRunningBuild(*build, jenkins, preserveStatus)
		case v1alpha2.BuildUnstableStatus, v1alpha2.BuildNotBuildStatus, v1alpha2.BuildFailureStatus, v1alpha2.BuildAbortedStatus:
			return jobs.ensureFailedBuild(*build, jenkins, parameters, preserveStatus)
		case v1alpha2.BuildExpiredStatus:
			return jobs.ensureExpiredBuild(*build, jenkins, preserveStatus)
		default:
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Unexpected build status, %+v", build))
			return false, ErrorUnexpectedBuildStatus
		}
	}

	// build is run first time - build job and update status
	created := metav1.Now()
	newBuild := v1alpha2.Build{
		JobName:    jobName,
		Hash:       hash,
		CreateTime: &created,
	}
	return jobs.buildJob(newBuild, parameters, jenkins)
}

func (jobs *Jobs) getBuildFromStatus(jobName string, hash string, jenkins *v1alpha2.Jenkins) *v1alpha2.Build {
	if jenkins != nil {
		builds := jenkins.Status.Builds
		for _, build := range builds {
			if build.JobName == jobName && build.Hash == hash {
				return &build
			}
		}
	}
	return nil
}

func (jobs *Jobs) ensureSuccessBuild(build v1alpha2.Build, jenkins *v1alpha2.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring success build, %+v", build))

	if !preserveStatus {
		err := jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}
	return true, nil
}

func (jobs *Jobs) ensureRunningBuild(build v1alpha2.Build, jenkins *v1alpha2.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring running build, %+v", build))
	// FIXME (antoniaklja) implement build expiration

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if isNotFoundError(err) {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Build still running , %+v", build))
		return false, nil
	} else if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't get jenkins build, %+v", build))
		return false, errors.WithStack(err)
	}

	if jenkinsBuild.GetResult() != "" {
		build.Status = v1alpha2.BuildStatus(strings.ToLower(jenkinsBuild.GetResult()))
	}

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Couldn't update build status, %+v", build))
		return false, err
	}

	if build.Status == v1alpha2.BuildSuccessStatus {
		jobs.logger.Info(fmt.Sprintf("Build finished successfully, %+v", build))
		return true, nil
	}

	if build.Status == v1alpha2.BuildFailureStatus || build.Status == v1alpha2.BuildUnstableStatus ||
		build.Status == v1alpha2.BuildNotBuildStatus || build.Status == v1alpha2.BuildAbortedStatus {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Build failed, %+v", build))
		return false, ErrorBuildFailed
	}

	return false, nil
}

func (jobs *Jobs) ensureFailedBuild(build v1alpha2.Build, jenkins *v1alpha2.Jenkins, parameters map[string]string, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring failed build, %+v", build))

	if build.Retires < BuildRetires {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Retrying build, %+v", build))
		build.Retires = build.Retires + 1
		_, err := jobs.buildJob(build, parameters, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't retry build, %+v", build))
			return false, err
		}
		return false, nil
	}

	lastFailedBuild, err := jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if err != nil {
		return false, err
	}
	jobs.logger.V(log.VWarn).Info(fmt.Sprintf("The retries limit was reached, build %+v, logs: %s", build, lastFailedBuild.GetConsoleOutput()))

	if !preserveStatus {
		err := jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}
	return false, ErrorUnrecoverableBuildFailed
}

func (jobs *Jobs) ensureExpiredBuild(build v1alpha2.Build, jenkins *v1alpha2.Jenkins, preserveStatus bool) (bool, error) {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Ensuring expired build, %+v", build))

	jenkinsBuild, err := jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if err != nil {
		return false, errors.WithStack(err)
	}

	_, err = jenkinsBuild.Stop()
	if err != nil {
		return false, errors.WithStack(err)
	}

	jenkinsBuild, err = jobs.jenkinsClient.GetBuild(build.JobName, build.Number)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if v1alpha2.BuildStatus(jenkinsBuild.GetResult()) != v1alpha2.BuildAbortedStatus {
		return false, ErrorAbortBuildFailed
	}

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		return false, err
	}

	// TODO(antoniaklja) clean up k8s resources

	if !preserveStatus {
		err = jobs.removeBuildFromStatus(build, jenkins)
		if err != nil {
			jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't remove build from status, %+v", build))
			return false, err
		}
	}

	return true, nil
}

func (jobs *Jobs) removeBuildFromStatus(build v1alpha2.Build, jenkins *v1alpha2.Jenkins) error {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Removing build from status, %+v", build))
	builds := make([]v1alpha2.Build, len(jenkins.Status.Builds))
	for _, existingBuild := range jenkins.Status.Builds {
		if existingBuild.JobName != build.JobName && existingBuild.Hash != build.Hash {
			builds = append(builds, existingBuild)
		}
	}
	jenkins.Status.Builds = builds
	err := jobs.k8sClient.Update(context.TODO(), jenkins)
	if err != nil {
		return err // don't wrap because apierrors.IsConflict(err) won't work in jenkins_controller
	}

	return nil
}

func (jobs *Jobs) buildJob(build v1alpha2.Build, parameters map[string]string, jenkins *v1alpha2.Jenkins) (bool, error) {
	jobs.logger.Info(fmt.Sprintf("Running job, %+v", build))
	job, err := jobs.jenkinsClient.GetJob(build.JobName)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't find jenkins job, %+v", build))
		return false, errors.WithStack(err)
	}
	nextBuildNumber := job.GetDetails().NextBuildNumber

	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Running build, %+v", build))
	_, err = jobs.jenkinsClient.BuildJob(build.JobName, parameters)
	if err != nil {
		jobs.logger.V(log.VWarn).Info(fmt.Sprintf("Couldn't run build, %+v", build))
		return false, errors.WithStack(err)
	}

	build.Status = v1alpha2.BuildRunningStatus
	build.Number = nextBuildNumber

	err = jobs.updateBuildStatus(build, jenkins)
	if err != nil {
		jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Couldn't update build status, %+v", build))
		return false, err
	}
	return false, nil
}

func (jobs *Jobs) updateBuildStatus(build v1alpha2.Build, jenkins *v1alpha2.Jenkins) error {
	jobs.logger.V(log.VDebug).Info(fmt.Sprintf("Updating build status, %+v", build))
	// get index of existing build from status if exists
	buildIndex := -1
	for index, existingBuild := range jenkins.Status.Builds {
		if build.JobName == existingBuild.JobName && build.Hash == existingBuild.Hash {
			buildIndex = index
		}
	}

	// update build status
	now := metav1.Now()
	build.LastUpdateTime = &now
	if buildIndex >= 0 {
		jenkins.Status.Builds[buildIndex] = build
	} else {
		build.CreateTime = &now
		jenkins.Status.Builds = append(jenkins.Status.Builds, build)
	}
	err := jobs.k8sClient.Update(context.TODO(), jenkins)
	if err != nil {
		return err // don't wrap because apierrors.IsConflict(err) won't work in jenkins_controller
	}

	return nil
}

func isNotFoundError(err error) bool {
	if err != nil {
		return err.Error() == ErrorNotFound.Error()
	}
	return false
}
