// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/dynamic"
)

func NewPgadminClient(rcg resource.RESTClientGetter) (
	*meta.RESTMapping, dynamic.NamespaceableResourceInterface, error,
) {
	gvk := GroupVersion.WithKind("PGAdmin")

	mapper, err := rcg.ToRESTMapper()
	if err != nil {
		return nil, nil, err
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, nil, err
	}

	config, err := rcg.ToRESTConfig()
	if err != nil {
		return nil, nil, err
	}

	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return mapping, client.Resource(mapping.Resource), nil
}
