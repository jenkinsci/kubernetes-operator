package groovy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const configurationType = "test-conf-type"

func TestGroovy_EnsureSingle(t *testing.T) {
	t.Skip("TODO: fix me, skipping the test suite temporarily")
	log.SetupLogger(true)
	emptyCustomization := v1alpha2.Customization{}
	hash := "hash"
	groovyScript := "groovy-script"
	groovyScriptName := "groovy-script-name"
	source := "source"
	ctx := context.TODO()
	jenkinsName := "jenkins"
	namespace := "default"

	t.Run("execute script and save status", func(t *testing.T) {
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		// when
		requeue, err := groovyClient.EnsureSingle(source, groovyScriptName, hash, groovyScript)

		// then
		require.NoError(t, err)
		assert.True(t, requeue)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, hash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("no execute script", func(t *testing.T) {
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
			Status: v1alpha2.JenkinsStatus{
				AppliedGroovyScripts: []v1alpha2.AppliedGroovyScript{
					{
						ConfigurationType: configurationType,
						Source:            source,
						Name:              groovyScriptName,
						Hash:              hash,
					},
				},
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		// when
		requeue, err := groovyClient.EnsureSingle(source, groovyScriptName, hash, groovyScript)

		// then
		require.NoError(t, err)
		assert.False(t, requeue)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, hash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("execute script with new version", func(t *testing.T) {
		anotherHash := "hash1"
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)

		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		// when
		requeue, err := groovyClient.EnsureSingle(source, groovyScriptName, hash, groovyScript)

		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, hash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)

		// Update with new hash
		requeue, err = groovyClient.EnsureSingle(source, groovyScriptName, anotherHash, groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, anotherHash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)

		requeue, err = groovyClient.EnsureSingle(source, groovyScriptName, hash, groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, hash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("execute three groovy scripts with another hash", func(t *testing.T) {
		// given
		firstGroovyScriptName := "testGroovy1"
		firstGroovyScriptHash := "testHash1"

		secondGroovyScriptName := "testGroovy2"
		secondGroovyScriptHash := "testHash2"

		thirdGroovyScriptName := "testGroovy3"
		thirdGroovyScriptHash := "testHash3"

		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)

		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		requeue, err := groovyClient.EnsureSingle(source, firstGroovyScriptName, firstGroovyScriptHash, groovyScript)

		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, firstGroovyScriptHash, jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, firstGroovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)

		requeue, err = groovyClient.EnsureSingle(source, secondGroovyScriptName, secondGroovyScriptHash, groovyScript)
		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 2, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[1].ConfigurationType)
		assert.Equal(t, secondGroovyScriptHash, jenkins.Status.AppliedGroovyScripts[1].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[1].Source)
		assert.Equal(t, secondGroovyScriptName, jenkins.Status.AppliedGroovyScripts[1].Name)

		requeue, err = groovyClient.EnsureSingle(source, thirdGroovyScriptName, thirdGroovyScriptHash, groovyScript)
		// then
		require.NoError(t, err)
		assert.True(t, requeue)
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 3, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[2].ConfigurationType)
		assert.Equal(t, thirdGroovyScriptHash, jenkins.Status.AppliedGroovyScripts[2].Hash)
		assert.Equal(t, source, jenkins.Status.AppliedGroovyScripts[2].Source)
		assert.Equal(t, thirdGroovyScriptName, jenkins.Status.AppliedGroovyScripts[2].Name)
	})
	t.Run("execute two groovy scripts with same names in two config maps", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)

		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		requeue, err := groovyClient.EnsureSingle("test-conf1", "test.groovy", hash, groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		requeue, err = groovyClient.EnsureSingle("test-conf2", "test.groovy", "anotherHash", groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		assert.Equal(t, 2, len(jenkins.Status.AppliedGroovyScripts))
	})
	t.Run("execute two groovy scripts with different configuration types", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		requeue, err := groovyClient.EnsureSingle(source, "test.groovy", hash, groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		groovyClient = New(jenkinsClient, fakeClient, jenkins, "another-test-configuration-type", emptyCustomization)

		requeue, err = groovyClient.EnsureSingle(source, "test.groovy", "anotherHash", groovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)

		assert.Equal(t, 2, len(jenkins.Status.AppliedGroovyScripts))
	})
	t.Run("execute script fails", func(t *testing.T) {
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("fail logs", &jenkinsclient.GroovyScriptExecutionFailed{})

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, emptyCustomization)

		// when
		requeue, err := groovyClient.EnsureSingle(source, groovyScriptName, hash, groovyScript)

		// then
		require.Error(t, err)
		assert.True(t, requeue)

		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 0, len(jenkins.Status.AppliedGroovyScripts))
	})
}

