package seedjobs

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	"github.com/jenkinsci/kubernetes-operator/pkg/configuration"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var fakeRSAPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArK4ld6i2iqW6L3jaTZaKD/v7PjDn+Ik9MXp+kvLcUw/+wEGm
285UwqLnDDlBhSi9nDgJ+m1XU87VCpz/DXW23R/CQcMX2xunib4wWLQqoR3CWbk3
SwiLd8TWAvXkxdXm8fDOGAZbYK2alMV+M+9E2OpZsBUCxmb/3FAofF6JccKoJOH8
UveRNSOx7IXPKtHFiypBhWM4l6ZjgJKm+DRIEhyvoC+pHzcum2ZEPOv+ZJDy5jXK
ZHcNQXVnAZtCcojcjVUBw2rZms+fQ6Volv2JT71Gpykzx/rChhwNwxdAEwjLjKjL
nBWEh/WxsS3NbM7zb4B2XGMCeWVeb/niUwpy+wIDAQABAoIBAQCjGkJNidARmYQI
/u/DxWNWwb2H+o3BFW/1YixYBIjS9BK96cT/bR5mUZRG2XXnnpmqCsxx/AE2KfDU
e4H1ZrB4oFzN3MaVsMNIuZnUzyhM0l0WfnmZp9KEKCm01ilmLCpdcARacPaylIej
6f7QcznmYUShqtbaK8OUhyoWfvz3s0VLkpBlqm63uPtjAx6sAl399THxHVwbYgYy
TxPY8wdjOvNzQJ7ColUh05Zq6TsCGGFUFg7v4to/AXtDhcTMVONlapP+XxekRx8P
98BepIgzgvQhWak8gm+cKQYANk14Q8BDzUCDplYuIZVvKl+/ZHltjHGjrqxDrcDA
0U7REgtxAoGBAN+LAEf2o14ffs/ebVSxiv7LnuAxFh2L6i7RqtehpSf7BnYC65vB
6TMsc/0/KFkD5Az7nrJmA7HmM8J/NI2ks0Mbft+0XCRFx/zfU6pOvPinRKp/8Vtm
aUmNzhz8UMaQ1JXOvBOqvXKWYrN1WPha1+/BnUQrpTdhGxAoAh1FW4eHAoGBAMXA
mXTN5X8+mp9KW2bIpFsjrZ+EyhxO6a6oBMZY54rceeOzf5RcXY7EOiTrnmr+lQvp
fAKBeX5V8G96nSEIDmPhKGZ1C1vEP6hRWahJo1XkN5E1j6hRHCu3DQLtL2lxlyfG
Fx11fysgmLoPVVytLAEQwt4WxMp7OsM1NWqB+u3tAoGBAILUg3Gas7pejIV0FGDB
GCxPV8i2cc8RGBoWs/pHrLVdgUaIJwSd1LISjj/lOuP+FvZSPWsDsZ3osNpgQI21
mwTnjrW2hUblYEprGjhOpOKSYum2v7dSlMRrrfng4hWUphaXTBPmlcH+qf2F7HBO
GptDoZtIQAXNW111TOd8tDj5AoGAC1PO9nvcy38giENQHQEdOQNALMUEdr6mcBS7
wUjSaofai4p6olrwGP9wfTDp8CMJEpebPOGBvhTaIuiZG41ElcAN+mB1+Bmzs8aF
JjihnIfoDu9MfU24GWDw49wGPTn+eI7GQC+8yxGg7fd24kohHSaCowoW16pbYVco
6iLr5rkCgYBt0bcYJ3AOTH0UXS8kvJvnyce/RBIAMoUABwvdkZt9r5B4UzsoLq5e
WrrU6fSRsE6lSsBd83pOAQ46tv+vntQ+0EihD9/0INhkQM99lBw1TFdFTgGSAs1e
ns4JGP6f5uIuwqu/nbqPqMyDovjkGbX2znuGBcvki90Pi97XL7MMWw==
-----END RSA PRIVATE KEY-----`

var fakeRSAInvalidPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArK4ld6i2iqW6L3jaTZaKD/v7PjDn+Ik9MXp+kvLcUw/+wEGm
285UwqLnDDlBhSi9nDgJ+m1XU87VCpz/DXW23R/CQcMX2xunib4wWLQqoR3CWbk3
SwiLd8TWAvXkxdXm8fDOGAZbYK2alMV+M+9E2OpZsBUCxmb/3FAofF6JccKoJOH8
`

