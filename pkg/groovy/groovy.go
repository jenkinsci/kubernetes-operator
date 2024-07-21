package groovy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"
	jenkinsclient "github.com/jenkinsci/kubernetes-operator/pkg/client"
	"github.com/jenkinsci/kubernetes-operator/pkg/log"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// Groovy defines API for groovy secrets execution via jenkins job
type Groovy struct {
	k8sClient         k8s.Client
	logger            logr.Logger
	jenkins           *v1alpha2.Jenkins
	jenkinsClient     jenkinsclient.Jenkins
	configurationType string
	customization     v1alpha2.Customization
}

// New creates new instance of Groovy
func New(jenkinsClient jenkinsclient.Jenkins, k8sClient k8s.Client, jenkins *v1alpha2.Jenkins, configurationType string, customization v1alpha2.Customization) *Groovy {
	return &Groovy{
		jenkinsClient:     jenkinsClient,
		k8sClient:         k8sClient,
		jenkins:           jenkins,
		configurationType: configurationType,
		customization:     customization,
		logger:            log.Log.WithValues("cr", jenkins.Name),
	}
}

// EnsureSingle runs single groovy script
func (g *Groovy) EnsureSingle(source, name, hash, groovyScript string) (requeue bool, err error) {
	if g.isGroovyScriptAlreadyApplied(source, name, hash) {
		return false, nil
	}

	logs, err := g.jenkinsClient.ExecuteScript(groovyScript)
	if err != nil {
		if groovyErr, ok := err.(*jenkinsclient.GroovyScriptExecutionFailed); ok {
			groovyErr.ConfigurationType = g.configurationType
			groovyErr.Name = name
			groovyErr.Source = source
			groovyErr.Logs = logs
			g.logger.V(log.VWarn).Info(fmt.Sprintf("%s Source '%s' Name '%s' groovy script execution failed, logs :\n%s", g.configurationType, source, name, logs))
		}
		return true, err
	}

	appliedGroovyScripts := []v1alpha2.AppliedGroovyScript{}

	for _, ags := range g.jenkins.Status.AppliedGroovyScripts {
		if g.configurationType == ags.ConfigurationType && ags.Source == source && ags.Name == name {
			continue
		}

		appliedGroovyScripts = append(appliedGroovyScripts, ags)
	}
	appliedGroovyScripts = append(appliedGroovyScripts, v1alpha2.AppliedGroovyScript{
		ConfigurationType: g.configurationType,
		Source:            source,
		Name:              name,
		Hash:              hash,
	})

	g.jenkins.Status.AppliedGroovyScripts = appliedGroovyScripts

	return true, g.k8sClient.Status().Update(context.TODO(), g.jenkins)
}

// WaitForSecretSynchronization runs groovy script which waits to synchronize secrets in pod by k8s
func (g *Groovy) WaitForSecretSynchronization(secretsPath string) (requeue bool, err error) {
	if len(g.customization.Secret.Name) == 0 {
		return false, nil
	}

	secret := &corev1.Secret{}
	err = g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: g.customization.Secret.Name, Namespace: g.jenkins.ObjectMeta.Namespace}, secret)
	if err != nil {
		return true, errors.WithStack(err)
	}

	toCalculate := map[string]string{}
	for secretKey, secretValue := range secret.Data {
		toCalculate[secretKey] = string(secretValue)
	}
	hash, err := g.calculateHash(toCalculate)
	if err != nil {
		return true, errors.WithStack(err)
	}

	name := "synchronizing-secret.groovy"
	if g.isGroovyScriptAlreadyApplied(g.customization.Secret.Name, name, hash) {
		return false, nil
	}

	g.logger.Info(fmt.Sprintf("%s Secret '%s' running synchronization", g.configurationType, secret.Name))
	return g.EnsureSingle(g.customization.Secret.Name, name, hash, fmt.Sprintf(synchronizeSecretsGroovyScriptFmt, secretsPath, hash))
}