func TestGroovy_Ensure(t *testing.T) {
	t.Skip("TODO: fix me, skipping the test suite temporarily")
	log.SetupLogger(true)
	groovyScript := "groovy-script"
	groovyScriptName := "groovy-script-name.groovy"
	ctx := context.TODO()
	jenkinsName := "jenkins"
	namespace := "default"
	configMapName := "config-map-name"
	secretName := "secret-name"

	allGroovyScriptsFunc := func(name string) bool {
		return true
	}
	noUpdateGroovyScript := func(groovyScript string) string {
		return groovyScript
	}

	t.Run("select groovy files with .groovy extension", func(t *testing.T) {
		// given
		groovyScriptExtension := ".groovy"
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		customization := v1alpha2.Customization{
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: configMapName,
				},
			},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				groovyScriptName: groovyScript,
				"to-omit":        "to-omit",
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)
		err = fakeClient.Create(ctx, configMap)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, customization)
		onlyGroovyFilesFunc := func(name string) bool {
			return strings.HasSuffix(name, groovyScriptExtension)
		}

		// when
		requeue, err := groovyClient.Ensure(onlyGroovyFilesFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)
		requeue, err = groovyClient.Ensure(onlyGroovyFilesFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.False(t, requeue)

		// then
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, "qoXeeh4ia+KXhT01lYNxe+oxByDf8dfT2npP9fgzjbk=", jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, configMapName, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("change groovy script", func(t *testing.T) {
		// given
		groovyScriptSuffix := "suffix"
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		customization := v1alpha2.Customization{
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: configMapName,
				},
			},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				groovyScriptName: groovyScript,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)
		err = fakeClient.Create(ctx, configMap)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript+groovyScriptSuffix).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, customization)
		updateGroovyFunc := func(groovyScript string) string {
			return groovyScript + groovyScriptSuffix
		}

		// when
		requeue, err := groovyClient.Ensure(allGroovyScriptsFunc, updateGroovyFunc)
		require.NoError(t, err)
		assert.True(t, requeue)
		requeue, err = groovyClient.Ensure(allGroovyScriptsFunc, updateGroovyFunc)
		require.NoError(t, err)
		assert.False(t, requeue)

		// then
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, "TgTpV3nDxMNMM93t6jgni0UHa7C+uL+D+BLcW3a7b6M=", jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, configMapName, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("execute script without secret and save status", func(t *testing.T) {
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		customization := v1alpha2.Customization{
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: configMapName,
				},
			},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				groovyScriptName: groovyScript,
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)
		err = fakeClient.Create(ctx, configMap)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, customization)

		// when
		requeue, err := groovyClient.Ensure(allGroovyScriptsFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)
		requeue, err = groovyClient.Ensure(allGroovyScriptsFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.False(t, requeue)

		// then
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, "qoXeeh4ia+KXhT01lYNxe+oxByDf8dfT2npP9fgzjbk=", jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, configMapName, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
	t.Run("execute script with secret and save status", func(t *testing.T) {
		// given
		jenkins := &v1alpha2.Jenkins{
			ObjectMeta: metav1.ObjectMeta{
				Name:      jenkinsName,
				Namespace: namespace,
			},
		}
		customization := v1alpha2.Customization{
			Secret: v1alpha2.SecretRef{Name: secretName},
			Configurations: []v1alpha2.ConfigMapRef{
				{
					Name: configMapName,
				},
			},
		}
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: namespace,
			},
			Data: map[string]string{
				groovyScriptName: groovyScript,
			},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"SECRET_KEY": []byte("secret-value"),
			},
		}
		err := v1alpha2.SchemeBuilder.AddToScheme(scheme.Scheme)
		require.NoError(t, err)
		fakeClient := fake.NewClientBuilder().Build()
		err = fakeClient.Create(ctx, jenkins)
		require.NoError(t, err)
		err = fakeClient.Create(ctx, secret)
		require.NoError(t, err)
		err = fakeClient.Create(ctx, configMap)
		require.NoError(t, err)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		jenkinsClient := jenkinsclient.NewMockJenkins(ctrl)
		jenkinsClient.EXPECT().ExecuteScript(groovyScript).Return("logs", nil)

		groovyClient := New(jenkinsClient, fakeClient, jenkins, configurationType, customization)

		// when
		requeue, err := groovyClient.Ensure(allGroovyScriptsFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.True(t, requeue)
		requeue, err = groovyClient.Ensure(allGroovyScriptsFunc, noUpdateGroovyScript)
		require.NoError(t, err)
		assert.False(t, requeue)

		// then
		err = fakeClient.Get(ctx, types.NamespacedName{Name: jenkins.Name, Namespace: jenkins.Namespace}, jenkins)
		require.NoError(t, err)
		assert.Equal(t, 1, len(jenkins.Status.AppliedGroovyScripts))
		assert.Equal(t, configurationType, jenkins.Status.AppliedGroovyScripts[0].ConfigurationType)
		assert.Equal(t, "em9pjw9mUheUpPRCJWD2Dww+80YQPoHCZbzzKZZw4lo=", jenkins.Status.AppliedGroovyScripts[0].Hash)
		assert.Equal(t, configMapName, jenkins.Status.AppliedGroovyScripts[0].Source)
		assert.Equal(t, groovyScriptName, jenkins.Status.AppliedGroovyScripts[0].Name)
	})
}

