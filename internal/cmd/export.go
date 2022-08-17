// Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

var namespacedResources = []schema.GroupVersionResource{{
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
	Group:    batchv1beta1.SchemeGroupVersion.Group,
	Version:  batchv1beta1.SchemeGroupVersion.Version,
	Resource: "cronjobs",
}, {
	Group: policyv1.SchemeGroupVersion.Group,
	// As of PGO 5.2.x, we use `v1beta1` as the version for poddisruptionbudgets;
	// this works from k8s 1.19 to 1.24, which is our current k8s
	// supported range.
	// If/when we start supporting k8s 1.25, we may need to revisit this decision for
	// both pdb and cronjobs.
	Version:  "v1beta1",
	Resource: "poddisruptionbudgets",
}, {
	Version:  "v1",
	Resource: "pods",
}, {
	Version:  "v1",
	Resource: "persistentvolumeclaims",
}, {
	Version:  "v1",
	Resource: "configmaps",
}, {
	Version:  "v1",
	Resource: "services",
}, {
	Version:  "v1",
	Resource: "endpoints",
}, {
	Version:  "v1",
	Resource: "serviceaccounts",
}}

// newSupportCommand returns the support subcommand of the PGO plugin.
func newSupportExportCommand(config *internal.Config) *cobra.Command {

	var collectedResources []string
	for _, resource := range namespacedResources {
		collectedResources = append(collectedResources, resource.Resource)
	}

	cmd := &cobra.Command{
		Use:   "export CLUSTER_NAME",
		Short: "Export a snapshot of a PostgresCluster",
		Long: fmt.Sprintf(`
The support export tool will collect information that is commonly necessary for troubleshooting a
PostgresCluster.

Collected Resources: %v

RBAC Requirements
Resources                                           Verbs
---------                                           -----
configmaps                                          [get list]
cronjobs.batch                                      [get list]
deployments.apps                                    [get list]
endpoints                                           [get list]
events                                              [get list]
jobs.batch                                          [get list]
namespaces                                          [get]
nodes                                               [list]
persistentvolumeclaims                              [get list]
poddisruptionbudgets.policy                         [get list]
pods                                                [get list]
pods/exec                                           [create]
pods/log                                            [get]
postgresclusters.postgres-operator.crunchydata.com  [get]
replicasets.apps                                    [get list]
serviceaccounts                                     [get list]
services                                            [get list]
statefulsets.apps                                   [get list]

Note: This RBAC needs to be cluster-scoped to retrieve information on nodes.`, collectedResources),
	}

	// Set output to log and write to buffer for writing to file
	var cliOutput bytes.Buffer
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		outMW := io.MultiWriter(os.Stdout, &cliOutput)
		errMW := io.MultiWriter(os.Stderr, &cliOutput)
		cmd.SetOut(outMW)
		cmd.SetErr(errMW)

		return nil
	}

	var outputDir string
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Path to save export tarball")
	cobra.CheckErr(cmd.MarkFlagRequired("output"))

	var numLogs int
	cmd.Flags().IntVarP(&numLogs, "pg-logs-count", "l", 2, "Number of pg_log files to save")

	cmd.Args = cobra.ExactArgs(1)

	cmd.Example = internal.FormatExample(`
# Short Flags
kubectl pgo support export daisy -o . -l 2

# Long Flags
kubectl pgo support export daisy --output . --pg-logs-count 2
	`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		clusterName := args[0]

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
		get, err := postgresClient.Namespace(namespace).Get(ctx,
			clusterName, metav1.GetOptions{})
		if err != nil || get == nil {
			if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
				return fmt.Errorf(err.Error())
			}
			return fmt.Errorf("could not find cluster %s in namespace %s: %w", clusterName, namespace, err)
		}

		// Name file with year-month-day-HrMinSecTimezone suffix
		// Example: crunchy_k8s_support_export_2022-08-08-115726-0400.tar
		outputFile := "crunchy_k8s_support_export_" + time.Now().Format("2006-01-02-150405-0700") + ".tar"
		// #nosec G304 -- We intentionally write to the directory supplied by the user.
		tarFile, err := os.Create(outputDir + "/" + outputFile)
		if err != nil {
			return err
		}

		tw := tar.NewWriter(tarFile)
		defer func() {
			// ignore any errors from Close functions, the writers will be
			// closed when the program exits
			if tw != nil {
				_ = tw.Close()
			}
			if tarFile != nil {
				_ = tarFile.Close()
			}
		}()

		// TODO (jmckulk): collect context info
		// TODO (jmckulk): collect client version, after pgo version command is implemented

		// Gather cluster wide resources
		err = gatherKubeServerVersion(ctx, discoveryClient, clusterName, tw, cmd)

		if err == nil {
			err = gatherNodes(ctx, clientset, clusterName, tw, cmd)
		}

		if err == nil {
			err = gatherCurrentNamespace(ctx, clientset, namespace, clusterName, tw, cmd)
		}

		// Namespaced resources
		if err == nil {
			err = gatherClusterSpec(ctx, get, clusterName, tw, cmd)
		}

		// TODO (jmckulk): pod describe output
		if err == nil {
			err = gatherNamespacedAPIResources(ctx, dynamicClient, namespace, clusterName, tw, cmd)
		}

		if err == nil {
			err = gatherEvents(ctx, clientset, namespace, clusterName, tw, cmd)
		}

		// Logs
    if numLogs > 0 {
		  if err == nil {
			  err = gatherPostgresqlLogs(ctx, clientset, restConfig, namespace, clusterName, numLogs, tw, cmd)
      }
		}

		if err == nil {
			err = gatherPodLogs(ctx, clientset, namespace, clusterName, tw, cmd)
		}

		// Exec resources
		if err == nil {
			err = gatherPatroniInfo(ctx, clientset, restConfig, namespace, clusterName, tw, cmd)
		}

		// Print cli output
		path := clusterName + "/logs/cli"
		if logErr := writeTar(tw, cliOutput.Bytes(), path, cmd); logErr != nil {
			return logErr
		}

		return err
	}

	return cmd
}

