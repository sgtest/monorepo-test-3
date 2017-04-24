/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-7/pkg/api"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-7/pkg/kubectl/cmd/templates"
	cmdutil "github.com/sourcegraph/monorepo-test-1/kubernetes-7/pkg/kubectl/cmd/util"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-7/pkg/kubectl/resource"
	"github.com/sourcegraph/monorepo-test-1/kubernetes-7/pkg/util/i18n"

	"github.com/daviddengcn/go-colortext"
	"github.com/spf13/cobra"
)

var (
	longDescr = templates.LongDesc(i18n.T(`
  Display addresses of the master and services with label kubernetes.io/cluster-service=true
  To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.`))

	clusterinfo_example = templates.Examples(i18n.T(`
		# Print the address of the master and cluster services
		kubectl cluster-info`))
)

func NewCmdClusterInfo(f cmdutil.Factory, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use: "cluster-info",
		// clusterinfo is deprecated.
		Aliases: []string{"clusterinfo"},
		Short:   i18n.T("Display cluster info"),
		Long:    longDescr,
		Example: clusterinfo_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := RunClusterInfo(f, out, cmd)
			cmdutil.CheckErr(err)
		},
	}
	cmdutil.AddInclude3rdPartyFlags(cmd)
	cmd.AddCommand(NewCmdClusterInfoDump(f, out))
	return cmd
}

func RunClusterInfo(f cmdutil.Factory, out io.Writer, cmd *cobra.Command) error {
	if len(os.Args) > 1 && os.Args[1] == "clusterinfo" {
		printDeprecationWarning("cluster-info", "clusterinfo")
	}

	client, err := f.ClientConfig()
	if err != nil {
		return err
	}
	printService(out, "Kubernetes master", client.Host)

	mapper, typer := f.Object()
	cmdNamespace := cmdutil.GetFlagString(cmd, "namespace")
	if cmdNamespace == "" {
		cmdNamespace = metav1.NamespaceSystem
	}

	// TODO use generalized labels once they are implemented (#341)
	b := resource.NewBuilder(mapper, f.CategoryExpander(), typer, resource.ClientMapperFunc(f.ClientForMapping), f.Decoder(true)).
		NamespaceParam(cmdNamespace).DefaultNamespace().
		SelectorParam("kubernetes.io/cluster-service=true").
		ResourceTypeOrNameArgs(false, []string{"services"}...).
		Latest()
	b.Do().Visit(func(r *resource.Info, err error) error {
		if err != nil {
			return err
		}
		services := r.Object.(*api.ServiceList).Items
		for _, service := range services {
			var link string
			if len(service.Status.LoadBalancer.Ingress) > 0 {
				ingress := service.Status.LoadBalancer.Ingress[0]
				ip := ingress.IP
				if ip == "" {
					ip = ingress.Hostname
				}
				for _, port := range service.Spec.Ports {
					link += "http://" + ip + ":" + strconv.Itoa(int(port.Port)) + " "
				}
			} else {
				if len(client.GroupVersion.Group) == 0 {
					link = client.Host + "/api/" + client.GroupVersion.Version + "/namespaces/" + service.ObjectMeta.Namespace + "/services/" + service.ObjectMeta.Name + "/proxy"
				} else {
					link = client.Host + "/api/" + client.GroupVersion.Group + "/" + client.GroupVersion.Version + "/namespaces/" + service.ObjectMeta.Namespace + "/services/" + service.ObjectMeta.Name + "/proxy"

				}
			}
			name := service.ObjectMeta.Labels["kubernetes.io/name"]
			if len(name) == 0 {
				name = service.ObjectMeta.Name
			}
			printService(out, name, link)
		}
		return nil
	})
	out.Write([]byte("\nTo further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.\n"))
	return nil

	// TODO consider printing more information about cluster
}

func printService(out io.Writer, name, link string) {
	ct.ChangeColor(ct.Green, false, ct.None, false)
	fmt.Fprint(out, name)
	ct.ResetColor()
	fmt.Fprintf(out, " is running at ")
	ct.ChangeColor(ct.Yellow, false, ct.None, false)
	fmt.Fprint(out, link)
	ct.ResetColor()
	fmt.Fprintln(out, "")
}