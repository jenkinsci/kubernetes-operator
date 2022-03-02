package seedjobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkinsci/kubernetes-operator/api/v1alpha2"

	stackerr "github.com/pkg/errors"
	"github.com/robfig/cron"
	"golang.org/x/crypto/ssh"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// ValidateSeedJobs verify seed jobs configuration
func (s *seedJobs) ValidateSeedJobs(jenkins v1alpha2.Jenkins) ([]string, error) {
	var messages []string

	if msg := s.validateIfIDIsUnique(jenkins.Spec.SeedJobs); len(msg) > 0 {
		messages = append(messages, msg...)
	}

	for _, seedJob := range jenkins.Spec.SeedJobs {
		if len(seedJob.ID) == 0 {
			messages = append(messages, fmt.Sprintf("seedJob `%s` id can't be empty", seedJob.ID))
		}

		if len(seedJob.RepositoryBranch) == 0 {
			messages = append(messages, fmt.Sprintf("seedJob `%s` repository branch can't be empty", seedJob.ID))
		}

		if len(seedJob.RepositoryURL) == 0 {
			messages = append(messages, fmt.Sprintf("seedJob `%s` repository URL branch can't be empty", seedJob.ID))
		}

		if len(seedJob.Targets) == 0 {
			messages = append(messages, fmt.Sprintf("seedJob `%s` targets can't be empty", seedJob.ID))
		}

		if _, ok := v1alpha2.AllowedJenkinsCredentialMap[string(seedJob.JenkinsCredentialType)]; !ok {
			messages = append(messages, fmt.Sprintf("seedJob `%s` unknown credential type", seedJob.ID))
		}

		if (seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType ||
			seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType) && len(seedJob.CredentialID) == 0 {
			messages = append(messages, fmt.Sprintf("seedJob `%s` credential ID can't be empty", seedJob.ID))
		}

		// validate repository url match private key
		if strings.Contains(seedJob.RepositoryURL, "git@") && seedJob.JenkinsCredentialType == v1alpha2.NoJenkinsCredentialCredentialType {
			messages = append(messages, fmt.Sprintf("seedJob `%s` Jenkins credential must be set while using ssh repository url", seedJob.ID))
		}

		if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType ||
			seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType ||
			seedJob.JenkinsCredentialType == v1alpha2.GithubAppCredentialType {
			secret := &v1.Secret{}
			namespaceName := types.NamespacedName{Namespace: jenkins.Namespace, Name: seedJob.CredentialID}
			err := s.Client.Get(context.TODO(), namespaceName, secret)
			if err != nil && apierrors.IsNotFound(err) {
				messages = append(messages, fmt.Sprintf("seedJob `%s` required secret '%s' with Jenkins credential not found", seedJob.ID, seedJob.CredentialID))
			} else if err != nil {
				return nil, stackerr.WithStack(err)
			}

			if seedJob.JenkinsCredentialType == v1alpha2.BasicSSHCredentialType {
				if msg := validateBasicSSHSecret(*secret); len(msg) > 0 {
					for _, m := range msg {
						messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
					}
				}
			}
			if seedJob.JenkinsCredentialType == v1alpha2.UsernamePasswordCredentialType {
				if msg := validateUsernamePasswordSecret(*secret); len(msg) > 0 {
					for _, m := range msg {
						messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
					}
				}
			}
			if seedJob.JenkinsCredentialType == v1alpha2.GithubAppCredentialType {
				if msg := validateGithubAppSecret(*secret); len(msg) > 0 {
					for _, m := range msg {
						messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
					}
				}
			}
		}

		if len(seedJob.BuildPeriodically) > 0 {
			if msg := s.validateSchedule(seedJob, seedJob.BuildPeriodically, "buildPeriodically"); len(msg) > 0 {
				for _, m := range msg {
					messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
				}
			}
		}

		if len(seedJob.PollSCM) > 0 {
			if msg := s.validateSchedule(seedJob, seedJob.PollSCM, "pollSCM"); len(msg) > 0 {
				for _, m := range msg {
					messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
				}
			}
		}

		if seedJob.GitHubPushTrigger {
			if msg := s.validateGitHubPushTrigger(jenkins); len(msg) > 0 {
				for _, m := range msg {
					messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
				}
			}
		}

		if seedJob.BitbucketPushTrigger {
			if msg := s.validateBitbucketPushTrigger(jenkins); len(msg) > 0 {
				for _, m := range msg {
					messages = append(messages, fmt.Sprintf("seedJob `%s` %s", seedJob.ID, m))
				}
			}
		}
	}

	return messages, nil
}

