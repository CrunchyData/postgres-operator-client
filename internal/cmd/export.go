// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/crunchydata/postgres-operator-client/internal/util"
)

const (
	// define one mebibyte in float64
	// - https://mathworld.wolfram.com/Mebibyte.html
	mebibyte float64 = (1 << 20)

	// formatting for CLI log and stdout
	preBox  = "┌────────────────────────────────────────────────────────────────"
	postBox = "└────────────────────────────────────────────────────────────────"

	// Default support export message
	msg1 = "\n" + `| Archive file size: %.2f MiB
| Email the support export archive to support@crunchydata.com
| or attach as a email reply to your existing Support Ticket` + "\n"

	// Additional support export message. Shown when size is greater than 25 MiB.
	msg2 = "\n" + `| Archive file (%.2f MiB) may be too big to email.
| Please request file share link by emailing
| support@crunchydata.com` + "\n"

	// time formatting for CLI logs
	logTimeFormat = "2006-01-02 15:04:05.000 -0700 MST"
)

// namespaced resources that have a cluster Label
var clusterNamespacedResources = []schema.GroupVersionResource{{
	Group:    appsv1.SchemeGroupVersion.Group,
	Version:  appsv1.SchemeGroupVersion.Version,
	Resource: "statefulsets",
}, {
	Group:    appsv1.SchemeGroupVersion.Group,
	Version:  appsv1.SchemeGroupVersion.Version,
	Resource: "deployments",
}, {
	Group:    appsv1.SchemeGroupVersion.Group,
	Version:  appsv1.SchemeGroupVersion.Version,
	Resource: "replicasets",
}, {
	Group:    batchv1.SchemeGroupVersion.Group,
	Version:  batchv1.SchemeGroupVersion.Version,
	Resource: "jobs",
}, {
	Group:    batchv1.SchemeGroupVersion.Group,
	Version:  batchv1.SchemeGroupVersion.Version,
	Resource: "cronjobs",
}, {
	Group:    policyv1.SchemeGroupVersion.Group,
	Version:  policyv1.SchemeGroupVersion.Version,
	Resource: "poddisruptionbudgets",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "pods",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "persistentvolumeclaims",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "configmaps",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "services",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "endpoints",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "serviceaccounts",
}}

// Resources specifically for the operator;
// currently only pods, but leaving as is to allow expansion as requested.
var operatorNamespacedResources = []schema.GroupVersionResource{{
	Group:    appsv1.SchemeGroupVersion.Group,
	Version:  appsv1.SchemeGroupVersion.Version,
	Resource: "deployments",
}, {
	Group:    appsv1.SchemeGroupVersion.Group,
	Version:  appsv1.SchemeGroupVersion.Version,
	Resource: "replicasets",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "pods",
}}

// These "removed" GVRs are for making our CLI backwards compatible with older PGO versions.
var removedNamespacedResources = []schema.GroupVersionResource{{
	Group:    batchv1beta1.SchemeGroupVersion.Group,
	Version:  batchv1beta1.SchemeGroupVersion.Version,
	Resource: "cronjobs",
}, {
	Group:    policyv1beta1.SchemeGroupVersion.Group,
	Version:  policyv1beta1.SchemeGroupVersion.Version,
	Resource: "poddisruptionbudgets",
}}

// namespaced resources that impact PGO created objects in the namespace
var otherNamespacedResources = []schema.GroupVersionResource{{
	Group:    networkingv1.SchemeGroupVersion.Group,
	Version:  networkingv1.SchemeGroupVersion.Version,
	Resource: "ingresses",
}, {
	Group:    networkingv1.SchemeGroupVersion.Group,
	Version:  networkingv1.SchemeGroupVersion.Version,
	Resource: "networkpolicies",
}, {
	Group:    corev1.SchemeGroupVersion.Group,
	Version:  corev1.SchemeGroupVersion.Version,
	Resource: "limitranges",
}}

