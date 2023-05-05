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
	t.Run("home volume is present and is Tempdir", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					StorageSettings: v1alpha2.StorageSettings{
						UseTempDir: true,
					},
				},
			},
		}

		HomeExist, HomeTempdirExist, HomeEphemeralStorageExist := checkHomeVolumesPresence(jenkins)

		assert.True(t, HomeExist)
		assert.True(t, HomeTempdirExist)
		assert.False(t, HomeEphemeralStorageExist)
	})
	t.Run("home volume is present and it's ephemeral", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				Master: v1alpha2.JenkinsMaster{
					StorageSettings: v1alpha2.StorageSettings{
						UseEphemeralStorage: true,
						StorageClassName:    "",
						StorageRequest:      "1Gi",
					},
				},
			},
		}

		HomeExist, HomeTempdirExist, HomeEphemeralStorageExist := checkHomeVolumesPresence(jenkins)

		assert.True(t, HomeExist)
		assert.False(t, HomeTempdirExist)
		assert.True(t, HomeEphemeralStorageExist)
	})
}

func checkSecretVolumesPresence(jenkins *v1alpha2.Jenkins) (groovyExists bool, cascExists bool) {
	for _, volume := range GetJenkinsMasterPodBaseVolumes(jenkins) {
		if volume.Name == ("gs-" + jenkins.Spec.GroovyScripts.Secret.Name) {
			groovyExists = true
		} else if volume.Name == ("casc-" + jenkins.Spec.ConfigurationAsCode.Secret.Name) {
			cascExists = true
		}
	}
	return groovyExists, cascExists
}

func checkHomeVolumesPresence(jenkins *v1alpha2.Jenkins) (HomeExist bool, HomeTempdirExist bool, HomeEphemeralStorageExist bool) {
	for _, volume := range GetJenkinsMasterPodBaseVolumes(jenkins) {
		if volume.Name == ("jenkins-home") {
			HomeExist = true
			// check if the volume is an emptyDir
			if volume.VolumeSource.EmptyDir != nil {
				HomeTempdirExist = true
			} else if volume.VolumeSource.Ephemeral != nil {
				HomeEphemeralStorageExist = true
			}
		} else {
			HomeExist = false
			HomeTempdirExist = false
			HomeEphemeralStorageExist = false
		}
	}
	return HomeExist, HomeTempdirExist, HomeEphemeralStorageExist
}

func Test_validateStorageSize(t *testing.T) {
	type args struct {
		storageSize string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1Gi",
			args: args{
				storageSize: "1Gi",
			},
			want: true,
		},
		{
			name: "1Gi1",
			args: args{
				storageSize: "1Gi1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateStorageSize(tt.args.storageSize); got != tt.want {
				t.Errorf("validateStorageSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
