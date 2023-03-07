package cmd

import (
	"bufio"
	"context"
	"fmt"
	"github.com/crunchydata/postgres-operator-client/internal"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"os"
	"strings"
)

var flagAllCluster bool

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

	powerOnCmd.Flags().BoolVarP(&flagAllCluster, "all", "", false, "Apply to all clusters in the namespace if supplied or in the clusters if none")

	// Define the 'power on' command
	powerOnCmd.RunE = func(cmd *cobra.Command, args []string) error {

		_, clientCrunchy, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		var clusterNames []string
		if len(args) == 1 {
			clusterNames = strings.Split(args[0], ",")
		}
		ns := ""
		if *config.ConfigFlags.Namespace != "" {
			ns = *config.ConfigFlags.Namespace
		}

		clusters, err := collectClusters(clientCrunchy,
			ClustersToCollect{Namespace: ns, Clusters: clusterNames, All: flagAllCluster})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			os.Exit(1)
		}
		powerOnClusters(clientCrunchy, clusters)

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

	powerOffCmd.Flags().BoolVarP(&flagAllCluster, "all", "", false, "Apply to all clusters in the namespace if supplied or in the clusters if none")

	powerOffCmd.RunE = func(cmd *cobra.Command, args []string) error {

		// configure client
		_, clientCrunchy, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}
		var clusterNames []string
		if len(args) == 1 {
			clusterNames = strings.Split(args[0], ",")
		}

		ns := ""
		if *config.ConfigFlags.Namespace != "" {
			ns = *config.ConfigFlags.Namespace
		}

		if ns == "" && flagAllCluster {
			yellow := color.New(color.FgHiYellow)
			_, _ = yellow.Printf("[WARNING] You request to stop all PG clusters of this k8s cluster. Do you want to proceed ?(Y/N)\n")
			userInput := bufio.NewScanner(os.Stdin)
			for userInput.Scan() {
				answer := userInput.Text()
				switch answer {
				case "N":
					os.Exit(1)
				case "Y":
					goto answerYes
				default:
					fmt.Println("Please answer Y or N")
				}
			}
		answerYes:
			fmt.Println("Confirmation to shutdown all PG Clusters in the cluster.")
		}

		clusters, err := collectClusters(clientCrunchy,
			ClustersToCollect{Namespace: ns, Clusters: clusterNames, All: flagAllCluster})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			os.Exit(1)
		}
		powerOffClusters(clientCrunchy, clusters)

		return nil
	}

	return powerOffCmd
}

func collectClusters(clientCrunchy dynamic.NamespaceableResourceInterface, toCollect ClustersToCollect) ([]unstructured.Unstructured, error) {
	if toCollect.All {
		allClusters, err := clientCrunchy.Namespace(toCollect.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Failed to list postgresql clusters: %+v\n", err)
			return []unstructured.Unstructured{}, err
		}
		return allClusters.Items, nil
	}
	var clusters []unstructured.Unstructured
	for _, cluster := range toCollect.Clusters {
		retrieved, err := clientCrunchy.Namespace(toCollect.Namespace).Get(context.TODO(), cluster, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Failed to get postgresql cluster %q in namespace %q : %+v\n", cluster, toCollect.Namespace, err)
			return []unstructured.Unstructured{}, err
		}
		clusters = append(clusters, *retrieved)
	}
	return clusters, nil
}

func powerOffClusters(clientCrunchy dynamic.NamespaceableResourceInterface,
	clusters []unstructured.Unstructured) {
	for _, cluster := range clusters {
		err := powerOff(clientCrunchy, cluster)
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
}

func powerOnClusters(clientCrunchy dynamic.NamespaceableResourceInterface,
	clusters []unstructured.Unstructured) {
	for _, cluster := range clusters {
		err := powerOn(clientCrunchy, cluster)
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

func changePowerOnUnstructured(item unstructured.Unstructured, status ClusterStatus) (unstructured.Unstructured, error) {
	updatedCluster := item
	spec := updatedCluster.Object["spec"].(map[string]interface{})
	switch status {
	case RUNNING:
		if spec["shutdown"] == false {
			return unstructured.Unstructured{}, AlreadyRunning{}
		}
		spec["shutdown"] = false
	case SHUTDOWN:
		if spec["shutdown"] == true {
			return unstructured.Unstructured{}, AlreadyShutdown{}
		}
		spec["shutdown"] = true
	}
	updatedCluster.Object["spec"] = spec
	return updatedCluster, nil
}

func changePowerOnCluster(client dynamic.NamespaceableResourceInterface, item unstructured.Unstructured, status ClusterStatus) error {
	updatedCluster, err := changePowerOnUnstructured(item, status)
	if err != nil {
		return err
	}
	_, err = client.Namespace(updatedCluster.GetNamespace()).Update(context.TODO(), &updatedCluster, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

type ClustersToCollect struct {
	All       bool
	Namespace string
	Clusters  []string
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
