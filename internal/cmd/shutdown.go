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
		_, client, err := v1beta1.NewPostgresClusterClient(config)
		if err != nil {
			return err
		}

		// Get the namespace. This will either be from the Kubernetes configuration
		// or from the --namespace (-n) flag.
		ns := ""
		if *config.ConfigFlags.Namespace != "" {
			ns = *config.ConfigFlags.Namespace
		}
		if err != nil {
			return err
		}

		pgclusters, err := client.Namespace(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("failed to list postgresclusters due to %+v\n", err)
			os.Exit(1)
		}

		maxCharCounts := 0
		for _, item := range pgclusters.Items {
			nameForPresentation := getPresentationNameForCluster(item)
			if len(nameForPresentation) > maxCharCounts {
				maxCharCounts = len(nameForPresentation)
			}
		}

		for _, c := range pgclusters.Items {
			_ = shutdownCluster(client, c)
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
	spec["shutdown"] = true
	updatedCluster.Object["spec"] = spec

	_, err := client.Update(context.TODO(), &updatedCluster, metav1.UpdateOptions{})
	if err != nil {
		reportUpdateFailed(item, err)
		return err
	}
	reportUpdateSucceeded(item)
	return nil
}

type UpdateStatus int

const (
	SUCCESS UpdateStatus = iota
	FAILED
)

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
	clName := getPresentationNameForCluster(item)
	_, _ = green.Printf("%s", clName)
	_, _ = white.Printf(" %s [", strings.Repeat(".", 60-len(clName)+20))
	switch {
	case status == SUCCESS:
		_, _ = green.Printf("OK")
	case status == FAILED:
		_, _ = red.Printf("KO")
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
