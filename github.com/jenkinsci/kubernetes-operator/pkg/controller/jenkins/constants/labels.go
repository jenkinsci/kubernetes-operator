package constants

const (
	// LabelAppKey application Kubernetes label name
	LabelAppKey = "app"
	// LabelAppValue application Kubernetes label value
	LabelAppValue = OperatorName

	// LabelWatchKey Kubernetes label used to enable watch for reconcile loop
	LabelWatchKey = "watch"
	// LabelWatchValue Kubernetes label value to enable watch for reconcile loop
	LabelWatchValue = "true"

	// LabelJenkinsCRKey Kubernetes label name which contains Jenkins CR name
	LabelJenkinsCRKey = "jenkins-cr"
)