// gatherKubeServerVersion collects the server version from the Kubernetes cluster
func gatherKubeServerVersion(_ context.Context,
	client *discovery.DiscoveryClient,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
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
	list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
	}

	var buf bytes.Buffer
	if err := printers.NewTablePrinter(printers.PrintOptions{
		Wide: true,
	}).PrintObj(list, &buf); err != nil {
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
	get, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
			cmd.Println(err.Error())
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

func gatherClusterSpec(ctx context.Context,
	postgresCluster *unstructured.Unstructured,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
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
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	for _, gvr := range namespacedResources {
		list, err := client.Resource(gvr).Namespace(namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: "postgres-operator.crunchydata.com/cluster=" + clusterName,
			})
		if err != nil {
			if apierrors.IsForbidden(err) {
				cmd.Println(err.Error())
				// Continue and output errors for each resource type
				// Allow the user to see and address all issues at once
				continue
			}
			return err
		}
		if len(list.Items) == 0 {
			// If we didn't find any resources, skip
			cmd.Printf("Resource %s not found, skipping\n", gvr.Resource)
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
			// Get each object defined in list, marshal the object and print
			// to a file
			get, err := client.Resource(gvr).Namespace(namespace).
				Get(ctx, obj.GetName(), metav1.GetOptions{})
			if err != nil {
				if apierrors.IsForbidden(err) || apierrors.IsNotFound(err) {
					cmd.Println(err.Error())
					// Continue and output errors for each resource type
					// Allow the user to see and address all issues at once
					continue
				}
				return err
			}

			b, err := yaml.Marshal(get)
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
	// TODO (jmckulk): do we need to order events?
	list, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
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

	// translateTimestampSince returns the elapsed time since timestamp in
	// human-readable approximation.
	translateTimestampSince := func(timestamp metav1.Time) string {
		if timestamp.IsZero() {
			return "<unknown>"
		}

		return duration.HumanDuration(time.Since(timestamp.Time))
	}

	// Most of this printing code is pulled from kubectl's get events command
	// https://github.com/kubernetes/kubectl/blob/release-1.24/pkg/cmd/events/events.go#L262-L292
	var buf bytes.Buffer
	p := printers.GetNewTabWriter(&buf)
	fmt.Fprintf(p, "Last Seen\tTYPE\tREASON\tOBJECT\tMESSAGE\n")
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

		fmt.Fprintf(p, "%s\t%s\t%s\t%s/%s\t%v\n",
			interval,
			event.Type,
			event.Reason,
			event.InvolvedObject.Kind, event.InvolvedObject.Name,
			strings.TrimSpace(event.Message),
		)
	}

	path := clusterName + "/events"
	if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
		return err
	}

	return nil
}

