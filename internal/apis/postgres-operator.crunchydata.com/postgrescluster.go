// Copyright 2021 - 2026 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

// Package postgresoperator provides a version-aware client factory for the
// PostgresCluster CRD. It supports the v1 and v1beta1 API versions and will
// auto-detect which is served by the target cluster when no explicit override
// is provided.
package postgresoperator

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	v1 "github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1"
	"github.com/crunchydata/postgres-operator-client/internal/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// APIVersionV1 is the v1 API version of the postgres-operator.crunchydata.com group.
	APIVersionV1 = "v1"

	// APIVersionV1Beta1 is the v1beta1 API version of the postgres-operator.crunchydata.com group.
	APIVersionV1Beta1 = "v1beta1"

	// EnvAPIVersion is the environment variable consulted when no explicit
	// --pgo-api-version override is set.
	EnvAPIVersion = "PGO_API_VERSION"

	// DefaultAPIVersion is used when no override is set and discovery cannot
	// determine which version is served.
	DefaultAPIVersion = APIVersionV1Beta1
)

// Group is the API group for the PostgresCluster CRD. Sourced from one of the
// per-version GroupVersion vars so the three values can never drift.
var Group = v1beta1.GroupVersion.Group

// validateAPIVersion returns nil if v is one of the supported API versions.
func validateAPIVersion(v string) error {
	switch v {
	case APIVersionV1, APIVersionV1Beta1:
		return nil
	default:
		return fmt.Errorf(
			"unsupported PostgresCluster API version %q (supported: %s, %s)",
			v, APIVersionV1, APIVersionV1Beta1,
		)
	}
}

// groupVersionFor returns the schema.GroupVersion for the given supported
// API version. It panics on unrecognized input because all call sites have
// already validated the version.
func groupVersionFor(version string) schema.GroupVersion {
	switch version {
	case APIVersionV1:
		return v1.GroupVersion
	case APIVersionV1Beta1:
		return v1beta1.GroupVersion
	default:
		panic(fmt.Sprintf("postgresoperator: unsupported API version %q", version))
	}
}

// ResolvePostgresClusterVersion returns the API version to use for the
// PostgresCluster CRD. Resolution order:
//  1. explicit override (e.g. from the --pgo-api-version flag)
//  2. PGO_API_VERSION environment variable
//  3. discovery: if the cluster serves
//     postgres-operator.crunchydata.com/v1 with a postgresclusters resource,
//     use v1
//  4. fall back to DefaultAPIVersion (v1beta1)
//
// rcg may be nil, in which case only the override / env var are considered and
// the default is returned if neither is set.
func ResolvePostgresClusterVersion(override string, rcg resource.RESTClientGetter) (string, error) {
	if override != "" {
		if err := validateAPIVersion(override); err != nil {
			return "", err
		}
		return override, nil
	}

	if env := os.Getenv(EnvAPIVersion); env != "" {
		if err := validateAPIVersion(env); err != nil {
			return "", fmt.Errorf("invalid %s: %w", EnvAPIVersion, err)
		}
		return env, nil
	}

	if rcg == nil {
		return DefaultAPIVersion, nil
	}

	// Discovery failures below are deliberately swallowed in favor of the
	// v1beta1 default. v1beta1 has been served by every published PGO
	// release, so falling through is safer than refusing to run. The
	// trade-off is that on a v1-only cluster a transient discovery failure
	// will surface later as a confusing "no matches for kind PostgresCluster"
	// from the REST mapper rather than a clear connectivity error.
	restConfig, err := rcg.ToRESTConfig()
	if err != nil {
		return DefaultAPIVersion, nil
	}
	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return DefaultAPIVersion, nil
	}
	list, err := dc.ServerResourcesForGroupVersion(v1.GroupVersion.String())
	if err != nil || list == nil {
		return DefaultAPIVersion, nil
	}
	for _, r := range list.APIResources {
		if r.Name == "postgresclusters" {
			return APIVersionV1, nil
		}
	}
	return DefaultAPIVersion, nil
}

// newPostgresClusterClient constructs a REST mapping and dynamic client for
// the PostgresCluster kind in the given GroupVersion. This is the shared
// implementation used by NewPostgresClusterClient; the per-version subpackages
// (v1, v1beta1) intentionally only expose their GroupVersion so there is one
// place where the mapping/client wiring lives.
func newPostgresClusterClient(rcg resource.RESTClientGetter, gv schema.GroupVersion) (
	*meta.RESTMapping, dynamic.NamespaceableResourceInterface, error,
) {
	gvk := gv.WithKind("PostgresCluster")

	mapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, nil, err
	}
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, nil, err
	}

	restConfig, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, err
	}
	return mapping, client.Resource(mapping.Resource), nil
}

// NewPostgresClusterClient resolves the PostgresCluster API version (see
// ResolvePostgresClusterVersion) and returns the REST mapping, a dynamic
// client scoped to the PostgresCluster resource, and the resolved API version
// string. Callers that need to stamp an apiVersion onto a manifest (such as
// create) should use the returned version.
func NewPostgresClusterClient(override string, rcg resource.RESTClientGetter) (
	*meta.RESTMapping, dynamic.NamespaceableResourceInterface, string, error,
) {
	version, err := ResolvePostgresClusterVersion(override, rcg)
	if err != nil {
		return nil, nil, "", err
	}

	mapping, client, err := newPostgresClusterClient(rcg, groupVersionFor(version))
	return mapping, client, version, err
}

// GroupVersion returns the schema.GroupVersion string for the given API
// version, e.g. "postgres-operator.crunchydata.com/v1".
func GroupVersion(version string) string {
	return Group + "/" + version
}
