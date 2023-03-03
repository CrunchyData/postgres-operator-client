package cmd

import (
	"context"
	"fmt"
	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"os"
	"strings"
)

// newPowerCommand allows to power on/off a cluster, a list of clusters or
// all the crunchy clusters in this kubernetes cluster. This would happen in
// case of maintenance of a kubernetes cluster where the storage might be
// unavailable and could trigger some malfunctions. One would like to stop all
// the clusters and then restart them.
func newPowerCommand(config *internal.Config) *cobra.Command {

	cmdPower := &cobra.Command{
		Use:   "power",
		Short: "Power on/off one or more PostgresCluster",
		Long:  `Power on/off some postgresclusters.`,
	}

	cmdPower.AddCommand(
		newPowerOnCommand(config),
		newPowerOffCommand(config),
	)

	// No arguments for 'power', but there are arguments for the subcommands, e.g.
	// 'power on'
	cmdPower.Args = cobra.NoArgs

	return cmdPower
}

// newPowerOnCommand starts a shutdowned cluster.
func newPowerOnCommand(config *internal.Config) *cobra.Command {

	powerOnCmd := &cobra.Command{
		Use:   "on CLUSTER_NAME",
		Short: "Start up a PostgresCluster. If none specified, starts up all clusters.",
		Long: `Starts up PostgresCluster.

#### RBAC Requirements
    Resources          Verbs
    ---------          -----
    postgrescluster    [list,update]
`,
	}

	powerOnCmd.Example = internal.FormatExample(`
# Power on cluster hippo
pgo power on -n namespace hippo

# Power on cluster hippo and flamingo in a given namespace
pgo power on -n namespace hippo,flamingo

# Power on all clusters in namespace foo
pgo power on --all -n foo

# Power on all clusters in the K8s cluster
pgo power on --all
	`)

	// Define the 'power on' command
	powerOnCmd.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		_, clientCrunchy, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		configFlags := genericclioptions.NewConfigFlags(false)
		ns := configFlags.Namespace

		clusters, err := clientCrunchy.Namespace(*ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			os.Exit(1)
		}
		for _, cluster := range clusters.Items {
			err = powerOn(clientCrunchy, cluster)
			if err != nil {
				if _, ok := err.(AlreadyRunning); ok {
					reportAlreadyRunning(cluster)
					continue
				}
				reportUpdateFailed(cluster, err)
				continue
			}
			reportUpdateSucceeded(cluster)
		}

		return nil
	}

	return powerOnCmd
}

// newPowerOffCommand shutdown a running cluster.
func newPowerOffCommand(config *internal.Config) *cobra.Command {

	powerOffCmd := &cobra.Command{
		Use:   "off CLUSTER_NAME",
		Short: "Stops a PostgresCluster. If none specified, stops all clusters.",
		Long: `Stops a PostgresCluster.

#### RBAC Requirements
    Resources          Verbs
    ---------          -----
    postgrescluster    [list,update]
`,
	}

	powerOffCmd.Example = internal.FormatExample(`
# Power off cluster hippo
pgo power off -n namespace hippo

# Power off cluster hippo and flamingo in a given namespace
pgo power off -n namespace hippo,flamingo

# Power off all clusters in namespace foo
pgo power off --all -n foo

# Power off all clusters in the K8s cluster
pgo power off --all
	`)

	powerOffCmd.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		_, clientCrunchy, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		configFlags := genericclioptions.NewConfigFlags(false)
		ns := configFlags.Namespace

		clusters, err := clientCrunchy.Namespace(*ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			os.Exit(1)
		}
		for _, cluster := range clusters.Items {
			err = powerOff(clientCrunchy, cluster)
			if err != nil {
				if _, ok := err.(AlreadyShutdown); ok {
					reportAlreadyShutdown(cluster)
					continue
				}
				reportUpdateFailed(cluster, err)
				continue
			}
			reportUpdateSucceeded(cluster)
		}

		return nil
	}

	return powerOffCmd
}

func getPresentationNameForCluster(cluster unstructured.Unstructured) string {
	clName := cluster.GetName()
	clNamespace := cluster.GetNamespace()
	return fmt.Sprintf("%s/%s", clNamespace, clName)
}

func powerOn(client dynamic.NamespaceableResourceInterface, item unstructured.Unstructured) error {
	return changePowerOnCluster(client, item, RUNNING)
}

func powerOff(client dynamic.NamespaceableResourceInterface, item unstructured.Unstructured) error {
	return changePowerOnCluster(client, item, SHUTDOWN)
}

func changePowerOnCluster(client dynamic.NamespaceableResourceInterface, item unstructured.Unstructured, status ClusterStatus) error {
	updatedCluster := item
	spec := updatedCluster.Object["spec"].(map[string]interface{})
	switch status {
	case RUNNING:
		if spec["shutdown"] == false {
			return AlreadyRunning{}
		}
		spec["shutdown"] = false
	case SHUTDOWN:
		if spec["shutdown"] == true {
			return AlreadyShutdown{}
		}
		spec["shutdown"] = true
	}
	updatedCluster.Object["spec"] = spec

	_, err := client.Namespace(updatedCluster.GetNamespace()).Update(context.TODO(), &updatedCluster, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

type AlreadyShutdown struct{}

func (e AlreadyShutdown) Error() string {
	return "cluster is already shutdown"
}

type AlreadyRunning struct{}

func (e AlreadyRunning) Error() string {
	return "cluster is already running"
}

type ClusterStatus int

const (
	RUNNING ClusterStatus = iota
	SHUTDOWN
)

type UpdateStatus int

const (
	SUCCESS UpdateStatus = iota
	FAILED
	ALREADY_SHUTDOWN
	ALREADY_RUNNING
)

func reportAlreadyShutdown(item unstructured.Unstructured) {
	reportUpdateStatus(item, ALREADY_SHUTDOWN, "")
}

func reportAlreadyRunning(item unstructured.Unstructured) {
	reportUpdateStatus(item, ALREADY_RUNNING, "")
}

func reportUpdateFailed(item unstructured.Unstructured, err error) {
	reportUpdateStatus(item, FAILED, err.Error())
}

func reportUpdateSucceeded(item unstructured.Unstructured) {
	reportUpdateStatus(item, SUCCESS, "")
}

func reportUpdateStatus(item unstructured.Unstructured, status UpdateStatus, additionalMsg string) {
	green := color.New(color.FgGreen)
	white := color.New(color.FgWhite)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgHiYellow)
	clName := getPresentationNameForCluster(item)
	_, _ = green.Printf("%s", clName)
	_, _ = white.Printf(" %s [", strings.Repeat(".", 60-len(clName)+20))
	switch {
	case status == SUCCESS:
		_, _ = green.Printf("OK")
	case status == FAILED:
		_, _ = red.Printf("KO")
	case status == ALREADY_SHUTDOWN:
		_, _ = yellow.Printf("Already Shutdown")
	case status == ALREADY_RUNNING:
		_, _ = yellow.Printf("Already Running")
	}
	_, _ = white.Printf("]")
	if additionalMsg != "" {
		switch {
		case status == SUCCESS:
			_, _ = green.Printf(" %s", additionalMsg)
		case status == FAILED:
			_, _ = red.Printf(" %s", additionalMsg)
		}
	}
	fmt.Println("")
}
