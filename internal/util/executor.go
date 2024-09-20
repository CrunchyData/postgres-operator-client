// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// podExecutor runs command on container in pod in namespace. Non-nil streams
// (stdin, stdout, and stderr) are attached to the remote process.
type podExecutor func(
	namespace, pod, container string,
	stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// NewPodExecutor returns an executor function. It is used when commands are run
// from a Container shell using an 'exec' command.
// The RBAC settings required for this are "resources=pods/exec,verbs=create"
func NewPodExecutor(config *rest.Config) (podExecutor, error) {

	client, err := clientv1.NewForConfig(config)

	return func(
		namespace, pod, container string,
		stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		request := client.RESTClient().Post().
			Resource("pods").SubResource("exec").
			Namespace(namespace).Name(pod).
			VersionedParams(&corev1.PodExecOptions{
				Container: container,
				Command:   command,
				Stdin:     stdin != nil,
				Stdout:    stdout != nil,
				Stderr:    stderr != nil,
			}, scheme.ParameterCodec)

		exec, err := remotecommand.NewSPDYExecutor(config, "POST", request.URL())

		if err == nil {
			err = exec.Stream(remotecommand.StreamOptions{
				Stdin:  stdin,
				Stdout: stdout,
				Stderr: stderr,
			})
		}

		return err
	}, err
}
