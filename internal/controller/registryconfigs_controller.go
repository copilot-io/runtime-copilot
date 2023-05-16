/*
Copyright 2023.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	configregistryv1alpha1 "github.com/copilot-io/runtime-copilot/api/v1alpha1"
	"github.com/copilot-io/runtime-copilot/internal/utils"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// RegistryConfigsReconciler reconciles a RegistryConfigs object.
type RegistryConfigsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=registryconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=registryconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=registryconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=noderegistryconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups=,resources=nodes,verbs=get;list;watch
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.namme`
// +kubebuilder:printcolumn:name="STATE",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RegistryConfigs object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *RegistryConfigsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling RegistryConfigs ", "NamespacedName", req.NamespacedName)
	// step1. get the RegistryConfigs object
	// TODO(lrf), current object may be is NodereRistryConfigs, current we don't to handle it
	var registryConfigs configregistryv1alpha1.RegistryConfigs
	if err := r.Get(ctx, req.NamespacedName, &registryConfigs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// step2. get the node list
	var nodeList v1.NodeList
	if err := r.List(ctx, &nodeList); err != nil {
		return ctrl.Result{}, err
	}
	// step3. get the nodeRegistryConfigs list
	var existNodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList
	listOption := client.MatchingLabels{
		"config.registry.runtime.copilot.io/registryconfigs": registryConfigs.Name,
	}
	for k, v := range registryConfigs.Spec.Selector.MatchLabels {
		listOption[k] = v
	}
	if err := r.List(ctx, &existNodeRegistryConfigsList, listOption); err != nil {
		return ctrl.Result{}, err
	}
	nodeExistCr := make(map[string]configregistryv1alpha1.NodeRegistryConfigs)
	for index, item := range existNodeRegistryConfigsList.Items {
		nodeExistCr[item.Spec.NodeName] = existNodeRegistryConfigsList.Items[index]
	}
	// step4. generate the NodeRegistryConfigs
	nodeRegistryConfigsList := r.generateNodeRegistryConfigs(ctx, registryConfigs, nodeList)
	// step5. create or update the NodeRegistryConfigs
	for _, nodeRegistryConfigs := range nodeRegistryConfigsList.Items {
		if err := r.applyNodeRegistryConfigs(ctx, nodeRegistryConfigs, nodeExistCr); err != nil {
			return ctrl.Result{}, err
		}
	}
	// step6. update the RegistryConfigs
	if err := r.syncRegistryConfigsStatus(ctx, registryConfigs, nodeList); err != nil {
		return ctrl.Result{}, err
	}
	if registryConfigs.Status.State == configregistryv1alpha1.StatusStateSuccess {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *RegistryConfigsReconciler) generateNodeRegistryConfigs(ctx context.Context, registryConfigs configregistryv1alpha1.RegistryConfigs, nodeList v1.NodeList) configregistryv1alpha1.NodeRegistryConfigsList {
	var nodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList
	for _, node := range nodeList.Items {
		if registryConfigs.Spec.Template.Spec.RetryNum == 0 {
			registryConfigs.Spec.Template.Spec.RetryNum = configregistryv1alpha1.MaxRetryNum
		}
		nodeRegistryConfigs := configregistryv1alpha1.NodeRegistryConfigs{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", registryConfigs.Name, utils.RandUUID()),
				Namespace: registryConfigs.Namespace,
				Labels: map[string]string{
					"config.registry.runtime.copilot.io/registryconfigs": registryConfigs.Name,
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: registryConfigs.APIVersion,
						Kind:       registryConfigs.Kind,
						Name:       registryConfigs.Name,
						UID:        registryConfigs.UID,
					},
				},
			},
			Spec: configregistryv1alpha1.NodeRegistryConfigsSpec{
				NodeName:    node.Name,
				Type:        r.convertRuntimeType(node.Status.NodeInfo.ContainerRuntimeVersion),
				HostConfigs: registryConfigs.Spec.Template.Spec.HostConfigs,
				RetryNum:    registryConfigs.Spec.Template.Spec.RetryNum,
			},
		}

		for k, v := range registryConfigs.Spec.Selector.MatchLabels {
			nodeRegistryConfigs.ObjectMeta.Labels[k] = v
		}
		nodeRegistryConfigsList.Items = append(nodeRegistryConfigsList.Items, nodeRegistryConfigs)
	}
	return nodeRegistryConfigsList
}

func (r *RegistryConfigsReconciler) applyNodeRegistryConfigs(ctx context.Context, nodeRegistryConfigs configregistryv1alpha1.NodeRegistryConfigs, nodeExistCr map[string]configregistryv1alpha1.NodeRegistryConfigs) error {
	var nodeRegistryConfigsOld configregistryv1alpha1.NodeRegistryConfigs
	var ok bool
	if nodeRegistryConfigsOld, ok = nodeExistCr[nodeRegistryConfigs.Spec.NodeName]; !ok {
		return r.Create(ctx, &nodeRegistryConfigs)
	}
	lastAppliedConfiguration := configregistryv1alpha1.NodeRegistryConfigs{
		TypeMeta: nodeRegistryConfigsOld.TypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:            nodeRegistryConfigsOld.Name,
			Namespace:       nodeRegistryConfigsOld.Namespace,
			Labels:          nodeRegistryConfigsOld.Labels,
			OwnerReferences: nodeRegistryConfigsOld.OwnerReferences,
		},
		Spec: nodeRegistryConfigsOld.Spec,
	}
	bytes, _ := json.Marshal(lastAppliedConfiguration)
	if nodeRegistryConfigsOld.ObjectMeta.Annotations == nil {
		nodeRegistryConfigsOld.ObjectMeta.Annotations = make(map[string]string)
	}
	nodeRegistryConfigsOld.ObjectMeta.Annotations["config.registry.runtime.copilot.io/last-applied-configuration"] = string(bytes)
	nodeRegistryConfigsOld.Spec = nodeRegistryConfigs.Spec
	return r.Patch(ctx, &nodeRegistryConfigsOld, client.Merge, &client.PatchOptions{FieldManager: string(types.JSONPatchType)})
}

func (r *RegistryConfigsReconciler) syncRegistryConfigsStatus(ctx context.Context, registryConfigs configregistryv1alpha1.RegistryConfigs, nodeList v1.NodeList) error {
	var nodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList
	listOption := client.MatchingLabels{
		"config.registry.runtime.copilot.io/registryconfigs": registryConfigs.Name,
	}
	for k, v := range registryConfigs.Spec.Selector.MatchLabels {
		listOption[k] = v
	}
	if err := r.List(ctx, &nodeRegistryConfigsList, listOption); err != nil {
		return err
	}

	registryConfigs.Status.State = configregistryv1alpha1.StatusStateRunning
	registryConfigs.Status.TotalNodes = r.countTotalNode(ctx, nodeList)
	registryConfigs.Status.SuccessNodes = r.countSuccessNode(ctx, nodeList, nodeRegistryConfigsList)
	registryConfigs.Status.FailedNodes = r.countFailedNode(ctx, nodeList, nodeRegistryConfigsList)
	registryConfigs.Status.RunningNodes = r.countRunningNode(ctx, nodeList, nodeRegistryConfigsList)

	if len(registryConfigs.Status.FailedNodes) > 0 {
		registryConfigs.Status.State = configregistryv1alpha1.StatusStateFailed
	}
	if r.equalSuccessAndTotal(registryConfigs.Status.TotalNodes, registryConfigs.Status.SuccessNodes) {
		registryConfigs.Status.State = configregistryv1alpha1.StatusStateSuccess
	}
	return r.Status().Update(ctx, &registryConfigs)
}

func (r *RegistryConfigsReconciler) convertRuntimeType(containerRuntimeVersion string) configregistryv1alpha1.RuntimeType {
	if strings.Contains(containerRuntimeVersion, "docker") {
		return configregistryv1alpha1.RuntimeTypeDocker
	}
	if strings.Contains(containerRuntimeVersion, "containerd") {
		return configregistryv1alpha1.RuntimeTypeContainerd
	}
	if strings.Contains(containerRuntimeVersion, "crio") {
		return configregistryv1alpha1.RuntimeTypeCrio
	}
	return configregistryv1alpha1.RuntimeTypeUnknown
}

func (r *RegistryConfigsReconciler) countTotalNode(ctx context.Context, nodeList v1.NodeList) []configregistryv1alpha1.RuntimeNum {
	runtimeNumsMap := make(map[configregistryv1alpha1.RuntimeType]int)
	for _, node := range nodeList.Items {
		runtimeType := r.convertRuntimeType(node.Status.NodeInfo.ContainerRuntimeVersion)
		if _, ok := runtimeNumsMap[runtimeType]; !ok {
			runtimeNumsMap[runtimeType] = 1
		} else {
			runtimeNumsMap[runtimeType]++
		}
	}
	runtimeNums := make([]configregistryv1alpha1.RuntimeNum, 0)
	for k, v := range runtimeNumsMap {
		runtimeNums = append(runtimeNums, configregistryv1alpha1.RuntimeNum{
			RuntimeType: k,
			Num:         v,
		})
	}
	return runtimeNums
}

func (r *RegistryConfigsReconciler) countSuccessNode(ctx context.Context, nodeList v1.NodeList, nodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList) []configregistryv1alpha1.RuntimeNum {
	runtimeTypeMap := make(map[string]configregistryv1alpha1.RuntimeType)
	for _, node := range nodeList.Items {
		runtimeType := r.convertRuntimeType(node.Status.NodeInfo.ContainerRuntimeVersion)
		if _, ok := runtimeTypeMap[node.Name]; !ok {
			runtimeTypeMap[node.Name] = runtimeType
		}
	}
	runtimeNumsMap := make(map[configregistryv1alpha1.RuntimeType]int)
	for _, item := range nodeRegistryConfigsList.Items {
		if item.Status.State != configregistryv1alpha1.StatusStateSuccess {
			continue
		}
		runtimeType := runtimeTypeMap[item.Spec.NodeName]
		if _, ok := runtimeNumsMap[runtimeType]; !ok {
			runtimeNumsMap[runtimeType] = 1
		} else {
			runtimeNumsMap[runtimeType]++
		}
	}
	runtimeNums := make([]configregistryv1alpha1.RuntimeNum, 0)
	for k, v := range runtimeNumsMap {
		runtimeNums = append(runtimeNums, configregistryv1alpha1.RuntimeNum{
			RuntimeType: k,
			Num:         v,
		})
	}
	return runtimeNums
}

func (r *RegistryConfigsReconciler) countFailedNode(ctx context.Context, nodeList v1.NodeList, nodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList) []configregistryv1alpha1.RuntimeNum {
	runtimeTypeMap := make(map[string]configregistryv1alpha1.RuntimeType)
	for _, node := range nodeList.Items {
		runtimeType := r.convertRuntimeType(node.Status.NodeInfo.ContainerRuntimeVersion)
		if _, ok := runtimeTypeMap[node.Name]; !ok {
			runtimeTypeMap[node.Name] = runtimeType
		}
	}
	runtimeNumsMap := make(map[configregistryv1alpha1.RuntimeType]int)
	for _, item := range nodeRegistryConfigsList.Items {
		if item.Status.State != configregistryv1alpha1.StatusStateFailed {
			continue
		}
		runtimeType := runtimeTypeMap[item.Spec.NodeName]
		if _, ok := runtimeNumsMap[runtimeType]; !ok {
			runtimeNumsMap[runtimeType] = 1
		} else {
			runtimeNumsMap[runtimeType]++
		}
	}
	runtimeNums := make([]configregistryv1alpha1.RuntimeNum, 0)
	for k, v := range runtimeNumsMap {
		runtimeNums = append(runtimeNums, configregistryv1alpha1.RuntimeNum{
			RuntimeType: k,
			Num:         v,
		})
	}
	return runtimeNums
}

func (r *RegistryConfigsReconciler) countRunningNode(ctx context.Context, nodeList v1.NodeList, nodeRegistryConfigsList configregistryv1alpha1.NodeRegistryConfigsList) []configregistryv1alpha1.RuntimeNum {
	runtimeTypeMap := make(map[string]configregistryv1alpha1.RuntimeType)
	for _, node := range nodeList.Items {
		runtimeType := r.convertRuntimeType(node.Status.NodeInfo.ContainerRuntimeVersion)
		if _, ok := runtimeTypeMap[node.Name]; !ok {
			runtimeTypeMap[node.Name] = runtimeType
		}
	}
	runtimeNumsMap := make(map[configregistryv1alpha1.RuntimeType]int)
	for _, item := range nodeRegistryConfigsList.Items {
		if item.Status.State != configregistryv1alpha1.StatusStateRunning {
			continue
		}
		runtimeType := runtimeTypeMap[item.Spec.NodeName]
		if _, ok := runtimeNumsMap[runtimeType]; !ok {
			runtimeNumsMap[runtimeType] = 1
		} else {
			runtimeNumsMap[runtimeType]++
		}
	}
	runtimeNums := make([]configregistryv1alpha1.RuntimeNum, 0)
	for k, v := range runtimeNumsMap {
		runtimeNums = append(runtimeNums, configregistryv1alpha1.RuntimeNum{
			RuntimeType: k,
			Num:         v,
		})
	}
	return runtimeNums
}

func (r *RegistryConfigsReconciler) equalSuccessAndTotal(total, success []configregistryv1alpha1.RuntimeNum) bool {
	if len(total) != len(success) {
		return false
	}
	totalMap, successMap := make(map[configregistryv1alpha1.RuntimeType]int), make(map[configregistryv1alpha1.RuntimeType]int)
	for _, item := range total {
		totalMap[item.RuntimeType] = item.Num
	}
	for _, item := range success {
		successMap[item.RuntimeType] = item.Num
	}
	for k, v := range totalMap {
		if successMap[k] != v {
			return false
		}
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryConfigsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configregistryv1alpha1.RegistryConfigs{}).
		Complete(r)
}
