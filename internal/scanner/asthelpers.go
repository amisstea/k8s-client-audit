package scanner

import "go/ast"

// knownResourceSelectors lists common typed-client resource selector names
// across core, apps, batch and common ecosystem resources (extend as needed).
var knownResourceSelectors = []string{
	"Pods",
	"Deployments",
	"Services",
	"StatefulSets",
	"ConfigMaps",
	"Secrets",
	"Nodes",
	"Namespaces",
	"Events",
	"Jobs",
	"CronJobs",
	"PersistentVolumes",
	"PersistentVolumeClaims",
	"DaemonSets",
	"ReplicaSets",
	// Konflux CRDs
	"ACRAccessToken",
	"AddonInstance",
	"AddonOperator",
	"Addon",
	"AdminNetworkPolicy",
	"AdminPolicyBasedExternalRoute",
	"AdmissionCheck",
	"AlertingRule",
	"AlertmanagerConfig",
	"Alertmanager",
	"AlertRelabelConfig",
	"AnalysisRun",
	"AnalysisTemplate",
	"APIRequestCount",
	"APIServer",
	"ApiServerSource",
	"Application",
	"ApplicationSet",
	"AppProject",
	"ArgoCD",
	"ArtifactBuild",
	"AuthCode",
	"Authentication",
	"AuthRequest",
	"Backstage",
	"BackupRepository",
	"Backup",
	"BackupStorageLocation",
	"BareMetalHost",
	"BaselineAdminNetworkPolicy",
	"BMCEventSubscription",
	"Broker",
	"BuildPipelineSelector",
	"Build",
	"CatalogSource",
	"CertificateRequest",
	"Certificate",
	"CertManager",
	"Challenge",
	"Channel",
	"CleanupPolicy",
	"CloudCredential",
	"CloudPrivateIPConfig",
	"CloudStorage",
	"ClusterAnalysisTemplate",
	"ClusterAutoscaler",
	"ClusterCleanupPolicy",
	"ClusterCSIDriver",
	"ClusterEphemeralReport",
	"ClusterExternalSecret",
	"ClusterGenerator",
	"ClusterInterceptor",
	"ClusterIssuer",
	"ClusterKubeArchiveConfig",
	"ClusterLogForwarder",
	"ClusterLogging",
	"ClusterObjectDeployment",
	"ClusterObjectSetPhase",
	"ClusterObjectSet",
	"ClusterObjectSlice",
	"ClusterObjectTemplate",
	"ClusterOperator",
	"ClusterPackage",
	"ClusterPolicy",
	"ClusterPolicyReport",
	"ClusterQueue",
	"ClusterRegistrar",
	"ClusterResourceQuota",
	"ClusterSecretStore",
	"ClusterServiceVersion",
	"ClusterTask",
	"ClusterTriggerBinding",
	"ClusterUrlMonitor",
	"ClusterVacuumConfig",
	"ClusterVersion",
	"ComponentDetectionQuery",
	"Component",
	"CompositeResourceDefinition",
	"CompositionRevision",
	"Composition",
	"Config",
	"ConfigurationRevision",
	"Configuration",
	"Connector",
	"ConsoleCLIDownload",
	"ConsoleExternalLogLink",
	"ConsoleLink",
	"ConsoleNotification",
	"ConsolePlugin",
	"ConsoleQuickStart",
	"Console",
	"ConsoleSample",
	"ConsoleYAMLSample",
	"ContainerRuntimeConfig",
	"ContainerSource",
	"ControllerConfig",
	"ControlPlaneMachineSet",
	"CostManagementMetricsConfig",
	"CredentialsRequest",
	"CSISnapshotController",
	"CustomDomain",
	"CustomRun",
	"DataDownload",
	"DataImage",
	"DataProtectionApplication",
	"DataUpload",
	"DeleteBackupRequest",
	"DependencyBuild",
	"DependencyUpdateCheck",
	"DeploymentRuntimeConfig",
	"DeploymentTargetClaim",
	"DeploymentTargetClass",
	"DeploymentTarget",
	"DeviceRequest",
	"DeviceToken",
	"DNS",
	"DNSRecord",
	"DownloadRequest",
	"ECRAuthorizationToken",
	"EgressFirewall",
	"EgressIP",
	"EgressQoS",
	"EgressRouter",
	"EgressService",
	"EnterpriseContractPolicy",
	"EnvironmentConfig",
	"Environment",
	"EphemeralReport",
	"Etcd",
	"EventListener",
	"EventPolicy",
	"EventType",
	"Experiment",
	"ExternalSecret",
	"Fake",
	"FeatureGate",
	"FirmwareSchema",
	"FunctionRevision",
	"Function",
	"GCRAccessToken",
	"GithubAccessToken",
	"GitOpsDeploymentManagedEnvironment",
	"GitOpsDeploymentRepositoryCredential",
	"GitOpsDeployment",
	"GitOpsDeploymentSyncRun",
	"GitopsService",
	"GlobalContextEntry",
	"GoTemplate",
	"GrafanaAlertRuleGroup",
	"GrafanaContactPoint",
	"GrafanaDashboard",
	"GrafanaDatasource",
	"GrafanaDataSource",
	"GrafanaFolder",
	"GrafanaLibraryPanel",
	"GrafanaMuteTiming",
	"GrafanaNotificationChannel",
	"GrafanaNotificationPolicy",
	"GrafanaNotificationPolicyRoute",
	"GrafanaNotificationTemplate",
	"Grafana",
	"GroupSync",
	"HardwareData",
	"HelmChartRepository",
	"HostFirmwareComponents",
	"HostFirmwareSettings",
	"HubConfig",
	"Idler",
	"ImageConfig",
	"ImageContentPolicy",
	"ImageContentSourcePolicy",
	"ImageDigestMirrorSet",
	"ImagePruner",
	"ImageRepository",
	"Image",
	"ImageTagMirrorSet",
	"Infrastructure",
	"IngressController",
	"Ingress",
	"InMemoryChannel",
	"InsightsOperator",
	"InstallPlan",
	"IntegrationTestScenario",
	"Interceptor",
	"InternalRequest",
	"InternalServicesConfig",
	"IPAddressClaim",
	"IPAddress",
	"IPPool",
	"Issuer",
	"IstioCSR",
	"JBSConfig",
	"JobSink",
	"JvmImageScan",
	"KeplerInternal",
	"Kepler",
	"KubeAPIServer",
	"KubeArchiveConfig",
	"KubeControllerManager",
	"KubeletConfig",
	"KubeScheduler",
	"KubeStorageVersionMigrator",
	"Kueue",
	"LocalQueue",
	"Lock",
	"LogFileMetricExporter",
	"MachineAutoscaler",
	"MachineConfigNode",
	"MachineConfigPool",
	"MachineConfig",
	"MachineConfiguration",
	"MachineHealthCheck",
	"Machine",
	"MachineSet",
	"ManagedFleetNotificationRecord",
	"ManagedFleetNotification",
	"ManagedNotification",
	"ManualApprovalGate",
	"MemberOperatorConfig",
	"MemberStatus",
	"Metal3Remediation",
	"Metal3RemediationTemplate",
	"MonitoringStack",
	"MultiKueueCluster",
	"MultiKueueConfig",
	"MustGather",
	"Namespace",
	"NamespaceVacuumConfig",
	"NetworkAttachmentDefinition",
	"Network",
	"Node",
	"NSTemplateSet",
	"OAuth2Client",
	"OAuth",
	"ObjectDeployment",
	"Object",
	"ObjectSetPhase",
	"ObjectSet",
	"ObjectSlice",
	"ObjectTemplate",
	"ObservedObjectCollection",
	"OcmAgent",
	"OfflineSessions",
	"OLMConfig",
	"OpenShiftAPIServer",
	"OpenShiftControllerManager",
	"OpenShiftPipelinesAsCode",
	"Operation",
	"OperatorCondition",
	"OperatorConfig",
	"OperatorGroup",
	"OperatorHub",
	"OperatorPKI",
	"Operator",
	"Order",
	"OverlappingRangeIPReservation",
	"Package",
	"Parallel",
	"Password",
	"PerformanceProfile",
	"Perses",
	"PersesDashboard",
	"PersesDatasource",
	"PingSource",
	"PipelineRun",
	"Pipeline",
	"PodMonitor",
	"PodNetworkConnectivityCheck",
	"PodVolumeBackup",
	"PodVolumeRestore",
	"Policy",
	"PolicyException",
	"PolicyReport",
	"PreprovisioningImage",
	"Probe",
	"Profile",
	"ProjectDevelopmentStream",
	"ProjectDevelopmentStreamTemplate",
	"ProjectHelmChartRepository",
	"Project",
	"PrometheusAgent",
	"Prometheus",
	"PrometheusRule",
	"PromotionRun",
	"ProviderConfig",
	"ProviderConfigUsage",
	"ProviderRevision",
	"Provider",
	"ProvisioningRequestConfig",
	"Provisioning",
	"Proxy",
	"PulpAccessRequest",
	"PushSecret",
	"RangeAllocation",
	"RebuiltArtifact",
	"RefreshToken",
	"RegisteredCluster",
	"ReleasePlanAdmission",
	"ReleasePlan",
	"Release",
	"ReleaseServiceConfig",
	"ReleaseStrategy",
	"RemoteSecret",
	"Repository",
	"ResolutionRequest",
	"ResourceFlavor",
	"Resources",
	"ResticRepository",
	"Restore",
	"RoleBindingRestriction",
	"RolloutManager",
	"Rollout",
	"RouteMonitor",
	"Scheduler",
	"Schedule",
	"ScrapeConfig",
	"SecretStore",
	"SecurityContextConstraints",
	"Sequence",
	"ServerStatusRequest",
	"ServiceCA",
	"ServiceMonitor",
	"SharedConfigMap",
	"SharedSecret",
	"SigningKey",
	"SinkBinding",
	"SinkFilter",
	"SnapshotEnvironmentBinding",
	"Snapshot",
	"SpaceBindingRequest",
	"SpaceRequest",
	"SPIAccessCheck",
	"SPIAccessTokenBinding",
	"SPIAccessTokenDataUpdate",
	"SPIAccessToken",
	"SPIFileContentRequest",
	"SplunkForwarder",
	"StepAction",
	"Storage",
	"StorageState",
	"StorageVersionMigration",
	"StoreConfig",
	"STSSessionToken",
	"SubjectPermission",
	"Subscription",
	"SystemConfig",
	"TaskRun",
	"Task",
	"TektonAddon",
	"TektonChain",
	"TektonConfig",
	"TektonHub",
	"TektonInstallerSet",
	"TektonPipeline",
	"TektonPruner",
	"TektonResult",
	"TektonTrigger",
	"TestPlatformCluster",
	"ThanosQuerier",
	"ThanosRuler",
	"ToolchainCluster",
	"TriggerBinding",
	"Trigger",
	"TriggerTemplate",
	"Tuned",
	"UIPlugin",
	"UpdateRequest",
	"UpgradeConfig",
	"Usage",
	"UserAccount",
	"UUID",
	"VaultDynamicSecret",
	"VeleroInstall",
	"VerificationPolicy",
	"VolumeSnapshotBackup",
	"VolumeSnapshotClass",
	"VolumeSnapshotContent",
	"VolumeSnapshotLocation",
	"VolumeSnapshotRestore",
	"VolumeSnapshot",
	"Webhook",
	"WorkloadPriorityClass",
	"Workload",
	"Workspace",
	"XNamespace",
	"XTestPlatformCluster",
}

