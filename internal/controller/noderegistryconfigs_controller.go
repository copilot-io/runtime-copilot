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
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"reflect"

	configregistryv1alpha1 "github.com/copilot-io/runtime-copilot/api/v1alpha1"

	"github.com/BurntSushi/toml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NodeRegistryConfigsReconciler reconciles a NodeRegistryConfigs object.
type NodeRegistryConfigsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// HostRootDir is the runtime config registry host root dir
	HostRootDir string
}

//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=noderegistryconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=noderegistryconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=config.registry.runtime.copilot.io,resources=noderegistryconfigs/finalizers,verbs=update
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.namme`
// +kubebuilder:printcolumn:name="STATE",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeRegistryConfigs object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *NodeRegistryConfigsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return ctrl.Result{}, nil
	}
	klog.Infof("reconcile node registry config for node %s", req.String())
	// step1. get the NodeRegistryConfigs object
	var nodeRegistryConfigs configregistryv1alpha1.NodeRegistryConfigs
	if err := r.Get(ctx, req.NamespacedName, &nodeRegistryConfigs); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if nodeRegistryConfigs.Spec.NodeName != nodeName {
		return ctrl.Result{}, nil
	}
	if nodeRegistryConfigs.Status.RetryNum == nodeRegistryConfigs.Spec.RetryNum {
		klog.Warning("node registry config retry num is: %d, please check the config,", nodeRegistryConfigs.Spec.RetryNum)
		return ctrl.Result{}, nil
	}
	// step2. compare last nodeRegistryConfigs and current nodeRegistryConfigs, find the delete server
	if v, ok := nodeRegistryConfigs.ObjectMeta.Annotations["config.registry.runtime.copilot.io/last-applied-configuration"]; ok {
		var old configregistryv1alpha1.NodeRegistryConfigs
		if err := json.Unmarshal([]byte(v), &old); err != nil {
			return ctrl.Result{}, err
		}
		deleteNodeRegistryHostConfigs := r.diffDifferent(old, nodeRegistryConfigs)
		// step4. delete this node registry config for this server
		if err := r.deleteRegistryByHost(deleteNodeRegistryHostConfigs); err != nil {
			nodeRegistryConfigs.Status.State = configregistryv1alpha1.StatusStateFailed
			nodeRegistryConfigs.Status.RetryNum++
			if err := r.Status().Update(ctx, &nodeRegistryConfigs); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}
	// step3. get current node for this server registry config, compare need update server
	if err := r.createOrUpdateRegistry(&nodeRegistryConfigs); err != nil {
		nodeRegistryConfigs.Status.State = configregistryv1alpha1.StatusStateFailed
		nodeRegistryConfigs.Status.RetryNum++
		if err := r.Status().Update(ctx, &nodeRegistryConfigs); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	// step5. create or update this node registry config for this server
	nodeRegistryConfigs.Status.State = configregistryv1alpha1.StatusStateSuccess
	if err := r.Status().Update(ctx, &nodeRegistryConfigs); err != nil {
		nodeRegistryConfigs.Status.State = configregistryv1alpha1.StatusStateFailed
		nodeRegistryConfigs.Status.RetryNum++
		if err := r.Status().Update(ctx, &nodeRegistryConfigs); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *NodeRegistryConfigsReconciler) diffDifferent(oldCR, newCR configregistryv1alpha1.NodeRegistryConfigs) []configregistryv1alpha1.NodeRegistryHostConfig {
	oldHostConfigs, newHostConfigs := oldCR.Spec.HostConfigs, newCR.Spec.HostConfigs
	oldHostConfigsMap, newHostConfigsMap := make(map[string]configregistryv1alpha1.NodeRegistryHostConfig), make(map[string]configregistryv1alpha1.NodeRegistryHostConfig)
	for i, v := range oldHostConfigs {
		oldHostConfigsMap[v.Server] = oldHostConfigs[i]
	}
	for i, v := range newHostConfigs {
		newHostConfigsMap[v.Server] = newHostConfigs[i]
	}
	for k := range newHostConfigsMap {
		delete(oldHostConfigsMap, k)
	}
	if len(oldHostConfigsMap) == 0 {
		return nil
	}

	deleteNodeRegistryHostConfigs := make([]configregistryv1alpha1.NodeRegistryHostConfig, 0)
	for k, v := range oldHostConfigsMap {
		deleteNodeRegistryHostConfigs = append(deleteNodeRegistryHostConfigs, v)
		klog.Infof("delete node registry config for server %s", k)
	}
	return deleteNodeRegistryHostConfigs
}

func (r *NodeRegistryConfigsReconciler) deleteRegistryByHost(deleteHosts []configregistryv1alpha1.NodeRegistryHostConfig) error {
	if len(deleteHosts) == 0 {
		return nil
	}
	for _, host := range deleteHosts {
		parse, _ := url.Parse(host.Server)
		dir := filepath.Join(r.HostRootDir, parse.Host)
		_, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			klog.Errorf("%s stat error: %+v", dir, err)
			return err
		}
		os.RemoveAll(dir)
	}
	return nil
}

func (r *NodeRegistryConfigsReconciler) createOrUpdateRegistry(nodeRegistryConfigs *configregistryv1alpha1.NodeRegistryConfigs) error {
	if len(nodeRegistryConfigs.Spec.HostConfigs) == 0 {
		return nil
	}
	if nodeRegistryConfigs.Status.Conditions == nil {
		nodeRegistryConfigs.Status.Conditions = make([]metav1.Condition, 0)
	}
	if _, err := os.Stat(r.HostRootDir); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(r.HostRootDir, 0o755); err != nil {
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeReadDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeReadDataError),
					Message:            err.Error(),
				})
				return err
			}
		} else {
			nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
				Type:               string(configregistryv1alpha1.ConditionTypeReadDataError),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(configregistryv1alpha1.ConditionTypeReadDataError),
				Message:            err.Error(),
			})
		}
		return err
	}
	for _, host := range nodeRegistryConfigs.Spec.HostConfigs {
		parse, _ := url.Parse(host.Server)
		hostsDir := filepath.Join(r.HostRootDir, parse.Host)
		if _, err := os.Stat(hostsDir); err != nil {
			if os.IsNotExist(err) {
				if err = os.MkdirAll(hostsDir, 0o755); err != nil {
					nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
						Type:               string(configregistryv1alpha1.ConditionTypeReadDataError),
						Status:             metav1.ConditionFalse,
						LastTransitionTime: metav1.Now(),
						Reason:             string(configregistryv1alpha1.ConditionTypeReadDataError),
						Message:            err.Error(),
					})
					return err
				}
			} else {
				return err
			}
		}
		b, err := os.ReadFile(filepath.Join(hostsDir, "hosts.toml"))
		if err != nil && !os.IsNotExist(err) {
			nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
				Type:               string(configregistryv1alpha1.ConditionTypeReadDataError),
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             string(configregistryv1alpha1.ConditionTypeReadDataError),
				Message:            err.Error(),
			})
			return err
		}
		hostFileConfigs := convertNodeRegistryHostConfigToHostFileConfigs(host)
		if os.IsNotExist(err) {
			// create
			file, err := os.Create(filepath.Join(hostsDir, "hosts.toml"))
			if err != nil {
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Message:            err.Error(),
				})
				return err
			}
			if err := toml.NewEncoder(file).Encode(hostFileConfigs); err != nil {
				klog.Errorf("encode toml error: %+v", err)
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Message:            err.Error(),
				})
				return err
			}
		} else {
			// update
			var cfg Config
			if err := toml.Unmarshal(b, &cfg); err != nil {
				klog.Errorf("unmarshal toml error: %+v", err)
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeReadDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeReadDataError),
					Message:            err.Error(),
				})
				return err
			}
			if cfg.Server == host.Server && reflect.DeepEqual(cfg.Host, hostFileConfigs.Host) {
				continue
			}
			cfg.Server = host.Server
			cfg.Host = hostFileConfigs.Host
			var buf bytes.Buffer
			if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Message:            err.Error(),
				})
				return err
			}
			if err := os.WriteFile(filepath.Join(hostsDir, "hosts.toml"), buf.Bytes(), 0o644); err != nil {
				klog.Errorf("encode toml error: %+v", err)
				nodeRegistryConfigs.Status.Conditions = append(nodeRegistryConfigs.Status.Conditions, metav1.Condition{
					Type:               string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Status:             metav1.ConditionFalse,
					LastTransitionTime: metav1.Now(),
					Reason:             string(configregistryv1alpha1.ConditionTypeWriteDataError),
					Message:            err.Error(),
				})
				return err
			}
		}
	}
	return nil
}

func convertNodeRegistryHostConfigToHostFileConfigs(host configregistryv1alpha1.NodeRegistryHostConfig) Config {
	capabilities := make([]string, 0)
	for _, v := range host.Capabilities {
		capabilities = append(capabilities, string(v))
	}
	var header map[string]interface{}
	if len(host.Header) != 0 {
		header = make(map[string]interface{})
		for k, v := range host.Header {
			header[k] = v
		}
	}
	hostFileConfigs := Config{
		Server: host.Server,
		Host: &map[string]HostConfig{
			host.Server: {
				Capabilities: capabilities,
				SkipVerify:   host.SkipVerify,
				OverridePath: host.OverridePath,
				Header:       header,
			},
		},
	}
	return hostFileConfigs
}

type Config struct {
	*HostConfig
	Server string                 `toml:"server"`
	Host   *map[string]HostConfig `toml:"host,omitempty"`
}

type HostConfig struct {
	// Capabilities determine what operations a host is
	// capable of performing. Allowed values
	//  - pull
	//  - resolve
	//  - push
	Capabilities []string `toml:"capabilities"`

	// CACert are the public key certificates for TLS
	// Accepted types
	// - string - Single file with certificate(s)
	// - []string - Multiple files with certificates
	CACert interface{} `toml:"ca"`

	// Client keypair(s) for TLS with client authentication
	// Accepted types
	// - string - Single file with public and private keys
	// - []string - Multiple files with public and private keys
	// - [][2]string - Multiple keypairs with public and private keys in separate files
	Client interface{} `toml:"client"`

	// SkipVerify skips verification of the server's certificate chain
	// and host name. This should only be used for testing or in
	// combination with other methods of verifying connections.
	SkipVerify *bool `toml:"skip_verify"`

	// Header are additional header files to send to the server
	Header map[string]interface{} `toml:"header"`

	// OverridePath indicates the API root endpoint is defined in the URL
	// path rather than by the API specification.
	// This may be used with non-compliant OCI registries to override the
	// API root endpoint.
	OverridePath bool `toml:"override_path"`
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeRegistryConfigsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configregistryv1alpha1.NodeRegistryConfigs{}).
		Complete(r)
}