// newSupportCommand returns the support subcommand of the PGO plugin.
func newSupportExportCommand(config *internal.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export CLUSTER_NAME",
		Short: "Export a snapshot of a PostgresCluster",
		Long: `The support export tool will collect information that is commonly necessary for troubleshooting a
PostgresCluster.

### RBAC Requirements
    Resources                                           Verbs
    ---------                                           -----
    configmaps                                          [list]
    cronjobs.batch                                      [list]
    deployments.apps                                    [list]
    endpoints                                           [list]
    events                                              [get list]
    ingresses.networking.k8s.io                         [list]
    jobs.batch                                          [list]
    limitranges                                         [list]
    namespaces                                          [get]
    networkpolicies.networking.k8s.io                   [list]
    nodes                                               [list]
    persistentvolumeclaims                              [list]
    poddisruptionbudgets.policy                         [list]
    pods                                                [list]
    pods/exec                                           [create]
    pods/log                                            [get]
    postgresclusters.postgres-operator.crunchydata.com  [get]
    replicasets.apps                                    [list]
    serviceaccounts                                     [list]
    services                                            [list]
    statefulsets.apps                                   [list]

    Note: This RBAC needs to be cluster-scoped to retrieve information on nodes and postgresclusters.

### Event Capture
    Support export captures all Events in the PostgresCluster's Namespace.
    Event duration is determined by the '--event-ttl' setting of the Kubernetes
    API server. Default is 1 hour.
    - https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/

### Usage`,
	}

	// Set output to log and write to buffer for writing to file
	var cliOutput bytes.Buffer
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// error messages should go to both stderr and the CLI log file
		errMW := io.MultiWriter(os.Stderr, &cliOutput)
		cmd.SetErr(errMW)
		// Messages printed with cmd.Print (those from the 'writeDebug' function)
		// will go only to the CLI log file. To print to the CLI log file and
		// stdout, the writeInfo function should be used.
		cmd.SetOut(&cliOutput)

		return nil
	}

	var outputDir string
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Path to save export tarball")
	cobra.CheckErr(cmd.MarkFlagRequired("output"))

	var numLogs int
	cmd.Flags().IntVarP(&numLogs, "pg-logs-count", "l", 2, "Number of pg_log files to save")

	var monitoringNamespace string
	cmd.Flags().StringVarP(&monitoringNamespace, "monitoring-namespace", "", "", "Monitoring namespace override")

	var operatorNamespace string
	cmd.Flags().StringVarP(&operatorNamespace, "operator-namespace", "", "", "Operator namespace override")

	cmd.Args = cobra.ExactArgs(1)

	cmd.Example = internal.FormatExample(`# Short Flags
kubectl pgo support export daisy -o . -l 2

# Long Flags
kubectl pgo support export daisy --output . --pg-logs-count 2

# Monitoring namespace override
# This is only required when monitoring is not deployed in the PostgresCluster's namespace.
kubectl pgo support export daisy --monitoring-namespace another-namespace --output .

# Operator namespace override
# This is only required when the Operator is not deployed in the PostgresCluster's namespace.
# This is used for getting the logs and specs for the operator pod(s).
kubectl pgo support export daisy --operator-namespace another-namespace --output .

### Example output
┌────────────────────────────────────────────────────────────────
| PGO CLI Support Export Tool
| The support export tool will collect information that is
| commonly necessary for troubleshooting a PostgresCluster.
| Note: No data or k8s secrets are collected.
| However, kubectl is used to list plugins on the user's machine.
└────────────────────────────────────────────────────────────────
Collecting PGO CLI version...
Collecting names and namespaces for PostgresClusters...
Collecting current Kubernetes context...
Collecting Kubernetes version...
Collecting nodes...
Collecting namespace...
Collecting PostgresCluster...
Collecting statefulsets...
Collecting deployments...
Collecting replicasets...
Collecting jobs...
Collecting cronjobs...
Collecting poddisruptionbudgets...
Collecting pods...
Collecting persistentvolumeclaims...
Collecting configmaps...
Collecting services...
Collecting endpoints...
Collecting serviceaccounts...
Collecting ingresses...
Collecting networkpolicies...
Collecting limitranges...
Collecting events...
Collecting Postgres logs...
Collecting pgBackRest logs...
Collecting Patroni logs...
Collecting pgBackRest Repo Host logs...
Collecting PostgresCluster pod logs...
Collecting monitoring pod logs...
Collecting operator pod logs...
Collecting Patroni info...
Collecting pgBackRest info...
Collecting processes...
Collecting system times from containers...
Collecting list of kubectl plugins...
Collecting PGO CLI logs...
┌────────────────────────────────────────────────────────────────
| Archive file size: 0.02 MiB
| Email the support export archive to support@crunchydata.com
| or attach as a email reply to your existing Support Ticket
└────────────────────────────────────────────────────────────────`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		writeInfo(cmd, preBox)
		writeInfo(cmd, "| PGO CLI Support Export Tool")
		writeInfo(cmd, "| The support export tool will collect information that is")
		writeInfo(cmd, "| commonly necessary for troubleshooting a PostgresCluster.")
		writeInfo(cmd, "| Note: No data or k8s secrets are collected.")
		writeInfo(cmd, postBox)

		clusterName := args[0]
		writeDebug(cmd, fmt.Sprintf("Arg - PostgresCluster Name: %s\n", clusterName))
		writeDebug(cmd, fmt.Sprintf("Flag - Output Directory: %s\n", outputDir))
		writeDebug(cmd, fmt.Sprintf("Flag - Num Logs: %d\n", numLogs))
		writeDebug(cmd, fmt.Sprintf("Flag - Monitoring Namespace: %s\n", monitoringNamespace))
		writeDebug(cmd, fmt.Sprintf("Flag - Operator Namespace: %s\n", operatorNamespace))

		namespace, err := config.Namespace()
		if err != nil {
			return err
		}

		restConfig, err := config.ToRESTConfig()
		if err != nil {
			return err
		}

		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		clientset, err := kubernetes.NewForConfig(restConfig)
		if err != nil {
			return err
		}

		discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
		if err != nil {
			return err
		}

		// Ensure cluster exists in the namespace before we create a file or gather
		// any information.
		// Since we check for the cluster before creating the file, these logs only
		// appear in stdout/stderr
		_, postgresClient, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		getCluster, err := postgresClient.Namespace(namespace).Get(ctx,
			clusterName, metav1.GetOptions{})
		if err != nil || getCluster == nil {
			if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
				return err
			}
			return fmt.Errorf("could not find cluster %s in namespace %s: %w", clusterName, namespace, err)
		}

		// Name file with year-month-day-HrMinSecTimezone suffix
		// Example: crunchy_k8s_support_export_2022-08-08-115726-0400.tar.gz
		outputFile := "crunchy_k8s_support_export_" + time.Now().Format("2006-01-02-150405-0700") + ".tar.gz"
		// #nosec G304 -- We intentionally write to the directory supplied by the user.
		tarFile, err := os.Create(outputDir + "/" + outputFile)
		if err != nil {
			return err
		}

		gw, err := gzip.NewWriterLevel(tarFile, gzip.BestCompression)
		if err != nil {
			return err
		}
		tw := tar.NewWriter(gw)
		defer func() {
			// ignore any errors from Close functions, the writers will be
			// closed when the program exits
			if gw != nil {
				_ = gw.Close()
			}
			if tw != nil {
				_ = tw.Close()
			}
			if tarFile != nil {
				_ = tarFile.Close()
			}
		}()

		// PGO CLI version
		err = gatherPGOCLIVersion(ctx, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering PGO CLI Version: %s", err))
		}

		// Postgres Cluster Names
		err = gatherPostgresClusterNames(clusterName, ctx, cmd, tw, postgresClient)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Postgres Cluster Names: %s", err))
		}

		// Current Kubernetes context
		err = gatherKubeContext(ctx, config, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering current Kubernetes context: %s", err))
		}

		// Gather Kubernetes Server Version
		err = gatherKubeServerVersion(ctx, discoveryClient, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Kubernetes server version: %s", err))
		}

		// Gather list of Kubernetes nodes
		err = gatherNodes(ctx, clientset, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering list of Kubernetes nodes: %s", err))
		}

		// Gather namespace information
		err = gatherCurrentNamespace(ctx, clientset, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering namespace information: %s", err))
		}

		// Gather PostgresCluster manifest
		err = gatherClusterSpec(getCluster, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering PostgresCluster manifest: %s", err))
		}

		// TODO (jmckulk): pod describe output
		// Gather Namespaced API Resources
		// get Namespaced resources that have cluster label
		nsListOpts := metav1.ListOptions{
			LabelSelector: "postgres-operator.crunchydata.com/cluster=" + clusterName,
		}
		err = gatherNamespacedAPIResources(ctx, dynamicClient, namespace,
			clusterName, clusterNamespacedResources, nsListOpts, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Namespaced API Resources: %s", err))
		}

		// Gather Namespaced API Resources
		// get other Namespaced resources that do not have the cluster label
		// but may otherwise impact the PostgresCluster's operation
		otherListOpts := metav1.ListOptions{}
		err = gatherNamespacedAPIResources(ctx, dynamicClient, namespace,
			clusterName, otherNamespacedResources, otherListOpts, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Namespaced API Resources: %s", err))
		}

		// Gather Events
		err = gatherEvents(ctx, clientset, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Events: %s", err))
		}

		// Logs
		// All Postgres Logs on the Postgres Instances (primary and replicas)
		if numLogs > 0 {
			err = gatherPostgresLogsAndConfigs(ctx, clientset, restConfig,
				namespace, clusterName, outputDir, outputFile, numLogs, tw, cmd, getCluster)
			if err != nil {
				writeInfo(cmd, fmt.Sprintf("Error gathering Postgres Logs and Config: %s", err))
			}
		}

		// All pgBackRest Logs on the Postgres Instances
		err = gatherDbBackrestLogs(ctx, clientset, restConfig, namespace, clusterName, outputDir, outputFile, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering pgBackRest DB Hosts Logs: %s", err))
		}

		// Patroni Logs that are stored on the Postgres Instances
		err = gatherPatroniLogs(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Patroni Logs from Instance Pods: %s", err))
		}

		// All pgBackRest Logs on the Repo Host
		err = gatherRepoHostLogs(ctx, clientset, restConfig, namespace, clusterName, outputDir, outputFile, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering pgBackRest Repo Host Logs: %s", err))
		}

		// get PostgresCluster Pod logs
		writeInfo(cmd, "Collecting PostgresCluster pod logs...")
		err = gatherPodLogs(ctx, clientset, namespace, fmt.Sprintf("%s=%s", util.LabelCluster, clusterName), clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering PostgresCluster pod logs: %s", err))
		}

		// get monitoring Pod logs
		if monitoringNamespace == "" {
			monitoringNamespace = namespace
		}
		writeInfo(cmd, "Collecting monitoring pod logs...")
		err = gatherPodLogs(ctx, clientset, monitoringNamespace, util.LabelMonitoring, "monitoring", tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering monitoring pod logs: %s", err))
		}

		// get operator Pod logs and descriptions
		if operatorNamespace == "" {
			operatorNamespace = namespace
		}
		// Operator and Operator upgrade pods should have
		// "postgres-operator.crunchydata.com/control-plane" label
		// but with different values
		req, _ := labels.NewRequirement(util.LabelOperator,
			selection.Exists, []string{},
		)
		nsListOpts = metav1.ListOptions{
			LabelSelector: req.String(),
		}
		err = gatherNamespacedAPIResources(ctx, dynamicClient,
			operatorNamespace, "operator", operatorNamespacedResources,
			nsListOpts, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Operator Namespace API Resources: %s", err))
		}

		// Gather Operator Pod Logs
		writeInfo(cmd, "Collecting operator pod logs...")
		err = gatherPodLogs(ctx, clientset, operatorNamespace, util.LabelOperator, "operator", tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Operator Pod logs: %s", err))
		}

		// Exec to get Patroni Information
		err = gatherPatroniInfo(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering Patroni Info: %s", err))
		}

		// Exec to get pgBackRest Information
		err = gatherPgBackRestInfo(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering pgBackRest Info: %s", err))
		}

		// Exec to get Container processes
		err = gatherProcessInfo(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering container processes: %s", err))
		}

		// Exec to get Container system time
		err = gatherSystemTime(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering container system time: %s", err))
		}

		// Get kubectl plugins
		writeInfo(cmd, "Collecting list of kubectl plugins...")
		err = gatherPluginList(clusterName, tw, cmd)
		if err != nil {
			writeInfo(cmd, fmt.Sprintf("Error gathering kubectl plugins: %s", err))
		}

		// Get PGUpgrade spec (if available)
		writeInfo(cmd, "Collecting PGUpgrade spec (if available)...")

		key := util.AllowUpgradeAnnotation()
		value, exists := getCluster.GetAnnotations()[key]

		if exists {
			writeInfo(cmd, fmt.Sprintf("The PGUpgrade object is: %s", value))
			err = gatherPGUpgradeSpec(clusterName, namespace, value, tw, cmd)
			if err != nil {
				writeInfo(cmd, fmt.Sprintf("Error gathering PGUpgrade spec: %s", err))
			}
		} else {
			writeInfo(cmd, fmt.Sprintf("There is no PGUpgrade object associated with cluster '%s'", clusterName))
		}

		// Print cli output
		writeInfo(cmd, "Collecting PGO CLI logs...")
		path := clusterName + "/cli.log"
		if logErr := writeTar(tw, cliOutput.Bytes(), path, cmd); logErr != nil {
			return logErr
		}

		// Print final message
		info, err := os.Stat(outputDir + "/" + outputFile)
		fmt.Print(exportSizeReport(float64(info.Size())))

		return err
	}

	return cmd
}

