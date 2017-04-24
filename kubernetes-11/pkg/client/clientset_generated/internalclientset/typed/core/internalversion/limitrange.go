/*
Copyright 2017 The Kubernetes Authors.

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

package internalversion

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
	api "github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/api"
	scheme "github.com/sourcegraph/monorepo-test-1/kubernetes-11/pkg/client/clientset_generated/internalclientset/scheme"
)

// LimitRangesGetter has a method to return a LimitRangeInterface.
// A group's client should implement this interface.
type LimitRangesGetter interface {
	LimitRanges(namespace string) LimitRangeInterface
}

// LimitRangeInterface has methods to work with LimitRange resources.
type LimitRangeInterface interface {
	Create(*api.LimitRange) (*api.LimitRange, error)
	Update(*api.LimitRange) (*api.LimitRange, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*api.LimitRange, error)
	List(opts v1.ListOptions) (*api.LimitRangeList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *api.LimitRange, err error)
	LimitRangeExpansion
}

// limitRanges implements LimitRangeInterface
type limitRanges struct {
	client rest.Interface
	ns     string
}

// newLimitRanges returns a LimitRanges
func newLimitRanges(c *CoreClient, namespace string) *limitRanges {
	return &limitRanges{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Create takes the representation of a limitRange and creates it.  Returns the server's representation of the limitRange, and an error, if there is any.
func (c *limitRanges) Create(limitRange *api.LimitRange) (result *api.LimitRange, err error) {
	result = &api.LimitRange{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("limitranges").
		Body(limitRange).
		Do().
		Into(result)
	return
}

// Update takes the representation of a limitRange and updates it. Returns the server's representation of the limitRange, and an error, if there is any.
func (c *limitRanges) Update(limitRange *api.LimitRange) (result *api.LimitRange, err error) {
	result = &api.LimitRange{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("limitranges").
		Name(limitRange.Name).
		Body(limitRange).
		Do().
		Into(result)
	return
}

// Delete takes name of the limitRange and deletes it. Returns an error if one occurs.
func (c *limitRanges) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("limitranges").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *limitRanges) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("limitranges").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Get takes name of the limitRange, and returns the corresponding limitRange object, and an error if there is any.
func (c *limitRanges) Get(name string, options v1.GetOptions) (result *api.LimitRange, err error) {
	result = &api.LimitRange{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("limitranges").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of LimitRanges that match those selectors.
func (c *limitRanges) List(opts v1.ListOptions) (result *api.LimitRangeList, err error) {
	result = &api.LimitRangeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("limitranges").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested limitRanges.
func (c *limitRanges) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("limitranges").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Patch applies the patch and returns the patched limitRange.
func (c *limitRanges) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *api.LimitRange, err error) {
	result = &api.LimitRange{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("limitranges").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}