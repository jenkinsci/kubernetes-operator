module github.com/jenkinsci/kubernetes-operator

go 1.15

require (
	github.com/bndr/gojenkins v1.0.1
	github.com/docker/distribution v2.7.1+incompatible
	github.com/emersion/go-smtp v0.11.2
	github.com/go-logr/logr v1.2.3
	github.com/go-logr/zapr v0.2.0
	github.com/golang/mock v1.4.4
	github.com/mailgun/mailgun-go/v3 v3.6.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.27.4
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openshift/api v3.9.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.1
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.1.0
	golang.org/x/mod v0.9.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	k8s.io/api v0.27.1
	k8s.io/apimachinery v0.27.1
	k8s.io/cli-runtime v0.27.1
	k8s.io/client-go v0.27.1
	k8s.io/utils v0.0.0-20230209194617-a36077c30491
	sigs.k8s.io/controller-runtime v0.7.0

)