func gatherPluginList(clusterName string, tw *tar.Writer, cmd *cobra.Command) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Ensure the context is canceled to avoid leaks

	ex := exec.CommandContext(ctx, "kubectl", "plugin", "list")
	msg, err := ex.Output()

	if err != nil {
		// Capture error message when kubectl is not found in $PATH.
		msg = append(msg, err.Error()...)
	}
	path := clusterName + "/plugin-list"
	if err := writeTar(tw, msg, path, cmd); err != nil {
		return err
	}

	return nil
}

func gatherPGUpgradeSpec(clusterName, namespace, pgUpgrade string, tw *tar.Writer, cmd *cobra.Command) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Ensure the context is canceled to avoid leaks

	ex := exec.CommandContext(ctx, "kubectl", "get", "pgupgrade", pgUpgrade, "-n", namespace, "-o", "yaml")
	msg, err := ex.Output()

	if err != nil {
		msg = append(msg, err.Error()...)
		msg = append(msg, []byte(`
There was an error running 'kubectl get pgupgrade'. Verify permissions and that the resource exists.`)...)

		writeInfo(cmd, fmt.Sprintf("Error: '%s'", msg))
	}

	path := clusterName + "/pgupgrade.yaml"
	if err := writeTar(tw, msg, path, cmd); err != nil {
		return err
	}

	return nil
}

// exportSizeReport defines the message displayed when a support export archive
// is created. If the size of the archive file is greater than 25MiB, an alternate
// message is displayed.
func exportSizeReport(size float64) string {

	finalMsg := preBox + fmt.Sprintf(msg1, size/mebibyte)

	// if file size is > 25 MiB, print alternate message
	if size > mebibyte*25 {
		finalMsg = preBox + fmt.Sprintf(msg2, size/mebibyte)
	}
	finalMsg = finalMsg + postBox + "\n"

	return finalMsg
}

// gatherPGOCLIVersion collects the PGO CLI version
func gatherPGOCLIVersion(_ context.Context,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting PGO CLI version...")
	writeInfo(cmd, fmt.Sprintf("PGO CLI version is %s", clientVersion))
	path := clusterName + "/pgo-cli-version"
	if err := writeTar(tw, []byte(clientVersion), path, cmd); err != nil {
		return err
	}
	return nil
}

func gatherPostgresClusterNames(clusterName string, ctx context.Context, cmd *cobra.Command, tw *tar.Writer, client dynamic.NamespaceableResourceInterface) error {
	result, err := client.List(ctx, metav1.ListOptions{})

	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	data := []byte{}
	for _, item := range result.Items {
		ns, _, _ := unstructured.NestedString(item.Object, "metadata", "namespace")
		name, _, _ := unstructured.NestedString(item.Object, "metadata", "name")
		data = append(data, []byte("Namespace: "+ns+"\t"+"Cluster: "+name+"\n")...)
	}

	path := clusterName + "/cluster-names"
	if err := writeTar(tw, data, path, cmd); err != nil {
		return err
	}

	return nil
}

// gatherKubeContext collects the current Kubernetes context
func gatherKubeContext(_ context.Context,
	config *internal.Config,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting current Kubernetes context...")
	path := clusterName + "/current-context"

	rawConfig, err := config.ConfigFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	if err := writeTar(tw, []byte(rawConfig.CurrentContext), path, cmd); err != nil {
		return err
	}
	return nil
}

// gatherKubeServerVersion collects the server version from the Kubernetes cluster
func gatherKubeServerVersion(_ context.Context,
	client *discovery.DiscoveryClient,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting Kubernetes version...")
	ver, err := client.ServerVersion()
	if err != nil {
		return err
	}

	path := clusterName + "/server-version"
	if err := writeTar(tw, []byte(ver.String()), path, cmd); err != nil {
		return err
	}
	return nil
}