func isResourceSelector(name string) bool {
	for _, n := range knownResourceSelectors {
		if n == name {
			return true
		}
	}
	return false
}

// hasEmptyStringNamespaceArg returns true if within the selector/call chain
// there is a call expression whose first argument is an empty string literal.
// Intended to catch typed client patterns like Pods("").List(...).
func hasEmptyStringNamespaceArg(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if len(e.Args) > 0 {
			if bl, ok := e.Args[0].(*ast.BasicLit); ok {
				if bl.Value == "\"\"" {
					return true
				}
			}
		}
		// Also dive into the function in case of chained calls
		return hasEmptyStringNamespaceArg(e.Fun)
	case *ast.SelectorExpr:
		return hasEmptyStringNamespaceArg(e.X)
	default:
		return false
	}
}

// isCommonKubeMethod reports whether a method name is commonly used for API operations.
func isCommonKubeMethod(name string) bool {
	switch name {
	case "Get", "List", "Create", "Update", "Patch", "Delete", "Watch":
		return true
	default:
		return false
	}
}

// looksLikeContext returns true if arg appears to be a context (ctx variable or context.Background/TODO()).
func looksLikeContext(arg ast.Expr) bool {
	// Accept calls like context.Background/TODO or identifiers named ctx
	switch a := arg.(type) {
	case *ast.CallExpr:
		if sel, ok := a.Fun.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				if id.Name == "context" && (sel.Sel.Name == "Background" || sel.Sel.Name == "TODO") {
					return true
				}
			}
		}
	case *ast.Ident:
		if a.Name == "ctx" || a.Name == "context" {
			return true
		}
	}
	return false
}