func (s *seedJobs) validateSchedule(job v1alpha2.SeedJob, str string, key string) []string {
	var messages []string
	_, err := cron.Parse(str)
	if err != nil {
		messages = append(messages, fmt.Sprintf("`%s` schedule '%s' is invalid cron spec in `%s`", key, str, job.ID))
	}
	return messages
}

func (s *seedJobs) validateGitHubPushTrigger(jenkins v1alpha2.Jenkins) []string {
	var messages []string
	if err := checkPluginExists(jenkins, "github"); err != nil {
		return append(messages, fmt.Sprintf("githubPushTrigger cannot be enabled: %s", err))
	}
	return messages
}

func (s *seedJobs) validateBitbucketPushTrigger(jenkins v1alpha2.Jenkins) []string {
	var messages []string
	if err := checkPluginExists(jenkins, "bitbucket"); err != nil {
		return append(messages, fmt.Sprintf("bitbucketPushTrigger cannot be enabled: %s", err))
	}
	return messages
}

func checkPluginExists(jenkins v1alpha2.Jenkins, name string) error {
	exists := false
	for _, plugin := range jenkins.Spec.Master.BasePlugins {
		if plugin.Name == name {
			exists = true
		}
	}

	userExists := false
	for _, plugin := range jenkins.Spec.Master.Plugins {
		if plugin.Name == name {
			userExists = true
		}
	}

	if !exists && !userExists {
		return fmt.Errorf("`%s` plugin not installed", name)
	}
	return nil
}

func (s *seedJobs) validateIfIDIsUnique(seedJobs []v1alpha2.SeedJob) []string {
	var messages []string
	ids := map[string]bool{}
	for _, seedJob := range seedJobs {
		if _, found := ids[seedJob.ID]; found {
			messages = append(messages, fmt.Sprintf("'%s' seed job ID is not unique", seedJob.ID))
		}
		ids[seedJob.ID] = true
	}
	return messages
}

func validateBasicSSHSecret(secret v1.Secret) []string {
	var messages []string
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	if len(username) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}

	privateKey, exists := secret.Data[PrivateKeySecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}
	if len(string(privateKey)) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}
	if err := validatePrivateKey(string(privateKey)); err != nil {
		messages = append(messages, fmt.Sprintf("private key '%s' invalid in secret '%s': %s", PrivateKeySecretKey, secret.ObjectMeta.Name, err))
	}

	return messages
}

func validateUsernamePasswordSecret(secret v1.Secret) []string {
	var messages []string
	username, exists := secret.Data[UsernameSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	if len(username) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", UsernameSecretKey, secret.ObjectMeta.Name))
	}
	password, exists := secret.Data[PasswordSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
	}
	if len(password) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", PasswordSecretKey, secret.ObjectMeta.Name))
	}

	return messages
}

func validateGithubAppSecret(secret v1.Secret) []string {
	var messages []string
	appid, exists := secret.Data[AppIDSecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", AppIDSecretKey, secret.ObjectMeta.Name))
	}
	if len(appid) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", AppIDSecretKey, secret.ObjectMeta.Name))
	}
	pkey, exists := secret.Data[PrivateKeySecretKey]
	if !exists {
		messages = append(messages, fmt.Sprintf("required data '%s' not found in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}
	if len(pkey) == 0 {
		messages = append(messages, fmt.Sprintf("required data '%s' is empty in secret '%s'", PrivateKeySecretKey, secret.ObjectMeta.Name))
	}

	return messages
}

func validatePrivateKey(privateKey string) error {
	_, err := ssh.ParseRawPrivateKey([]byte(privateKey))
	if err != nil {
		return stackerr.Wrap(err, "failed to decode key")
	}

	return nil
}