// gatherNodes gets list of nodes in the Kubernetes Cluster and prints them
// to a file using the `-o wide` output
func gatherNodes(ctx context.Context,
	clientset *kubernetes.Clientset,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting nodes...")
	list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 10, 1, 1, ' ', tabwriter.Debug)
	if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		"NAME", "STATUS", "ROLES", "AGE", "VERSION", "INTERNAL-IP", "EXTERNAL-IP",
		"OS-IMAGE", "KERNEL-VERSION", "CONTAINER-RUNTIME",
	); err != nil {
		return err
	}

	for _, item := range list.Items {

		b, err := yaml.Marshal(item)
		if err != nil {
			return err
		}

		path := clusterName + "/nodes/" + item.GetName() + ".yaml"
		if err := writeTar(tw, b, path, cmd); err != nil {
			return err
		}

		var status string
		for _, c := range item.Status.Conditions {
			if c.Type == "Ready" {
				if c.Status == "True" {
					status = "Ready"
				} else {
					status = "Not Ready"
				}
			}
		}

		rolePrefix := "node-role.kubernetes.io/"
		var roles string
		for k := range item.Labels {
			if strings.Contains(k, rolePrefix) {
				sa := strings.Split(k, rolePrefix)
				if len(sa) > 1 {
					roles = sa[1]
				}
			}
		}

		var internalIP = "<none>"
		var externalIP = "<none>"
		for _, a := range item.Status.Addresses {
			if a.Type == corev1.NodeInternalIP {
				internalIP = a.Address
			}
			if a.Type == corev1.NodeExternalIP {
				externalIP = a.Address
			}
		}

		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			item.GetName(), status, roles,
			translateTimestampSince(item.CreationTimestamp),
			item.Status.NodeInfo.KubeletVersion,
			internalIP, externalIP,
			item.Status.NodeInfo.OSImage,
			item.Status.NodeInfo.KernelVersion,
			item.Status.NodeInfo.ContainerRuntimeVersion,
		); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	path := clusterName + "/nodes/list"
	if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
		return err
	}

	return nil
}

// gatherCurrentNamespace collects the yaml output of the current namespace
func gatherCurrentNamespace(ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting namespace...")
	get, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	b, err := yaml.Marshal(get)
	if err != nil {
		return err
	}

	path := clusterName + "/current-namespace.yaml"
	if err = writeTar(tw, b, path, cmd); err != nil {
		return err
	}
	return nil
}

func gatherClusterSpec(postgresCluster *unstructured.Unstructured,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting PostgresCluster...")
	b, err := yaml.Marshal(postgresCluster)
	if err != nil {
		return err
	}

	path := clusterName + "/postgrescluster.yaml"
	if err := writeTar(tw, b, path, cmd); err != nil {
		return err
	}
	return nil
}

// gatherNamespacedAPIResources writes yaml and list output for each api-resource
// defined to an file. Using statefulsets as an example, two (or more) files will be created
// one with a list of statefulsets that were found and one yaml file for each
// statefulset
func gatherNamespacedAPIResources(ctx context.Context,
	client dynamic.Interface,
	namespace string,
	clusterName string,
	namespacedResources []schema.GroupVersionResource,
	listOpts metav1.ListOptions,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	for _, gvr := range namespacedResources {
		writeInfo(cmd, "Collecting "+gvr.Resource+"...")
		list, err := client.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
		// If the API returns an IsNotFound error, it is likely because the kube version in use
		// doesn't support the version of the resource we are attempting to use and there is an
		// earlier version we can use. This block will check the "removed" resources for a match
		// and use it if it exists.
		if apierrors.IsNotFound(err) {
			for _, bgvr := range removedNamespacedResources {
				if bgvr.Resource == gvr.Resource {
					gvr = bgvr
					list, err = client.Resource(gvr).Namespace(namespace).
						List(ctx, listOpts)
					break
				}
			}
		}
		if err != nil {
			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				// Continue and output errors for each resource type
				// Allow the user to see and address all issues at once
				continue
			}
			return err
		}
		if len(list.Items) == 0 {
			// If we didn't find any resources, skip
			writeInfo(cmd, fmt.Sprintf("Resource %s not found, skipping", gvr.Resource))
			continue
		}

		// Create a buffer to generate string with the table formatted list
		var buf bytes.Buffer
		if err := printers.NewTablePrinter(printers.PrintOptions{}).
			PrintObj(list, &buf); err != nil {
			return err
		}

		// Define the file name/path where the list file will be created and
		// write to the tar
		path := clusterName + "/" + gvr.Resource + "/list"
		if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
			return err
		}

		for _, obj := range list.Items {
			b, err := yaml.Marshal(obj)
			if err != nil {
				return err
			}

			path := clusterName + "/" + gvr.Resource + "/" + obj.GetName() + ".yaml"
			if err := writeTar(tw, b, path, cmd); err != nil {
				return err
			}
		}
	}
	return nil
}

// gatherEvents gathers all events from a namespace, selects information (based on
// what kubectl outputs), formats the data then prints to the tar file
func gatherEvents(ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting events...")
	list, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	// translateMicroTimestampSince returns the elapsed time since timestamp in
	// human-readable approximation.
	translateMicroTimestampSince := func(timestamp metav1.MicroTime) string {
		if timestamp.IsZero() {
			return "<unknown>"
		}

		return duration.HumanDuration(time.Since(timestamp.Time))
	}

	// Most of this printing code is pulled from kubectl's get events command
	// https://github.com/kubernetes/kubectl/blob/release-1.24/pkg/cmd/events/events.go#L262-L292
	var buf bytes.Buffer
	p := printers.GetNewTabWriter(&buf)
	if _, err := fmt.Fprintf(p, "Last Seen\tTYPE\tREASON\tOBJECT\tMESSAGE\n"); err != nil {
		return err
	}
	for _, event := range list.Items {
		var interval string
		firstTimestampSince := translateMicroTimestampSince(event.EventTime)
		if event.EventTime.IsZero() {
			firstTimestampSince = translateTimestampSince(event.FirstTimestamp)
		}
		if event.Series != nil {
			interval = fmt.Sprintf("%s (x%d over %s)", translateMicroTimestampSince(event.Series.LastObservedTime), event.Series.Count, firstTimestampSince)
		} else {
			interval = firstTimestampSince
		}

		if _, err := fmt.Fprintf(p, "%s\t%s\t%s\t%s/%s\t%v\n",
			interval,
			event.Type,
			event.Reason,
			event.InvolvedObject.Kind, event.InvolvedObject.Name,
			strings.TrimSpace(event.Message),
		); err != nil {
			return err
		}
	}
	if err := p.Flush(); err != nil {
		return err
	}

	path := clusterName + "/events"
	if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
		return err
	}

	return nil
}