var fakeEd25519PrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBXTVD0xWTOJhzVeznd3KUtJ4bSFwpHdk38qUwjfW4VxQAAAJiprsLlqa7C
5QAAAAtzc2gtZWQyNTUxOQAAACBXTVD0xWTOJhzVeznd3KUtJ4bSFwpHdk38qUwjfW4VxQ
AAAECiziLDuLDl5Xt+/WII77eTkUuOhRZreN6ZIqdUFqfokldNUPTFZM4mHNV7Od3cpS0n
htIXCkd2TfypTCN9bhXFAAAAD3ByeWJhQFZMLUQtMDg5MAECAwQFBg==
-----END OPENSSH PRIVATE KEY-----
`

var fakeEd25519InvalidPrivateKey = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACBXTVD0xWTOJhzVeznd3KUtJ4bSFwpHdk38qUwjfW4VxQAAAJiprsLlqa7C
5QAAAAtzc2gtZWQyNTUxOQAAACBXTVD0xWTOJhzVeznd3KUtJ4bSFwpHdk38qUwjfW4VxQ
AAAECiziLDuLDl5Xt+/WII77eTkUuOhRZreN6ZIqdUFqfokldNUPTFZM4mHNV7Od3cpS0n
-----END OPENSSH PRIVATE KEY-----
`

