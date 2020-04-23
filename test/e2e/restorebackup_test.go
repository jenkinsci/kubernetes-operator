package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/jenkinsci/kubernetes-operator/internal/try"
	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/constants"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pvcName = "pvc-jenkins"

func TestBackupAndRestore(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)

	defer showLogsAndCleanup(t, ctx)

	jobID := "e2e-jenkins-operator"
	createPVC(t, namespace)
	jenkins := createJenkinsWithBackupAndRestoreConfigured(t, "e2e", namespace)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)

	jenkinsClient, cleanUpFunc := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc()
	waitForJob(t, jenkinsClient, jobID)
	job, err := jenkinsClient.GetJob(jobID)
	require.NoError(t, err, job)
	i, err := job.InvokeSimple(map[string]string{})
	require.NoError(t, err, i)
	// FIXME: waitForJobToFinish use
	time.Sleep(60 * time.Second) // wait for the build to complete

	restartJenkinsMasterPod(t, jenkins)
	waitForRecreateJenkinsMasterPod(t, jenkins)
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	jenkinsClient2, cleanUpFunc2 := verifyJenkinsAPIConnection(t, jenkins, namespace)
	defer cleanUpFunc2()
	waitForJob(t, jenkinsClient2, jobID)
	verifyJobBuildsAfterRestoreBackup(t, jenkinsClient2, jobID)
}

func waitForJob(t *testing.T, jenkinsClient client.Jenkins, jobID string) {
	err := try.Until(func() (end bool, err error) {
		_, err = jenkinsClient.GetJob(jobID)
		return err == nil, err
	}, time.Second*2, time.Minute*3)
	require.NoErrorf(t, err, "Jenkins job '%s' not created by seed job", jobID)
}

func verifyJobBuildsAfterRestoreBackup(t *testing.T, jenkinsClient client.Jenkins, jobID string) {
	job, err := jenkinsClient.GetJob(jobID)
	require.NoError(t, err)
	build, err := job.GetLastBuild()
	require.NoError(t, err)
	assert.Equal(t, int64(1), build.GetBuildNumber())
}

func createPVC(t *testing.T, namespace string) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	err := framework.Global.Client.Create(context.TODO(), pvc, nil)
	require.NoError(t, err)
}

func createJenkinsWithBackupAndRestoreConfigured(t *testing.T, name, namespace string) *v1alpha2.Jenkins {
	containerName := "backup"
	jenkins := &v1alpha2.Jenkins{
		TypeMeta: v1alpha2.JenkinsTypeMeta(),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha2.JenkinsSpec{
			Backup: v1alpha2.Backup{
				ContainerName: containerName,
				Action: v1alpha2.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"/home/user/bin/backup.sh"},
					},
				},
			},
			Restore: v1alpha2.Restore{
				ContainerName: containerName,
				Action: v1alpha2.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"/home/user/bin/restore.sh"},
					},
				},
			},
			Master: v1alpha2.JenkinsMaster{
				Containers: []v1alpha2.Container{
					{
						Name: resources.JenkinsMasterContainerName,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "plugins-cache",
								MountPath: "/usr/share/jenkins/ref/plugins",
							},
						},
					},
					{
						Name:            containerName,
						Image:           "virtuslab/jenkins-operator-backup-pvc:v0.0.6",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "BACKUP_DIR",
								Value: "/backup",
							},
							{
								Name:  "JENKINS_HOME",
								Value: "/jenkins-home",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "backup",
								MountPath: "/backup",
							},
							{
								Name:      "jenkins-home",
								MountPath: "/jenkins-home",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "backup",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
					{
						Name: "plugins-cache",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
			},
			SeedJobs: []v1alpha2.SeedJob{
				{
					ID:                    "jenkins-operator",
					CredentialID:          "jenkins-operator",
					JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
					Targets:               "cicd/jobs/*.jenkins",
					Description:           "Jenkins Operator repository",
					RepositoryBranch:      "master",
					RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
				},
			},
			Service: v1alpha2.Service{
				Type: corev1.ServiceTypeNodePort,
				Port: constants.DefaultHTTPPortInt32,
			},
		},
	}
	updateJenkinsCR(t, jenkins)

	t.Logf("Jenkins CR %+v", *jenkins)
	err := framework.Global.Client.Create(context.TODO(), jenkins, nil)
	require.NoError(t, err)

	return jenkins
}