// gatherPostgresLogsAndConfigs take a client and writes logs and configs
// from primary and replicas to a buffer
func gatherPostgresLogsAndConfigs(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	outputDir string,
	outputFile string,
	numLogs int,
	tw *tar.Writer,
	cmd *cobra.Command,
	cluster *unstructured.Unstructured,
) error {
	writeInfo(cmd, "Collecting Postgres logs...")

	dbPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.DBInstanceLabels(clusterName),
	})

	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(dbPods.Items) == 0 {
		writeInfo(cmd, "No database instance pod found for gathering logs and config")
		return nil
	}

	writeDebug(cmd, fmt.Sprintf("Found %d Pods\n", len(dbPods.Items)))

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	for _, pod := range dbPods.Items {
		writeDebug(cmd, fmt.Sprintf("Pod Name is %s\n", pod.Name))

		exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return podExec(namespace, pod.Name, util.ContainerDatabase,
				stdin, stdout, stderr, command...)
		}

		// Get Postgres Log Files
		// Pass a boolean based on whether the cluster has instrumentation
		// since that will determine where the log files are stored
		_, hasInstrumentation, _ := unstructured.NestedMap(cluster.Object,
			"spec", "instrumentation")
		stdout, stderr, err := Executor(exec).listPGLogFiles(numLogs, hasInstrumentation)

		// Depending upon the list* function above:
		// An error may happen when err is non-nil or stderr is non-empty.
		// In both cases, we want to print helpful information and continue to the
		// next iteration.
		if err != nil || stderr != "" {

			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				return nil
			}

			writeDebug(cmd, "Error getting PG logs\n")

			if err != nil {
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
			}
			if stderr != "" {
				writeDebug(cmd, stderr)
			}

			if strings.Contains(stderr, "No such file or directory") {
				writeDebug(cmd, "Cannot find any Postgres log files. This is acceptable in some configurations.\n")
			}
			continue
		}

		logFiles := strings.Split(strings.TrimSpace(stdout), "\n")

		// localDirectory is created to save data on disk
		// e.g. outputDir/crunchy_k8s_support_export_2022-08-08-115726-0400/remotePath
		localDirectory := filepath.Join(outputDir, strings.ReplaceAll(outputFile, ".tar.gz", ""))

		// flag to determine whether or not to remove localDirectory after the loop
		// When an error happens, this flag will switch to false
		// It's nice to have the extra data around when errors have happened
		doCleanup := true

		for _, logFile := range logFiles {
			// get the file size to stream
			fileSize, err := getRemoteFileSize(config, namespace, pod.Name, util.ContainerDatabase, logFile)
			if err != nil {
				writeDebug(cmd, fmt.Sprintf("could not get file size for %s: %v\n", logFile, err))
				continue
			}

			// fileSpecSrc is namespace/podname:path/to/file
			// fileSpecDest is the local destination of the file
			// These are used to help the user grab the file manually when necessary
			// e.g. postgres-operator/hippo-instance1-vp9k-0:pgdata/pg16/log/postgresql-Tue.log
			fileSpecSrc := fmt.Sprintf("%s/%s:%s", namespace, pod.Name, logFile)
			fileSpecDest := filepath.Join(localDirectory, logFile)
			writeInfo(cmd, fmt.Sprintf("\tSize of %-85s %v", fileSpecSrc, convertBytes(fileSize)))

			// Stream the file to disk and write the local file to the tar
			err = streamFileFromPod(config, tw,
				localDirectory, clusterName, namespace, pod.Name, util.ContainerDatabase, logFile, fileSize)

			if err != nil {
				doCleanup = false // prevent the deletion of localDirectory so a user can examine contents
				writeInfo(cmd, fmt.Sprintf("\tError streaming file %s: %v", logFile, err))
				writeInfo(cmd, fmt.Sprintf("\tCollect manually with kubectl cp -c %s %s %s",
					util.ContainerDatabase, fileSpecSrc, fileSpecDest))
				writeInfo(cmd, fmt.Sprintf("\tRemove %s manually after gathering necessary information", localDirectory))
				continue
			}

		}

		// doCleanup is true when there are no errors above.
		if doCleanup {
			// Remove the local directory created to hold the data
			// Errors in removing localDirectory should instruct the user to remove manually.
			// This happens often on Windows
			err = os.RemoveAll(localDirectory)
			if err != nil {
				writeInfo(cmd, fmt.Sprintf("\tError removing %s: %v", localDirectory, err))
				writeInfo(cmd, fmt.Sprintf("\tYou may need to remove %s manually", localDirectory))
				continue
			}
		}

		// Get Postgres Conf Files
		stdout, stderr, err = Executor(exec).listPGConfFiles()

		// Depending upon the list* function above:
		// An error may happen when err is non-nil or stderr is non-empty.
		// In both cases, we want to print helpful information and continue to the
		// next iteration.
		if err != nil || stderr != "" {

			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				return nil
			}

			writeDebug(cmd, "Error getting PG Conf files\n")

			if err != nil {
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
			}
			if stderr != "" {
				writeDebug(cmd, stderr)
			}

			if strings.Contains(stderr, "No such file or directory") {
				writeDebug(cmd, "Cannot find any PG Conf files. This is acceptable in some configurations.\n")
			}
			continue
		}

		logFiles = strings.Split(strings.TrimSpace(stdout), "\n")
		for _, logFile := range logFiles {
			var buf bytes.Buffer

			stdout, stderr, err := Executor(exec).catFile(logFile)
			if err != nil {
				if apierrors.IsForbidden(err) {
					writeInfo(cmd, err.Error())
					// Continue and output errors for each log file
					// Allow the user to see and address all issues at once
					continue
				}
				return err
			}

			buf.Write([]byte(stdout))
			if stderr != "" {
				str := fmt.Sprintf("\nError returned: %s\n", stderr)
				buf.Write([]byte(str))
			}

			path := clusterName + fmt.Sprintf("/pods/%s/", pod.Name) + logFile
			if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
				return err
			}
		}

		// We will execute several bash commands in the DB container
		// text is command to execute and desc is a short description
		type Command struct {
			path        string
			description string
		}

		commands := []Command{
			{path: "pg_controldata", description: "pg_controldata"},
		}

		var buf bytes.Buffer

		for _, command := range commands {
			stdout, stderr, err := Executor(exec).bashCommand(command.path)
			if err != nil {
				if apierrors.IsForbidden(err) {
					writeInfo(cmd, err.Error())
					return nil
				}
				writeDebug(cmd, fmt.Sprintf("Error executing %s\n", command.path))
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
				writeDebug(cmd, "This is acceptable in some configurations.\n")
				continue
			}
			buf.Write([]byte(fmt.Sprintf("%s\n", command.description)))
			buf.Write([]byte(stdout))
			if stderr != "" {
				buf.Write([]byte(stderr))
			}
			buf.Write([]byte("\n\n"))
		}

		// Write the buffer to a file
		path := clusterName + fmt.Sprintf("/pods/%s/%s", pod.Name, "postgres-info")
		if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
			return err
		}

	}
	return nil
}