func TestValidateSeedJobs(t *testing.T) {
	secretTypeMeta := metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	secretObjectMeta := metav1.ObjectMeta{
		Name:      "deploy-keys",
		Namespace: "default",
	}
	jenkinsObjectMeta := metav1.ObjectMeta{
		Name:      "cr",
		Namespace: "default",
	}
	t.Run("Valid with public repository and without private key", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid without id", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `` id can't be empty"})
	})
	t.Run("Valid with ed25519 private key and secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey:   []byte("username"),
				PrivateKeySecretKey: []byte(fakeEd25519PrivateKey),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid ed25519 private key in secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey:   []byte("username"),
				PrivateKeySecretKey: []byte(fakeEd25519InvalidPrivateKey),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` private key 'privateKey' invalid in secret 'deploy-keys': failed to decode key: ssh: short read"})
	})
	t.Run("Valid with RSA private key and secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey:   []byte("username"),
				PrivateKeySecretKey: []byte(fakeRSAPrivateKey),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid RSA private key in secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey:   []byte("username"),
				PrivateKeySecretKey: []byte(fakeRSAInvalidPrivateKey),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` private key 'privateKey' invalid in secret 'deploy-keys': failed to decode key: ssh: no key found"})
	})
	t.Run("Invalid with PrivateKey and empty Secret data", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey:   []byte("username"),
				PrivateKeySecretKey: []byte(""),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'privateKey' not found in secret 'deploy-keys'", "seedJob `example` private key 'privateKey' invalid in secret 'deploy-keys': failed to decode key: ssh: no key found"})
	})
	t.Run("Invalid with ssh RepositoryURL and empty PrivateKey", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.BasicSSHCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "git@github.com:jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required secret 'jenkins-operator-e2e' with Jenkins credential not found", "seedJob `example` required data 'username' not found in secret ''", "seedJob `example` required data 'username' is empty in secret ''", "seedJob `example` required data 'privateKey' not found in secret ''", "seedJob `example` required data 'privateKey' not found in secret ''", "seedJob `example` private key 'privateKey' invalid in secret '': failed to decode key: ssh: no key found"})
	})
	t.Run("Invalid without targets", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` targets can't be empty"})
	})
	t.Run("Invalid without repository URL", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` repository URL branch can't be empty"})
	})
	t.Run("Invalid without repository branch", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` repository branch can't be empty"})
	})
	t.Run("Valid with username and password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.UsernamePasswordCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey: []byte("some-username"),
				PasswordSecretKey: []byte("some-password"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid with empty username", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.UsernamePasswordCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey: []byte(""),
				PasswordSecretKey: []byte("some-password"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'username' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid with empty password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.UsernamePasswordCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey: []byte("some-username"),
				PasswordSecretKey: []byte(""),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'password' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid without username", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.UsernamePasswordCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				PasswordSecretKey: []byte("some-password"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'username' not found in secret 'deploy-keys'", "seedJob `example` required data 'username' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid without password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.UsernamePasswordCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				UsernameSecretKey: []byte("some-username"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'password' not found in secret 'deploy-keys'", "seedJob `example` required data 'password' is empty in secret 'deploy-keys'"})
	})
	t.Run("Valid with appId and privateKey", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.GithubAppCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				AppIDSecretKey: []byte("some-id"),
				PrivateKeySecretKey: []byte("some-key"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid with empty app id", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.GithubAppCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				AppIDSecretKey: []byte(""),
				PrivateKeySecretKey: []byte("some-key"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'appId' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid with empty private key", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.GithubAppCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				AppIDSecretKey: []byte("some-id"),
				PrivateKeySecretKey: []byte(""),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'privateKey' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid without app id", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.GithubAppCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				PrivateKeySecretKey: []byte("some-key"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'appId' not found in secret 'deploy-keys'", "seedJob `example` required data 'appId' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid without private key", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			ObjectMeta: jenkinsObjectMeta,
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "deploy-keys",
						JenkinsCredentialType: v1alpha2.GithubAppCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
					},
				},
			},
		}
		secret := &corev1.Secret{
			TypeMeta:   secretTypeMeta,
			ObjectMeta: secretObjectMeta,
			Data: map[string][]byte{
				AppIDSecretKey: []byte("some-username"),
			},
		}
		fakeClient := fake.NewClientBuilder().Build()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` required data 'privateKey' not found in secret 'deploy-keys'", "seedJob `example` required data 'privateKey' is empty in secret 'deploy-keys'"})
	})
	t.Run("Invalid with wrong cron spec", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						BuildPeriodically:     "invalid-cron-spec",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` `buildPeriodically` schedule 'invalid-cron-spec' is invalid cron spec in `example`"})
	})
	t.Run("Valid with good cron spec", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						BuildPeriodically:     "1 2 3 4 5",
						PollSCM:               "1 2 3 4 5",
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid with set githubPushTrigger and not installed github plugin", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						GitHubPushTrigger:     true,
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` githubPushTrigger cannot be enabled: `github` plugin not installed"})
	})
	t.Run("Valid with set githubPushTrigger and installed github plugin", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						GitHubPushTrigger:     true,
					},
				},
				Master: v1alpha2.JenkinsMaster{
					Plugins: []v1alpha2.Plugin{
						{Name: "github", Version: "latest"},
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
	t.Run("Invalid with set bitbucketPushTrigger and not installed bitbucket plugin", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						BitbucketPushTrigger:  true,
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)

		assert.Equal(t, result, []string{"seedJob `example` bitbucketPushTrigger cannot be enabled: `bitbucket` plugin not installed"})
	})
	t.Run("Valid with set bitbucketPushTrigger and installed Bitbucket plugin", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
			Spec: v1alpha2.JenkinsSpec{
				SeedJobs: []v1alpha2.SeedJob{
					{
						ID:                    "example",
						CredentialID:          "jenkins-operator-e2e",
						JenkinsCredentialType: v1alpha2.NoJenkinsCredentialCredentialType,
						Targets:               "cicd/jobs/*.jenkins",
						RepositoryBranch:      "master",
						RepositoryURL:         "https://github.com/jenkinsci/kubernetes-operator.git",
						BitbucketPushTrigger:  true,
					},
				},
				Master: v1alpha2.JenkinsMaster{
					Plugins: []v1alpha2.Plugin{
						{Name: "bitbucket", Version: "latest"},
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().Build()

		config := configuration.Configuration{
			Client:        fakeClient,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		seedJobs := New(nil, config)
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestValidateIfIDIsUnique(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		seedJobs := []v1alpha2.SeedJob{
			{ID: "first"}, {ID: "second"},
		}

		config := configuration.Configuration{
			Client:        nil,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		ctrl := New(nil, config)
		got := ctrl.validateIfIDIsUnique(seedJobs)
		assert.Nil(t, got)
	})
	t.Run("duplicated ids", func(t *testing.T) {
		seedJobs := []v1alpha2.SeedJob{
			{ID: "first"}, {ID: "first"},
		}

		config := configuration.Configuration{
			Client:        nil,
			ClientSet:     kubernetes.Clientset{},
			Notifications: nil,
			Jenkins:       &v1alpha2.Jenkins{},
		}

		ctrl := New(nil, config)
		got := ctrl.validateIfIDIsUnique(seedJobs)

		assert.Equal(t, got, []string{"'first' seed job ID is not unique"})
	})
}
