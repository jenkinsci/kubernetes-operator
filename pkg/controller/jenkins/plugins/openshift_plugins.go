package plugins

const (
	loginPlugin  = "openshift-login:1.0.23"
	clientPlugin = "openshift-client:1.0.32"
	syncPlugin   = "openshift-sync:1.0.45"
)

// OpenshiftPluginsList contains Openshift plugins to install by operator.
var openshiftPluginsList = []Plugin{
	Must(New(loginPlugin)),
	Must(New(clientPlugin)),
	Must(New(syncPlugin)),
}

// OpenshiftPlugins returns list of Openshift plugins to install by operator.
func OpenshiftPlugins() []Plugin {
	return openshiftPluginsList
}