// gatherDbBackrestLogs gathers all the file-based pgBackRest logs on the DB instance.
// There may not be any logs depending upon pgBackRest's log-level-file.
func gatherDbBackrestLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	outputDir string,
	outputFile string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting pgBackRest logs...")

	dbPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.DBInstanceLabels(clusterName),
	})

	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(dbPods.Items) == 0 {
		writeInfo(cmd, "No database instance pod found for gathering logs")
		return nil
	}

	writeDebug(cmd, fmt.Sprintf("Found %d Pods\n", len(dbPods.Items)))

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	for _, pod := range dbPods.Items {
		writeDebug(cmd, fmt.Sprintf("Pod Name is %s\n", pod.Name))

		exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return podExec(namespace, pod.Name, util.ContainerDatabase,
				stdin, stdout, stderr, command...)
		}

		// Get pgBackRest Log Files
		stdout, stderr, err := Executor(exec).listBackrestLogFiles()

		// Depending upon the list* function above:
		// An error may happen when err is non-nil or stderr is non-empty.
		// In both cases, we want to print helpful information and continue to the
		// next iteration.
		if err != nil || stderr != "" {

			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				return nil
			}

			writeDebug(cmd, "Error getting pgBackRest logs\n")

			if err != nil {
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
			}
			if stderr != "" {
				writeDebug(cmd, stderr)
			}

			if strings.Contains(stderr, "No such file or directory") {
				writeDebug(cmd, "Cannot find any pgBackRest log files. This is acceptable in some configurations.\n")
			}
			continue
		}

		logFiles := strings.Split(strings.TrimSpace(stdout), "\n")

		// localDirectory is created to save data on disk
		// e.g. outputDir/crunchy_k8s_support_export_2022-08-08-115726-0400/remotePath
		localDirectory := filepath.Join(outputDir, strings.ReplaceAll(outputFile, ".tar.gz", ""))

		// flag to determine whether or not to remove localDirectory after the loop
		// When an error happens, this flag will switch to false
		// It's nice to have the extra data around when errors have happened
		doCleanup := true

		for _, logFile := range logFiles {
			writeDebug(cmd, fmt.Sprintf("LOG FILE: %s\n", logFile))
			// get the file size to stream
			fileSize, err := getRemoteFileSize(config, namespace, pod.Name, util.ContainerDatabase, logFile)
			if err != nil {
				writeDebug(cmd, fmt.Sprintf("could not get file size for %s: %v\n", logFile, err))
				continue
			}

			// fileSpecSrc is namespace/podname:path/to/file
			// fileSpecDest is the local destination of the file
			// These are used to help the user grab the file manually when necessary
			// e.g. postgres-operator/hippo-instance1-vp9k-0:pgdata/pgbackrest/log/db-stanza-create.log
			fileSpecSrc := fmt.Sprintf("%s/%s:%s", namespace, pod.Name, logFile)
			fileSpecDest := filepath.Join(localDirectory, logFile)
			writeInfo(cmd, fmt.Sprintf("\tSize of %-85s %v", fileSpecSrc, convertBytes(fileSize)))

			// Stream the file to disk and write the local file to the tar
			err = streamFileFromPod(config, tw,
				localDirectory, clusterName, namespace, pod.Name, util.ContainerDatabase, logFile, fileSize)

			if err != nil {
				doCleanup = false // prevent the deletion of localDirectory so a user can examine contents
				writeInfo(cmd, fmt.Sprintf("\tError streaming file %s: %v", logFile, err))
				writeInfo(cmd, fmt.Sprintf("\tCollect manually with kubectl cp -c %s %s %s",
					util.ContainerDatabase, fileSpecSrc, fileSpecDest))
				writeInfo(cmd, fmt.Sprintf("\tRemove %s manually after gathering necessary information", localDirectory))
				continue
			}

		}

		// doCleanup is true when there are no errors above.
		if doCleanup {
			// Remove the local directory created to hold the data
			// Errors in removing localDirectory should instruct the user to remove manually.
			// This happens often on Windows
			err = os.RemoveAll(localDirectory)
			if err != nil {
				writeInfo(cmd, fmt.Sprintf("\tError removing %s: %v", localDirectory, err))
				writeInfo(cmd, fmt.Sprintf("\tYou may need to remove %s manually", localDirectory))
				continue
			}
		}

	}
	return nil
}

// gatherPatroniLogs gathers all the file-based Patroni logs on the DB instance,
// if configured. By default, these logs will be sent to stdout and captured as
// Pod logs instead.
func gatherPatroniLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting Patroni logs...")

	dbPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.DBInstanceLabels(clusterName),
	})

	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(dbPods.Items) == 0 {
		writeInfo(cmd, "No database instance pod found for gathering logs")
		return nil
	}

	writeDebug(cmd, fmt.Sprintf("Found %d Pods\n", len(dbPods.Items)))

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	for _, pod := range dbPods.Items {
		writeDebug(cmd, fmt.Sprintf("Pod Name is %s\n", pod.Name))

		exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return podExec(namespace, pod.Name, util.ContainerDatabase,
				stdin, stdout, stderr, command...)
		}

		// Get Patroni Log Files
		stdout, stderr, err := Executor(exec).listPatroniLogFiles()

		// Depending upon the list* function above:
		// An error may happen when err is non-nil or stderr is non-empty.
		// In both cases, we want to print helpful information and continue to the
		// next iteration.
		if err != nil || stderr != "" {

			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				return nil
			}

			writeDebug(cmd, "Error getting Patroni logs\n")

			if err != nil {
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
			}
			if stderr != "" {
				writeDebug(cmd, stderr)
			}

			if strings.Contains(stderr, "No such file or directory") {
				writeDebug(cmd, "Cannot find any Patroni log files. This is acceptable in some configurations.\n")
			}
			continue
		}

		logFiles := strings.Split(strings.TrimSpace(stdout), "\n")
		for _, logFile := range logFiles {
			writeDebug(cmd, fmt.Sprintf("LOG FILE: %s\n", logFile))
			var buf bytes.Buffer

			stdout, stderr, err := Executor(exec).catFile(logFile)
			if err != nil {
				if apierrors.IsForbidden(err) {
					writeInfo(cmd, err.Error())
					// Continue and output errors for each log file
					// Allow the user to see and address all issues at once
					continue
				}
				return err
			}

			buf.Write([]byte(stdout))
			if stderr != "" {
				str := fmt.Sprintf("\nError returned: %s\n", stderr)
				buf.Write([]byte(str))
			}

			path := clusterName + fmt.Sprintf("/pods/%s/", pod.Name) + logFile
			if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
				return err
			}
		}

	}
	return nil
}

// gatherRepoHostLogs gathers all the file-based pgBackRest logs on the repo host.
// There may not be any logs depending upon pgBackRest's log-level-file.
func gatherRepoHostLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	outputDir string,
	outputFile string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting pgBackRest Repo Host logs...")

	repoHostPods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.RepoHostInstanceLabels(clusterName),
	})

	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(repoHostPods.Items) == 0 {
		writeInfo(cmd, "No Repo Host pod found for gathering logs")
	}

	writeDebug(cmd, fmt.Sprintf("Found %d Repo Host Pod\n", len(repoHostPods.Items)))

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	for _, pod := range repoHostPods.Items {
		writeDebug(cmd, fmt.Sprintf("Pod Name is %s\n", pod.Name))

		exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			return podExec(namespace, pod.Name, util.ContainerPGBackrest,
				stdin, stdout, stderr, command...)
		}

		// Get BackRest Repo Host Log Files
		stdout, stderr, err := Executor(exec).listBackrestRepoHostLogFiles()

		// Depending upon the list* function above:
		// An error may happen when err is non-nil or stderr is non-empty.
		// In both cases, we want to print helpful information and continue to the
		// next iteration.
		if err != nil || stderr != "" {

			if apierrors.IsForbidden(err) {
				writeInfo(cmd, err.Error())
				return nil
			}

			writeDebug(cmd, "Error getting pgBackRest logs\n")

			if err != nil {
				writeDebug(cmd, fmt.Sprintf("%s\n", err.Error()))
			}
			if stderr != "" {
				writeDebug(cmd, stderr)
			}

			if strings.Contains(stderr, "No such file or directory") {
				writeDebug(cmd, "Cannot find any pgBackRest log files. This is acceptable in some configurations.\n")
			}
			continue
		}

		logFiles := strings.Split(strings.TrimSpace(stdout), "\n")

		// localDirectory is created to save data on disk
		// e.g. outputDir/crunchy_k8s_support_export_2022-08-08-115726-0400/remotePath
		localDirectory := filepath.Join(outputDir, strings.ReplaceAll(outputFile, ".tar.gz", ""))

		// flag to determine whether or not to remove localDirectory after the loop
		// When an error happens, this flag will switch to false
		// It's nice to have the extra data around when errors have happened
		doCleanup := true

		for _, logFile := range logFiles {
			writeDebug(cmd, fmt.Sprintf("LOG FILE: %s\n", logFile))
			// get the file size to stream
			fileSize, err := getRemoteFileSize(config, namespace, pod.Name, util.ContainerPGBackrest, logFile)
			if err != nil {
				writeDebug(cmd, fmt.Sprintf("could not get file size for %s: %v\n", logFile, err))
				continue
			}

			// fileSpecSrc is namespace/podname:path/to/file
			// fileSpecDest is the local destination of the file
			// These are used to help the user grab the file manually when necessary
			// e.g. postgres-operator/hippo-repo-host-0:pgbackrest/repo1/log/db-backup.log
			fileSpecSrc := fmt.Sprintf("%s/%s:%s", namespace, pod.Name, logFile)
			fileSpecDest := filepath.Join(localDirectory, logFile)
			writeInfo(cmd, fmt.Sprintf("\tSize of %-85s %v", fileSpecSrc, convertBytes(fileSize)))

			// Stream the file to disk and write the local file to the tar
			err = streamFileFromPod(config, tw,
				localDirectory, clusterName, namespace, pod.Name, util.ContainerPGBackrest, logFile, fileSize)

			if err != nil {
				doCleanup = false // prevent the deletion of localDirectory so a user can examine contents
				writeInfo(cmd, fmt.Sprintf("\tError streaming file %s: %v", logFile, err))
				writeInfo(cmd, fmt.Sprintf("\tCollect manually with kubectl cp -c %s %s %s",
					util.ContainerPGBackrest, fileSpecSrc, fileSpecDest))
				writeInfo(cmd, fmt.Sprintf("\tRemove %s manually after gathering necessary information", localDirectory))
				continue
			}
		}

		// doCleanup is true when there are no errors above.
		if doCleanup {
			// Remove the local directory created to hold the data
			// Errors in removing localDirectory should instruct the user to remove manually.
			// This happens often on Windows
			err = os.RemoveAll(localDirectory)
			if err != nil {
				writeInfo(cmd, fmt.Sprintf("\tError removing %s: %v", localDirectory, err))
				writeInfo(cmd, fmt.Sprintf("\tYou may need to remove %s manually", localDirectory))
				continue
			}
		}

	}
	return nil
}

