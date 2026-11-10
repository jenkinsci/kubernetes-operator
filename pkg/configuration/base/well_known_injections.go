package base

// addWellKnownInjections extends a private policy snapshot, never the Jenkins spec.
func (p *podComparisonPolicy) addWellKnownInjections() {
	p.env.Insert(
		"AWS_ROLE_ARN", "AWS_WEB_IDENTITY_TOKEN_FILE", "AWS_REGION", "AWS_DEFAULT_REGION", "AWS_STS_REGIONAL_ENDPOINTS",
		"AWS_CONTAINER_CREDENTIALS_FULL_URI", "AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE",
		"AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_AUTHORITY_HOST", "AZURE_FEDERATED_TOKEN_FILE",
		"GOOGLE_APPLICATION_CREDENTIALS",
	)
	p.volumes.Insert("aws-iam-token", "eks-pod-identity-token", "azure-identity-token")
	p.volumePrefixes = append(p.volumePrefixes, "azure-workload-identity-reserved-")
}
