package seedjobs

import (
	"context"
	"fmt"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/controller/jenkins/configuration/base/resources"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestEnsureSeedJobs(t *testing.T) {
	// given
	logger := logf.ZapLogger(false)
	ctrl := gomock.NewController(t)
	ctx := context.TODO()
	defer ctrl.Finish()

	jenkinsClient := client.NewMockJenkins(ctrl)
	fakeClient := fake.NewFakeClient()
	err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
	assert.NoError(t, err)

	jenkins := jenkinsCustomResource()
	err = fakeClient.Create(ctx, jenkins)
	assert.NoError(t, err)
	buildNumber := int64(1)

	for reconcileAttempt := 1; reconcileAttempt <= 2; reconcileAttempt++ {
		logger.Info(fmt.Sprintf("Reconcile attempt #%d", reconcileAttempt))

		seedJobs := New(jenkinsClient, fakeClient, logger)

		// first run - should create job and schedule build
		if reconcileAttempt == 1 {
			jenkinsClient.
				EXPECT().
				CreateOrUpdateJob(seedJobConfigXML, ConfigureSeedJobsName).
				Return(nil, true, nil)

			jenkinsClient.
				EXPECT().
				GetJob(ConfigureSeedJobsName).
				Return(&gojenkins.Job{
					Raw: &gojenkins.JobResponse{
						NextBuildNumber: buildNumber,
					},
				}, nil)

			jenkinsClient.
				EXPECT().
				BuildJob(ConfigureSeedJobsName, gomock.Any()).
				Return(int64(0), nil)
		}

		// second run - should update and finish job
		if reconcileAttempt == 2 {
			jenkinsClient.
				EXPECT().
				CreateOrUpdateJob(seedJobConfigXML, ConfigureSeedJobsName).
				Return(nil, false, nil)

			jenkinsClient.
				EXPECT().
				GetBuild(ConfigureSeedJobsName, gomock.Any()).
				Return(&gojenkins.Build{
					Raw: &gojenkins.BuildResponse{
						Result: string(v1alpha2.BuildSuccessStatus),
					},
				}, nil)
		}

		done, err := seedJobs.EnsureSeedJobs(jenkins)
		assert.NoError(t, err)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(jenkins.Status.Builds), "There is one running job")
		build := jenkins.Status.Builds[0]
		assert.Equal(t, buildNumber, build.Number)
		assert.Equal(t, ConfigureSeedJobsName, build.JobName)
		assert.NotNil(t, build.CreateTime)
		assert.NotEmpty(t, build.Hash)
		assert.NotNil(t, build.LastUpdateTime)
		assert.Equal(t, 0, build.Retires)

		// first run - should create job and schedule build
		if reconcileAttempt == 1 {
			assert.False(t, done)
			assert.Equal(t, string(v1alpha2.BuildRunningStatus), string(build.Status))
		}

		// second run - should update and finish job
		if reconcileAttempt == 2 {
			assert.False(t, done)
			assert.Equal(t, string(v1alpha2.BuildSuccessStatus), string(build.Status))
		}

	}
}

func jenkinsCustomResource() *v1alpha2.Jenkins {
	return &v1alpha2.Jenkins{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jenkins",
			Namespace: "default",
		},
		Spec: v1alpha2.JenkinsSpec{
			Master: v1alpha2.JenkinsMaster{
				Annotations: map[string]string{"test": "label"},
				Containers: []v1alpha2.Container{
					{
						Name:  resources.JenkinsMasterContainerName,
						Image: "jenkins/jenkins",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("300m"),
								corev1.ResourceMemory: resource.MustParse("500Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("2"),
								corev1.ResourceMemory: resource.MustParse("2Gi"),
							},
						},
					},
				},
			},
			SeedJobs: []v1alpha2.SeedJob{
				{
					ID:                    "jenkins-operator-e2e",
					JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
					Targets:               "cicd/jobs/*.jenkins",
					Description:           "Jenkins Operator e2e tests repository",
					RepositoryBranch:      "master",
					RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
				},
			},
		},
	}
}

func TestSeedJobs_isRecreatePodNeeded(t *testing.T) {
	seedJobsClient := New(nil, nil, nil)
	t.Run("empty", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.False(t, got)
	})
	t.Run("same", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.False(t, got)
	})
	t.Run("removed one", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name1",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name1", "name2"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.True(t, got)
	})
	t.Run("renamed one", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID: "name1",
					},
					{
						ID: "name3",
					},
				},
			},
			Status: v1alpha2.JenkinsStatus{
				CreatedSeedJobs: []string{"name1", "name2"},
			},
		}

		got := seedJobsClient.isRecreatePodNeeded(jenkins)

		assert.True(t, got)
	})
}