// argsContainKubeOptions checks whether any argument is a known kube list option or controller-runtime option.
func argsContainKubeOptions(args []ast.Expr) bool {
	for _, a := range args {
		switch x := a.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if sel.Sel.Name == "InNamespace" || sel.Sel.Name == "MatchingLabels" || sel.Sel.Name == "MatchingFields" || sel.Sel.Name == "MatchingFieldsSelector" {
					return true
				}
			}
		case *ast.CompositeLit:
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					if ident, ok := kv.Key.(*ast.Ident); ok {
						if ident.Name == "LabelSelector" || ident.Name == "FieldSelector" {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// isLikelyKubeAPICall heuristically determines if a call expression is interacting
// with the Kubernetes API. Works without full type info; best-effort only.
func isLikelyKubeAPICall(call *ast.CallExpr) bool {
	// Method name heuristics
	method := ""
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if fun.Sel != nil {
			method = fun.Sel.Name
		}
		// Check receiver chain for resource names or namespace-arg pattern
		if hasEmptyStringNamespaceArg(fun.X) || chainHasResourceName(fun.X) {
			if isCommonKubeMethod(method) {
				return true
			}
		}
	}
	// controller-runtime style: common methods with context-like first arg
	if isCommonKubeMethod(method) && len(call.Args) > 0 {
		if looksLikeContext(call.Args[0]) || argsContainKubeOptions(call.Args) {
			return true
		}
	}
	return false
}

// chainHasResourceName looks for common typed client resource selectors in a selector/call chain.
func chainHasResourceName(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if e.Sel != nil {
			if isResourceSelector(e.Sel.Name) {
				return true
			}
		}
		return chainHasResourceName(e.X)
	case *ast.CallExpr:
		// traverse into function for chained calls like Pods("")
		return chainHasResourceName(e.Fun)
	}
	return false
}
