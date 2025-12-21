package resources

import (
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	"github.com/stretchr/testify/assert"
)

func TestGetJenkinsMasterPodBaseVolumes(t *testing.T) {
	t.Run("casc and groovy script with different configMap names", func(t *testing.T) {
		configMapName := "config-map"
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: configMapName,
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "casc-script",
						},
					},
				},
				GroovyScripts: v1alpha2.GroovyScripts{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: configMapName,
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "groovy-script",
						},
					},
				},
			},
		}

		groovyExists, cascExists := checkSecretVolumesPresence(jenkins)

		assert.True(t, groovyExists)
		assert.True(t, cascExists)
	})
	t.Run("groovy script without secret name", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "casc-scripts",
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "jenkins-secret",
						},
					},
				},
				GroovyScripts: v1alpha2.GroovyScripts{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "groovy-scripts",
							},
						},
					},
				},
			},
		}

		groovyExists, cascExists := checkSecretVolumesPresence(jenkins)

		assert.True(t, cascExists)
		assert.False(t, groovyExists)
	})
	t.Run("casc without secret name", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "casc-scripts",
							},
						},
					},
				},
				GroovyScripts: v1alpha2.GroovyScripts{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "groovy-scripts",
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "jenkins-secret",
						},
					},
				},
			},
		}

		groovyExists, cascExists := checkSecretVolumesPresence(jenkins)

		assert.True(t, groovyExists)
		assert.False(t, cascExists)
	})
	t.Run("casc and groovy script shared secret name", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				ConfigurationAsCode: v1alpha2.ConfigurationAsCode{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "casc-scripts",
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "jenkins-secret",
						},
					},
				},
				GroovyScripts: v1alpha2.GroovyScripts{
					Customization: v1alpha2.Customization{
						Configurations: []v1alpha2.ConfigMapRef{
							{
								Name: "groovy-scripts",
							},
						},
						Secret: v1alpha2.SecretRef{
							Name: "jenkins-secret",
						},
					},
				},
			},
		}

		groovyExists, cascExists := checkSecretVolumesPresence(jenkins)

		assert.True(t, groovyExists)
		assert.True(t, cascExists)
	})
}

func checkSecretVolumesPresence(jenkins *v1alpha2.Jenkins) (groovyExists bool, cascExists bool) {
	for _, volume := range GetJenkinsMasterPodBaseVolumes(jenkins) {
		switch volume.Name {
		case "gs-" + jenkins.Spec.GroovyScripts.Secret.Name:
			groovyExists = true
		case "casc-" + jenkins.Spec.ConfigurationAsCode.Secret.Name:
			cascExists = true
		}
	}
	return groovyExists, cascExists
}