// gatherPodLogs uses the clientset to gather logs from each container in every
// pod
func gatherPodLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
	labelSelector string,
	rootDir string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	// Get all Pods that match the given Label
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(pods.Items) == 0 {
		// If we didn't find any Pods, skip
		writeInfo(cmd, fmt.Sprintf("%s Pods not found, skipping", rootDir))
	}

	for _, pod := range pods.Items {
		containers := pod.Spec.Containers
		containers = append(containers, pod.Spec.InitContainers...)
		for _, container := range containers {
			result := clientset.CoreV1().Pods(namespace).
				GetLogs(pod.GetName(), &corev1.PodLogOptions{
					// TODO (jmckulk): we have the option to grab previous logs
					Container: container.Name,
				}).Do(ctx)

			err = result.Error()
			if err != nil {
				if apierrors.IsForbidden(result.Error()) {
					writeInfo(cmd, result.Error().Error())
					// Continue and output errors for each pod log
					// Allow the user to see and address all issues at once
					continue
				}
				return err
			}

			b, err := result.Raw()
			if err != nil {
				return err
			}

			path := rootDir + "/pods/" +
				pod.GetName() + "/containers/" + container.Name + ".log"
			if err := writeTar(tw, b, path, cmd); err != nil {
				return err
			}
		}
	}

	return nil
}

// gatherPatroniInfo takes a client and buffer
// execs into relevant pods to grab information
func gatherPatroniInfo(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting Patroni info...")
	// Get the primary instance Pod by its labels
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.PrimaryInstanceLabels(clusterName),
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}
	if len(pods.Items) < 1 {
		writeInfo(cmd, "No pod found for patroni info")
		return nil
	}

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return podExec(namespace, pods.Items[0].GetName(), util.ContainerDatabase,
			stdin, stdout, stderr, command...)
	}

	var buf bytes.Buffer

	buf.Write([]byte("patronictl list\n"))
	stdout, stderr, err := Executor(exec).patronictl("list", "")
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		writeInfo(cmd, fmt.Sprintf("Error with patronictl list: %s: %s", err, strings.TrimSpace(stderr)))
	}

	buf.Write([]byte(stdout))
	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	buf.Write([]byte("patronictl history\n"))
	stdout, stderr, err = Executor(exec).patronictl("history", "")
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		writeInfo(cmd, fmt.Sprintf("Error with patronictl history: %s: %s", err, strings.TrimSpace(stderr)))
	}

	buf.Write([]byte(stdout))
	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	path := clusterName + "/patroni-info"
	if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
		return err
	}

	return nil
}

// gatherPgBackRestInfo takes a client and buffer
// execs into relevant pods to grab information
func gatherPgBackRestInfo(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting pgBackRest info...")
	// Get the primary instance Pod by its labels
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.PrimaryInstanceLabels(clusterName),
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}
	if len(pods.Items) < 1 {
		writeInfo(cmd, "No pod found for pgBackRest info")
		return nil
	}

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return podExec(namespace, pods.Items[0].GetName(), util.ContainerDatabase,
			stdin, stdout, stderr, command...)
	}

	var buf bytes.Buffer

	buf.Write([]byte("pgbackrest info\n"))
	stdout, stderr, err := Executor(exec).pgBackRestInfo("text", "")
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		writeInfo(cmd, fmt.Sprintf("Error with pgbackrest info: %s: %s", err, strings.TrimSpace(stderr)))
	}

	buf.Write([]byte(stdout))
	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	buf.Write([]byte("pgbackrest check\n"))
	stdout, stderr, err = Executor(exec).pgBackRestCheck()
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		writeInfo(cmd, fmt.Sprintf("Error with pgbackrest check: %s: %s", err, strings.TrimSpace(stderr)))
	}

	buf.Write([]byte(stdout))
	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	path := clusterName + "/pgbackrest-info"
	return writeTar(tw, buf.Bytes(), path, cmd)
}

// gatherSystemTime takes a client and buffer and collects system time
// in each Pod and calculates the delta against client system time.
func gatherSystemTime(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting system times from containers...")
	// Get the cluster Pods by label
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "postgres-operator.crunchydata.com/cluster=" + clusterName,
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(pods.Items) == 0 {
		// If we didn't find any resources, skip
		writeInfo(cmd, "PostgresCluster Pods not found when gathering system time information, skipping")
		return nil
	}

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			// Attempt to exec in and run 'date' command in the first available container.
			exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
			) error {
				return podExec(namespace, pod.GetName(), container.Name,
					stdin, stdout, stderr, command...)
			}

			stdout, stderr, err := Executor(exec).systemTime()
			if err == nil {
				buf = writeSystemTime(buf, pod, stdout, stderr)
				break
			} else if err != nil {
				// If we get an RBAC error, let the user know and try the next pod.
				// Otherwise, try the next container.
				if apierrors.IsForbidden(err) {
					writeInfo(cmd, fmt.Sprintf(
						"Failed to get system time for Pod \"%s\". Error: \"%s\"",
						pod.GetName(), err.Error()))
					break
				}
				continue
			}
		}
	}

	path := clusterName + "/" + "system-time"
	if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
		return err
	}

	return nil
}