// gatherLogs takes a client and buffer to write logs to a buffer
func gatherPostgresqlLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	numLogs int,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	// Get the primary instance Pod by its labels
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		// TODO(jmckulk): should we be getting replica logs?
		LabelSelector: util.PrimaryInstanceLabels(clusterName),
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
	}
	if len(pods.Items) != 1 {
		msg := "expect one primary instance pod for gathering logs"
		cmd.PrintErr(msg + "\n")
		return fmt.Errorf(msg)
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

	stdout, stderr, err := Executor(exec).listPGLogFiles(numLogs)
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
	}
	if stderr != "" {
		cmd.Println(stderr)
	}

	logFiles := strings.Split(strings.TrimSpace(stdout), "\n")
	for _, logFile := range logFiles {
		var buf bytes.Buffer

		stdout, stderr, err := Executor(exec).catFile(logFile)
		if err != nil {
			if apierrors.IsForbidden(err) {
				cmd.Println(err.Error())
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

		path := clusterName + "/logs/postgresql/" + logFile
		if err := writeTar(tw, buf.Bytes(), path, cmd); err != nil {
			return err
		}
	}

	return nil
}

// gatherPodLogs uses the clientset to gather logs from each container in every
// pod
func gatherPodLogs(ctx context.Context,
	clientset *kubernetes.Clientset,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	// TODO: update to use specific client after SSA change
	// Get the primary instance Pod by its labels
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "postgres-operator.crunchydata.com/cluster=" + clusterName,
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
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

			if result.Error() != nil {
				if apierrors.IsForbidden(result.Error()) {
					cmd.Println(result.Error().Error())
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

			path := clusterName + "/logs/" + pod.GetName() + "/" + container.Name
			if err := writeTar(tw, b, path, cmd); err != nil {
				return err
			}
		}
	}

	return nil
}

// gatherExecInfo takes a client and buffer
// execs into relevant pods to grab information
func gatherPatroniInfo(ctx context.Context,
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace string,
	clusterName string,
	tw *tar.Writer,
	cmd *cobra.Command,
) error {
	// TODO: update to use specific client after SSA change
	// Get the primary instance Pod by its labels
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: util.PrimaryInstanceLabels(clusterName),
	})
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
	}
	if len(pods.Items) < 1 {
		msg := "expect at least one pod for patroni info"
		cmd.PrintErr(msg + "\n")
		return fmt.Errorf(msg)
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
	stdout, stderr, err := Executor(exec).patronictl("list")
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
	}

	buf.Write([]byte(stdout))
	if stderr != "" {
		buf.Write([]byte(stderr))
	}

	buf.Write([]byte("patronictl history\n"))
	stdout, stderr, err = Executor(exec).patronictl("history")
	if err != nil {
		if apierrors.IsForbidden(err) {
			cmd.Println(err.Error())
			return nil
		}
		return err
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

// writeTar takes content as a byte slice and writes the content to a tar writer
func writeTar(tw *tar.Writer, content []byte, name string, cmd *cobra.Command) error {
	hdr := &tar.Header{
		Name: name,
		Mode: 0600,
		Size: int64(len(content)),
	}

	// TODO (jmckulk): figure out what support tool output looks like and make
	// this match
	cmd.Printf("File: %s Size: %d\n", name, hdr.Size)
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
