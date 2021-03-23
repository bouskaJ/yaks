/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package install

import (
	"context"
	"errors"
	"fmt"
	"github.com/citrusframework/yaks/pkg/apis/yaks/v1alpha1"
	"strconv"
	"time"

	"github.com/citrusframework/yaks/deploy"
	"github.com/citrusframework/yaks/pkg/client"
	"github.com/citrusframework/yaks/pkg/util/kubernetes"
	"github.com/citrusframework/yaks/pkg/util/kubernetes/customclient"

	"k8s.io/apimachinery/pkg/util/yaml"

	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SetupClusterWideResourcesOrCollect --
func SetupClusterWideResourcesOrCollect(ctx context.Context, clientProvider client.Provider, collection *kubernetes.Collection) error {
	// Get a client to install the CRD
	c, err := clientProvider.Get()
	if err != nil {
		return err
	}

	// Install CRD for Instance (if needed)
	if err := installCRD(ctx, c, v1alpha1.InstanceKind, v1alpha1.SchemeGroupVersion.Version, "crd-instance.yaml", collection); err != nil {
		return err
	}

	// Install CRD for Test (if needed)
	if err := installCRD(ctx, c, v1alpha1.TestKind, v1alpha1.SchemeGroupVersion.Version, "crd-test.yaml", collection); err != nil {
		return err
	}

	// Wait for all CRDs to be installed before proceeding
	if err := WaitForAllCRDInstallation(ctx, clientProvider, 25*time.Second); err != nil {
		return err
	}

	// Installing ClusterRole
	clusterRoleInstalled, err := IsClusterRoleInstalled(ctx, c)
	if err != nil {
		return err
	}
	if !clusterRoleInstalled || collection != nil {
		err := installClusterRole(ctx, c, collection)
		if err != nil {
			return err
		}
	}

	// Install OpenShift Console download links if possible
	err = OpenShiftConsoleDownloadLink(ctx, c)
	if err != nil {
		return err
	}

	return nil
}

// WaitForAllCRDInstallation waits until all CRDs are installed
func WaitForAllCRDInstallation(ctx context.Context, clientProvider client.Provider, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		var c client.Client
		var err error
		if c, err = clientProvider.Get(); err != nil {
			return err
		}
		var inst bool
		if inst, err = AreAllCRDInstalled(ctx, c); err != nil {
			return err
		} else if inst {
			return nil
		}
		// Check after 2 seconds if not expired
		if time.Now().After(deadline) {
			return errors.New("cannot check CRD installation after " + strconv.FormatInt(timeout.Nanoseconds()/1000000000, 10) + " seconds")
		}
		time.Sleep(2 * time.Second)
	}
}

// AreAllCRDInstalled check if all the required CRDs are installed
func AreAllCRDInstalled(ctx context.Context, c client.Client) (bool, error) {
	return IsCRDInstalled(ctx, c, "Test", "v1alpha1")
}

// IsCRDInstalled check if the given CRD kind is installed
func IsCRDInstalled(ctx context.Context, c client.Client, kind string, version string) (bool, error) {
	lst, err := c.Discovery().ServerResourcesForGroupVersion(fmt.Sprintf("yaks.citrusframework.org/%s", version))
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	for _, res := range lst.APIResources {
		if res.Kind == kind {
			return true, nil
		}
	}
	return false, nil
}

func installCRD(ctx context.Context, c client.Client, kind string, version string, resourceName string, collection *kubernetes.Collection) error {
	crd := deploy.Resource(resourceName)
	if collection != nil {
		unstr, err := kubernetes.LoadRawResourceFromYaml(string(crd))
		if err != nil {
			return err
		}
		collection.Add(unstr)
		return nil
	}

	// Installing Integration CRD
	installed, err := IsCRDInstalled(ctx, c, kind, version)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	crdJSON, err := yaml.ToJSON(crd)
	if err != nil {
		return err
	}
	restClient, err := customclient.GetClientFor(c, "apiextensions.k8s.io", "v1beta1")
	if err != nil {
		return err
	}
	// Post using dynamic client
	result := restClient.
		Post().
		Body(crdJSON).
		Resource("customresourcedefinitions").
		Do(ctx)
	// Check result
	if result.Error() != nil && !k8serrors.IsAlreadyExists(result.Error()) {
		return result.Error()
	}

	return nil
}

// IsClusterRoleInstalled check if cluster role yaks:edit is installed
func IsClusterRoleInstalled(ctx context.Context, c client.Client) (bool, error) {
	clusterRole := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "yaks:edit",
		},
	}
	key := k8sclient.ObjectKeyFromObject(&clusterRole)
	err := c.Get(ctx, key, &clusterRole)
	if err != nil && k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func installClusterRole(ctx context.Context, c client.Client, collection *kubernetes.Collection) error {
	obj, err := kubernetes.LoadResourceFromYaml(c.GetScheme(), deploy.ResourceAsString("/user-cluster-role.yaml"))
	if err != nil {
		return err
	}

	if collection != nil {
		collection.Add(obj)
		return nil
	}
	return c.Create(ctx, obj)
}