func TestGroovy_isGroovyScriptAlreadyApplied(t *testing.T) {
	t.Skip("TODO: fix me, skipping the test suite temporarily")
	log.SetupLogger(true)
	emptyCustomization := v1alpha2.Customization{}

	t.Run("found", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: v1alpha2.JenkinsStatus{
				AppliedGroovyScripts: []v1alpha2.AppliedGroovyScript{
					{
						ConfigurationType: configurationType,
						Source:            "source",
						Name:              "name",
						Hash:              "hash",
					},
				},
			},
		}
		groovyClient := New(nil, nil, jenkins, configurationType, emptyCustomization)

		got := groovyClient.isGroovyScriptAlreadyApplied("source", "name", "hash")

		assert.True(t, got)
	})
	t.Run("not found", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{
			Status: v1alpha2.JenkinsStatus{
				AppliedGroovyScripts: []v1alpha2.AppliedGroovyScript{
					{
						ConfigurationType: configurationType,
						Source:            "source",
						Name:              "name",
						Hash:              "hash",
					},
				},
			},
		}
		groovyClient := New(nil, nil, jenkins, configurationType, emptyCustomization)

		got := groovyClient.isGroovyScriptAlreadyApplied("source", "not-exist", "hash")

		assert.False(t, got)
	})
	t.Run("empty Jenkins status", func(t *testing.T) {
		jenkins := &v1alpha2.Jenkins{}
		groovyClient := New(nil, nil, jenkins, configurationType, emptyCustomization)

		got := groovyClient.isGroovyScriptAlreadyApplied("source", "name", "hash")

		assert.False(t, got)
	})
}

func TestAddSecretsLoaderToGroovyScript(t *testing.T) {
	secretsPath := "/var/jenkins/groovy-scripts-secrets"
	secretsLoader := fmt.Sprintf(secretsLoaderGroovyScriptFmt, secretsPath)

	t.Run("without imports", func(t *testing.T) {
		groovyScript := "println 'Simple groovy script"
		updater := AddSecretsLoaderToGroovyScript(secretsPath)

		got := updater(groovyScript)

		assert.Equal(t, secretsLoader+groovyScript, got)
	})
	t.Run("with imports", func(t *testing.T) {
		groovyScript := `import com.foo.bar
import com.foo.bar2
println 'Simple groovy script'`
		imports := `import com.foo.bar
import com.foo.bar2`
		tail := `println 'Simple groovy script'`
		update := AddSecretsLoaderToGroovyScript(secretsPath)

		got := update(groovyScript)

		assert.Equal(t, imports+"\n\n"+secretsLoader+"\n\n"+tail, got)
	})
	t.Run("with imports and separate section", func(t *testing.T) {
		groovyScript := `import com.foo.bar
import com.foo.bar2

println 'Simple groovy script'`
		imports := `import com.foo.bar
import com.foo.bar2`
		tail := `println 'Simple groovy script'`
		update := AddSecretsLoaderToGroovyScript(secretsPath)

		got := update(groovyScript)

		assert.Equal(t, imports+"\n\n"+secretsLoader+"\n\n\n"+tail, got)
	})
}
