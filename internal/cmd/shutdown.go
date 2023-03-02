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

// newShutdownCommand allows to shutdown one cluster, a list of clusters or
// all the crunchy clusters in this kubernetes cluster. This would happen in
// case of maintenance of a kubernetes cluster where the storage might be
// unavailable and could trigger some malfunctions.
func newShutdownCommand(config *internal.Config) *cobra.Command {

	cmdShutdown := &cobra.Command{
		Use:   "shutdown",
		Short: "Shutdown one or more PostgresCluster",
		Long: `Shuts down some postgresclusters.

#### RBAC Requirements
    Resources         Verbs
    ---------         -----
    postgresclusters  [list,update]
`,
	}

	cmdShutdown.Example = internal.FormatExample(`
# Shutdown cluster hippo
pgo shutdown -n namespace hippo

# Shutdown cluster hippo and flamingo in a given namespace
pgo shutdown -n namespace hippo,flamingo

# Shutdown all clusters in namespace foo
pgo shutdown --all -n foo

# Shutdown all clusters in the K8s cluster
pgo shutdown --all
`)

	var all bool
	cmdShutdown.Flags().BoolVar(&all, "all", false, "requires to shutdown all clusters in the namespace. If no namespace is specified, shutdowns all clusters of all namespaces")

	cmdShutdown.Args = cobra.MinimumNArgs(0)

	cmdShutdown.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !all {
			cmdShutdown.Printf("If you define no cluster, you should set the --all flag. Nothing to shutdown.\n")
			os.Exit(1)
		}

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
		maxCharCounts := 0
		for _, item := range clusters.Items {
			nameForPresentation := getPresentationNameForCluster(item)
			if len(nameForPresentation) > maxCharCounts {
				maxCharCounts = len(nameForPresentation)
			}
		}
		for _, unstructuredPgCluster := range clusters.Items {
			err = shutdownCluster(clientCrunchy, unstructuredPgCluster)
			if err != nil {
				if _, ok := err.(AlreadyShutdown); ok {
					reportAlreadyShutdown(unstructuredPgCluster)
					continue
				}
				reportUpdateFailed(unstructuredPgCluster, err)
				continue
			}
			reportUpdateSucceeded(unstructuredPgCluster)
		}

		return nil
	}

	return cmdShutdown
}

func getPresentationNameForCluster(cluster unstructured.Unstructured) string {
	clName := cluster.GetName()
	clNamespace := cluster.GetNamespace()
	return fmt.Sprintf("%s/%s", clNamespace, clName)
}

func shutdownCluster(client dynamic.NamespaceableResourceInterface, item unstructured.Unstructured) error {
	updatedCluster := item
	spec := updatedCluster.Object["spec"].(map[string]interface{})
	if spec["shutdown"] == true {
		return AlreadyShutdown{}
	}
	spec["shutdown"] = true
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

type UpdateStatus int

const (
	SUCCESS UpdateStatus = iota
	FAILED
	ALREADY_SHUTDOWN
)

func reportUpdateFailed(item unstructured.Unstructured, err error) {
	reportUpdateStatus(item, FAILED, err.Error())
}

func reportAlreadyShutdown(item unstructured.Unstructured) {
	reportUpdateStatus(item, ALREADY_SHUTDOWN, "")
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