// Ensure runs all groovy scripts configured in customization structure
func (g *Groovy) Ensure(filter func(name string) bool, updateGroovyScript func(groovyScript string) string) (requeue bool, err error) {
	secret := &corev1.Secret{}
	if len(g.customization.Secret.Name) > 0 {
		err := g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: g.customization.Secret.Name, Namespace: g.jenkins.ObjectMeta.Namespace}, secret)
		if err != nil {
			return true, err
		}
	}

	for _, configMapRef := range g.customization.Configurations {
		configMap := &corev1.ConfigMap{}
		err := g.k8sClient.Get(context.TODO(), types.NamespacedName{Name: configMapRef.Name, Namespace: g.jenkins.ObjectMeta.Namespace}, configMap)
		if err != nil {
			return true, errors.WithStack(err)
		}

		var names []string
		for name := range configMap.Data {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			groovyScript := updateGroovyScript(configMap.Data[name])
			if !filter(name) {
				g.logger.V(log.VDebug).Info(fmt.Sprintf("Skipping %s ConfigMap '%s' name '%s'", g.configurationType, configMap.Name, name))
				continue
			}

			hash, err := g.calculateCustomizationHash(*secret, name, groovyScript)
			if err != nil {
				return true, errors.WithStack(err)
			}
			if g.isGroovyScriptAlreadyApplied(configMap.Name, name, hash) {
				continue
			}

			g.logger.Info(fmt.Sprintf("%s ConfigMap '%s' name '%s' running groovy script", g.configurationType, configMap.Name, name))
			requeue, err := g.EnsureSingle(configMap.Name, name, hash, groovyScript)
			if err != nil || requeue {
				return requeue, err
			}
		}
	}

	return false, nil
}

func (g *Groovy) calculateCustomizationHash(secret corev1.Secret, key, groovyScript string) (string, error) {
	toCalculate := map[string]string{}
	for secretKey, secretValue := range secret.Data {
		toCalculate[secretKey] = string(secretValue)
	}
	toCalculate[key] = groovyScript
	return g.calculateHash(toCalculate)
}

func (g *Groovy) isGroovyScriptAlreadyApplied(source, name, hash string) bool {
	for _, appliedGroovyScript := range g.jenkins.Status.AppliedGroovyScripts {
		if appliedGroovyScript.ConfigurationType == g.configurationType && appliedGroovyScript.Hash == hash &&
			appliedGroovyScript.Name == name && appliedGroovyScript.Source == source {
			return true
		}
	}

	return false
}

func (g *Groovy) calculateHash(data map[string]string) (string, error) {
	hash := sha256.New()

	keys := []string{}
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, err := hash.Write([]byte(key))
		if err != nil {
			return "", err
		}
		_, err = hash.Write([]byte(data[key]))
		if err != nil {
			return "", err
		}
	}
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// AddSecretsLoaderToGroovyScript modify groovy scripts to load Kubernetes secrets into groovy map
func AddSecretsLoaderToGroovyScript(secretsPath string) func(groovyScript string) string {
	return func(groovyScript string) string {
		if !strings.HasPrefix(groovyScript, importPrefix) {
			return fmt.Sprintf(secretsLoaderGroovyScriptFmt, secretsPath) + groovyScript
		}

		lines := strings.Split(groovyScript, "\n")
		importIndex := -1
		for i, line := range lines {
			if !strings.HasPrefix(line, importPrefix) {
				importIndex = i
				break
			}
		}

		return strings.Join(lines[:importIndex], "\n") + "\n\n" + fmt.Sprintf(secretsLoaderGroovyScriptFmt, secretsPath) + "\n\n" + strings.Join(lines[importIndex:], "\n")
	}
}

const importPrefix = "import "

const secretsLoaderGroovyScriptFmt = `def secretsPath = '%s'
def secrets = [:]
"ls ${secretsPath}".execute().text.eachLine {secrets[it] = new File("${secretsPath}/${it}").text}`

const synchronizeSecretsGroovyScriptFmt = `
def secretsPath = '%s'
def expectedHash = '%s'

println "Synchronizing Kubernetes Secret to the Jenkins master pod, timeout 60 seconds."

def complete = false
for(int i = 1; i <= 30; i++) {
    def fileList = "ls ${secretsPath}".execute()
    def secrets = []
    fileList .text.eachLine {secrets.add(it)}
    println "Mounted secrets: ${secrets}"
    def actualHash = calculateHash((String[])secrets, secretsPath)
    println "Expected hash '${expectedHash}', actual hash '${actualHash}', will retry"
    if(expectedHash == actualHash) {
        complete = true
        break
    }
    sleep 2000
}
if(!complete) {
    throw new Exception("Timeout while synchronizing files")
}

def calculateHash(String[] secrets, String secretsPath) {
    def hash = java.security.MessageDigest.getInstance("SHA-256")
    for(secret in secrets) {
        hash.update(secret.getBytes())
        def fileLocation = java.nio.file.Paths.get("${secretsPath}/${secret}")
        def fileData = java.nio.file.Files.readAllBytes(fileLocation)
        hash.update(fileData)
    }
    return Base64.getEncoder().encodeToString(hash.digest())
}
`