func writeSystemTime(buf bytes.Buffer, pod corev1.Pod, stdout, stderr string) bytes.Buffer {
	// Get client datetime.
	clientTime := time.Now().UTC()
	clientDateTimeStr := clientTime.Format(time.UnixDate)

	var deltaStr string
	var containerDateTimeStr string
	if containerDateTime, err := time.Parse(time.UnixDate, strings.TrimSpace(stdout)); err == nil {
		// Calculate difference between client and container datetime.
		containerDateTimeStr = containerDateTime.Format(time.UnixDate)
		deltaStr = fmt.Sprint(clientTime.Sub(containerDateTime).Truncate(time.Second))
	} else {
		// Parse failed, use stdout instead.
		containerDateTimeStr = strings.TrimSpace(stdout)
		deltaStr = "No result"
	}

	// Build report.
	fmt.Fprintln(&buf, "Delta: "+deltaStr+"\tPod time: "+containerDateTimeStr+
		"\tClient time: "+clientDateTimeStr+"\tPod name: "+pod.GetName())

	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	return buf
}

// gatherProcessInfo takes a client and buffer execs into relevant pods to grab
// running process information for each Pod.
func gatherProcessInfo(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	writeInfo(cmd, "Collecting processes...")
	// Get the cluster Pods by label
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "postgres-operator.crunchydata.com/cluster=" + clusterName,
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			writeInfo(cmd, err.Error())
			return nil
		}
		return err
	}

	if len(pods.Items) == 0 {
		// If we didn't find any resources, skip
		writeInfo(cmd, "PostgresCluster Pods not found when gathering process information, skipping")
		return nil
	}

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			// Attempt to exec in and run 'ps' command in all available containers,
			// regardless of state, etc. Many of the resulting process lists will
			// be nearly identical because certain Pods use a shared process
			// namespace, but this function aims to gather as much detail as possible.
			// - https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace/
			exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
			) error {
				return podExec(namespace, pod.GetName(), container.Name,
					stdin, stdout, stderr, command...)
			}

			stdout, stderr, err := Executor(exec).processes()
			if err != nil {
				// If we get an RBAC error, let the user know and try the next pod.
				// Otherwise, try the next container.
				if apierrors.IsForbidden(err) {
					writeInfo(cmd, fmt.Sprintf(
						"Failed to get processes for Pod \"%s\". Error: \"%s\"",
						pod.GetName(), err.Error()))
					break
				}
				continue
			}

			var buf bytes.Buffer
			buf.Write([]byte(stdout))
			if stderr != "" {
				buf.Write([]byte(stderr))
			}

			path := clusterName + "/" + "processes" + "/" + pod.GetName() + "/" + container.Name
			if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
				return err
			}
		}
	}

	return nil
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

// writeTar takes content as a byte slice and writes the content to a tar writer
func writeTar(tw *tar.Writer, content []byte, name string, cmd *cobra.Command) error {
	hdr := &tar.Header{
		Name:    name,
		Mode:    0600,
		ModTime: time.Now(),
		Size:    int64(len(content)),
	}

	writeDebug(cmd, fmt.Sprintf("File: %s Size: %d\n", name, hdr.Size))
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	// After we write content to the file, call Flush to ensure the files block is fully padded.
	// This shouldn't be necessary based on the tar docs: https://pkg.go.dev/archive/tar#Writer.Flush
	if err := tw.Flush(); err != nil {
		return err
	}
	return nil
}

// writeInfo logs to both the PGO CLI log file and stdout
// TODO(tjmoore4): In the future, should we implement a logger instead?
func writeInfo(cmd *cobra.Command, s string) {
	t := time.Now()
	// write to CLI log buffer
	cmd.Printf("%s - INFO - %s\n", t.Format(logTimeFormat), s)
	// write to stdout
	fmt.Println(s)
}

// writeDebug logs to only the PGO CLI log file
func writeDebug(cmd *cobra.Command, s string) {
	t := time.Now()
	cmd.Printf("%s - DEBUG - %s", t.Format(logTimeFormat), s)
}

// streamFileFromPod streams the file from the Kubernetes pod to a local file.
func streamFileFromPod(config *rest.Config, tw *tar.Writer,
	localDirectory, clusterName, namespace, podName, containerName, remotePath string,
	remoteFileSize int64) error {

	// create localPath to write the streamed data from remotePath
	// use the uniqueness of outputFile to avoid overwriting other files
	localPath := filepath.Join(localDirectory, remotePath)
	if err := os.MkdirAll(filepath.Dir(localPath), 0750); err != nil {
		return fmt.Errorf("failed to create path for file: %w", err)
	}
	outFile, err := os.Create(filepath.Clean(localPath))
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}

	defer func() {
		// ignore any errors from Close functions, the writers will be
		// closed when the program exits
		if outFile != nil {
			_ = outFile.Close()
		}
	}()

	// Get Postgres Log Files
	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return err
	}
	exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return podExec(namespace, podName, containerName,
			stdin, stdout, stderr, command...)
	}

	_, err = Executor(exec).copyFile(remotePath, outFile)
	if err != nil {
		return fmt.Errorf("error during file streaming: %w", err)
	}

	// compare file sizes
	localFileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	if remoteFileSize != localFileInfo.Size() {
		return fmt.Errorf("filesize mismatch: remote size is %v and local size is %v",
			remoteFileSize, localFileInfo.Size())
	}

	// add localPath to the support export tar
	tarPath := fmt.Sprintf("%s/pods/%s/%s", clusterName, podName, remotePath)
	err = addFileToTar(tw, localPath, tarPath)
	if err != nil {
		return fmt.Errorf("error writing to tar: %w", err)
	}

	return nil
}

// addFileToTar copies a local file into a tar archive
func addFileToTar(tw *tar.Writer, localPath, tarPath string) error {
	// Open the file to be added to the tar
	file, err := os.Open(filepath.Clean(localPath))
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		// ignore any errors from Close functions, the writers will be
		// closed when the program exits
		if file != nil {
			_ = file.Close()
		}
	}()

	// Get file info to create tar header
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create tar header
	header := &tar.Header{
		Name:    tarPath,                // Name in the tar archive
		Size:    fileInfo.Size(),        // File size
		Mode:    int64(fileInfo.Mode()), // File mode
		ModTime: fileInfo.ModTime(),     // Modification time
	}

	// Write header to the tar
	err = tw.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	// Stream the file content to the tar
	_, err = io.Copy(tw, file)
	if err != nil {
		return fmt.Errorf("failed to copy file data to tar: %w", err)
	}

	return nil
}

// getRemoteFileSize returns the size of a file within a container so that we can stream its contents
func getRemoteFileSize(config *rest.Config,
	namespace string, podName string, containerName string, filePath string) (int64, error) {

	podExec, err := util.NewPodExecutor(config)
	if err != nil {
		return 0, fmt.Errorf("could not create executor: %w", err)
	}
	exec := func(stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		return podExec(namespace, podName, containerName,
			stdin, stdout, stderr, command...)
	}

	// Prepare the command to get the file size using "stat -c %s <file>"
	command := fmt.Sprintf("stat -c %s %s", "%s", filePath)
	stdout, stderr, err := Executor(exec).bashCommand(command)
	if err != nil {
		return 0, fmt.Errorf("could not get file size: %w, stderr: %s", err, stderr)
	}

	// Parse the file size from stdout
	size, err := strconv.ParseInt(strings.TrimSpace(stdout), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse file size: %w", err)
	}

	return size, nil
}

// convertBytes converts a byte size (int64) into a human-readable format.
func convertBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
