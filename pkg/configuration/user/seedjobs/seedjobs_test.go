package seedjobs

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration/base/resources"

	"github.com/bndr/gojenkins"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const agentSecret = "test-secret"

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

func TestEnsureSeedJobs(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		// given
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewClientBuilder().Build()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkins := jenkinsCustomResource()
		err = fakeClient.Create(ctx, jenkins)
		assert.NoError(t, err)

		testNode := &gojenkins.Node{
			Raw: &gojenkins.NodeResponse{
				DisplayName: AgentName,
			},
		}

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       jenkins,
		}

		seedJobCreatingScript, err := seedJobCreatingGroovyScript(jenkins.Spec.SeedJobs[0])
		assert.NoError(t, err)

		jenkinsClient.EXPECT().GetNode(AgentName).Return(nil, nil).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(AgentName, 1, "The jenkins-operator generated agent", "/home/jenkins", AgentName).Return(testNode, nil).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(AgentName).Return(agentSecret, nil).AnyTimes()
		jenkinsClient.EXPECT().ExecuteScript(seedJobCreatingScript).AnyTimes()

		seedJobClient := New(jenkinsClient, config)

		// when
		_, err = seedJobClient.EnsureSeedJobs(jenkins)

		// then
		assert.NoError(t, err)

		var agentDeployment appsv1.Deployment
		err = fakeClient.Get(ctx, types.NamespacedName{Namespace: jenkins.Namespace, Name: agentDeploymentName(*jenkins, AgentName)}, &agentDeployment)
		assert.NoError(t, err)
		assert.Equal(t, "jenkins/inbound-agent:4.10-3", agentDeployment.Spec.Template.Spec.Containers[0].Image)
	})

	t.Run("delete agent deployment when no seed jobs", func(t *testing.T) {
		// given
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		jenkins := jenkinsCustomResource()
		jenkins.Spec.SeedJobs = []v1alpha2.SeedJob{}

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewClientBuilder().Build()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       jenkins,
		}

		jenkinsClient.EXPECT().GetNode(AgentName).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(AgentName, 1, "The jenkins-operator generated agent", "/home/jenkins", AgentName).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(AgentName).Return(agentSecret, nil).AnyTimes()

		seedJobsClient := New(jenkinsClient, config)

		err = fakeClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentDeploymentName(*jenkins, AgentName),
				Namespace: jenkins.Namespace,
			},
		})
		assert.NoError(t, err)

		// when
		_, err = seedJobsClient.EnsureSeedJobs(jenkins)

		// then
		assert.NoError(t, err)

		var deployment appsv1.Deployment
		err = fakeClient.Get(ctx, types.NamespacedName{Namespace: jenkins.Namespace, Name: agentDeploymentName(*jenkins, AgentName)}, &deployment)

		assert.True(t, errors.IsNotFound(err), "Agent deployment hasn't been deleted")
	})
}

func TestCreateAgent(t *testing.T) {
	t.Run("don't fail when deployment is already created", func(t *testing.T) {
		// given
		ctrl := gomock.NewController(t)
		ctx := context.TODO()
		defer ctrl.Finish()

		jenkins := jenkinsCustomResource()

		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		fakeClient := fake.NewClientBuilder().Build()
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		assert.NoError(t, err)

		jenkinsClient.EXPECT().GetNode(AgentName).AnyTimes()
		jenkinsClient.EXPECT().CreateNode(AgentName, 1, "The jenkins-operator generated agent", "/home/jenkins", AgentName).AnyTimes()
		jenkinsClient.EXPECT().GetNodeSecret(AgentName).Return(agentSecret, nil).AnyTimes()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobsClient := New(jenkinsClient, config)

		err = fakeClient.Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentDeploymentName(*jenkins, AgentName),
				Namespace: jenkins.Namespace,
			},
		})
		assert.NoError(t, err)

		// when
		err = seedJobsClient.createAgent(jenkinsClient, fakeClient, jenkinsCustomResource(), jenkins.Namespace, AgentName)

		// then
		assert.NoError(t, err)
	})
}

func TestSeedJobs_isRecreatePodNeeded(t *testing.T) {
	config := configuration.Configuration{
		Client:        nil,
		ClientSet:     kubernetes.Clientset{},
		Notifications: nil,
		Jenkins:       &v1alpha2.Jenkins{},
	}
	seedJobsClient := New(nil, config)
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
