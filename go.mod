module github.com/jenkinsci/kubernetes-operator

go 1.15

require (
	github.com/bndr/gojenkins v1.1.0
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible
	github.com/emersion/go-smtp v0.21.1
	github.com/go-logr/logr v1.4.1
	github.com/go-logr/zapr v1.3.0
	github.com/golang/mock v1.6.0
	github.com/imdario/mergo v0.3.10 // indirect
	github.com/mailgun/mailgun-go/v3 v3.6.4
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.32.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.4
	go.uber.org/zap v1.26.0
	golang.org/x/crypto v0.21.0
	golang.org/x/mod v0.14.0
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	k8s.io/api v0.29.4
	k8s.io/apimachinery v0.29.4
	k8s.io/cli-runtime v0.29.4
	k8s.io/client-go v0.29.4
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b
	sigs.k8s.io/controller-runtime v0.17.3

)
