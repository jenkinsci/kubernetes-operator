package seedjobs

import (
	"context"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/pkg/apis/jenkins/v1alpha2"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var fakePrivateKey = `-----BEGIN RSA PRIVATE KEY-----
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

var fakeInvalidPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArK4ld6i2iqW6L3jaTZaKD/v7PjDn+Ik9MXp+kvLcUw/+wEGm
285UwqLnDDlBhSi9nDgJ+m1XU87VCpz/DXW23R/CQcMX2xunib4wWLQqoR3CWbk3
SwiLd8TWAvXkxdXm8fDOGAZbYK2alMV+M+9E2OpZsBUCxmb/3FAofF6JccKoJOH8
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Valid with private key and secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
				PrivateKeySecretKey: []byte(fakePrivateKey),
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
	})
	t.Run("Invalid private key in secret", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
				PrivateKeySecretKey: []byte(fakeInvalidPrivateKey),
			},
		}
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Invalid with PrivateKey and empty Secret data", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
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

		seedJobs := New(nil, fake.NewFakeClient(), logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Valid with username and password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
	})
	t.Run("Invalid with empty username", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Invalid with empty password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Invalid without username", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
	t.Run("Invalid without password", func(t *testing.T) {
		jenkins := v1alpha2.Jenkins{
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
		fakeClient := fake.NewFakeClient()
		err := fakeClient.Create(context.TODO(), secret)
		assert.NoError(t, err)

		seedJobs := New(nil, fakeClient, logf.ZapLogger(false))
		result, err := seedJobs.ValidateSeedJobs(jenkins)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

func TestValidateIfIDIsUnique(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		seedJobs := []v1alpha2.SeedJob{
			{ID: "first"}, {ID: "second"},
		}
		ctrl := New(nil, nil, logf.ZapLogger(false))
		got := ctrl.validateIfIDIsUnique(seedJobs)
		assert.Equal(t, true, got)
	})
	t.Run("duplicated ids", func(t *testing.T) {
		seedJobs := []v1alpha2.SeedJob{
			{ID: "first"}, {ID: "first"},
		}
		ctrl := New(nil, nil, logf.ZapLogger(false))
		got := ctrl.validateIfIDIsUnique(seedJobs)
		assert.Equal(t, false, got)
	})
}
