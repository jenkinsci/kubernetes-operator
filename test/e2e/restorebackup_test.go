package e2e

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"
	"github.com/jenkinsci/kubernetes-operator/pkg/constants"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const pvcName = "pvc-jenkins"

func waitForJobCreation(jenkinsClient client.Jenkins, jobID string) {
	By("waiting for Jenkins job creation")

	var err error
	Eventually(func() (bool, error) {
		_, err = jenkinsClient.GetJob(jobID)
		if err != nil {
			return false, err
		}
		return err == nil, err
	}, time.Minute*3, time.Second*2).Should(BeTrue())

	Expect(err).NotTo(HaveOccurred())
}

func verifyJobBuildsAfterRestoreBackup(jenkinsClient client.Jenkins, jobID string) {
	By("checking if job builds after restoring backup")

	job, err := jenkinsClient.GetJob(jobID)
	Expect(err).NotTo(HaveOccurred())
	build, err := job.GetLastBuild()
	Expect(err).NotTo(HaveOccurred())

	Expect(build.GetBuildNumber()).To(Equal(int64(1)))
}

func createPVC(namespace string) {
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

	Expect(K8sClient.Create(context.TODO(), pvc)).Should(Succeed())
}

func createJenkinsWithBackupAndRestoreConfigured(name, namespace string) *v1alpha2.Jenkins {
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
				GetLatestAction: v1alpha2.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"/home/user/bin/get-latest.sh"},
					},
				},
				Action: v1alpha2.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{"/home/user/bin/restore.sh"},
					},
				},
			},
			GroovyScripts: v1alpha2.GroovyScripts{
				Customization: v1alpha2.Customization{
					Configurations: []v1alpha2.ConfigMapRef{},
				},
			},
			ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
				Customization: v1alpha2.Customization{
					Configurations: []v1alpha2.ConfigMapRef{},
				},
			},
			Master: v1alpha2.JenkinsMaster{
				Containers: []v1alpha2.Container{
					{
						Name:  resources.JenkinsMasterContainerName,
						Image: JenkinsTestImage,
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "plugins-cache",
								MountPath: "/usr/share/jenkins/ref/plugins",
							},
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(100),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(30),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(5),
						},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path:   "/login",
									Port:   intstr.FromString("http"),
									Scheme: corev1.URISchemeHTTP,
								},
							},
							InitialDelaySeconds: int32(80),
							TimeoutSeconds:      int32(4),
							FailureThreshold:    int32(30),
							SuccessThreshold:    int32(1),
							PeriodSeconds:       int32(5),
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("250m"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("1000m"),
								corev1.ResourceMemory: resource.MustParse("3Gi"),
							},
						},
					},
					{
						Name:            containerName,
						Image:           "virtuslab/jenkins-operator-backup-pvc:v0.1.1",
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

	updateJenkinsCR(jenkins)

	_, _ = fmt.Fprintf(GinkgoWriter, "Jenkins CR %+v\n", *jenkins)

	Expect(K8sClient.Create(context.TODO(), jenkins)).Should(Succeed())

	return jenkins
}

func resetJenkinsStatus(jenkins *v1alpha2.Jenkins) {
	By("resetting Jenkins status")

	jenkins = getJenkins(jenkins.Namespace, jenkins.Name)
	jenkins.Status = v1alpha2.JenkinsStatus{}
	Expect(K8sClient.Status().Update(context.TODO(), jenkins)).Should(Succeed())
}
