package deploymentv1

import (
	"fmt"

	"github.com/giantswarm/kvmtpr"
	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/resource"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	extensionsv1 "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/giantswarm/kvm-operator/service/keyv1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newWorkerDeployments(customObject kvmtpr.CustomObject) ([]*extensionsv1.Deployment, error) {
	var deployments []*extensionsv1.Deployment

	privileged := true
	replicas := int32(1)

	for i, workerNode := range customObject.Spec.Cluster.Workers {
		capabilities := customObject.Spec.KVM.Workers[i]

		cpuQuantity, err := keyv1.CPUQuantity(capabilities)
		if err != nil {
			return nil, microerror.Maskf(err, "creating CPU quantity")
		}

		memoryQuantity, err := keyv1.MemoryQuantity(capabilities)
		if err != nil {
			return nil, microerror.Maskf(err, "creating memory quantity")
		}

		deployment := &extensionsv1.Deployment{
			TypeMeta: apismetav1.TypeMeta{
				Kind:       "deployment",
				APIVersion: "extensions/v1beta",
			},
			ObjectMeta: apismetav1.ObjectMeta{
				Name: keyv1.DeploymentName(keyv1.WorkerID, workerNode.ID),
				Annotations: map[string]string{
					VersionBundleVersionAnnotation: keyv1.VersionBundleVersion(customObject),
				},
				Labels: map[string]string{
					"app":      keyv1.WorkerID,
					"cluster":  keyv1.ClusterID(customObject),
					"customer": keyv1.ClusterCustomer(customObject),
					"node":     workerNode.ID,
				},
			},
			Spec: extensionsv1.DeploymentSpec{
				Strategy: extensionsv1.DeploymentStrategy{
					Type: extensionsv1.RecreateDeploymentStrategyType,
				},
				Replicas: &replicas,
				Template: apiv1.PodTemplateSpec{
					ObjectMeta: apismetav1.ObjectMeta{
						Name: keyv1.WorkerID,
						Labels: map[string]string{
							"cluster":  keyv1.ClusterID(customObject),
							"customer": keyv1.ClusterCustomer(customObject),
							"app":      keyv1.WorkerID,
							"node":     workerNode.ID,
						},
						Annotations: map[string]string{},
					},
					Spec: apiv1.PodSpec{
						Affinity:    newWorkerPodAfinity(customObject),
						HostNetwork: true,
						NodeSelector: map[string]string{
							"role": keyv1.WorkerID,
						},
						Volumes: []apiv1.Volume{
							{
								Name: "cloud-config",
								VolumeSource: apiv1.VolumeSource{
									ConfigMap: &apiv1.ConfigMapVolumeSource{
										LocalObjectReference: apiv1.LocalObjectReference{
											Name: keyv1.ConfigMapName(customObject, workerNode, keyv1.WorkerID),
										},
									},
								},
							},
							{
								Name: "images",
								VolumeSource: apiv1.VolumeSource{
									HostPath: &apiv1.HostPathVolumeSource{
										Path: "/home/core/images/",
									},
								},
							},
							{
								Name: "rootfs",
								VolumeSource: apiv1.VolumeSource{
									EmptyDir: &apiv1.EmptyDirVolumeSource{},
								},
							},
							{
								Name: "flannel",
								VolumeSource: apiv1.VolumeSource{
									HostPath: &apiv1.HostPathVolumeSource{
										Path: keyv1.FlannelEnvPathPrefix,
									},
								},
							},
						},
						Containers: []apiv1.Container{
							{
								Name:            "k8s-endpoint-updater",
								Image:           customObject.Spec.KVM.EndpointUpdater.Docker.Image,
								ImagePullPolicy: apiv1.PullIfNotPresent,
								Command: []string{
									"/bin/sh",
									"-c",
									"/opt/k8s-endpoint-updater update --provider.bridge.name=" + keyv1.NetworkBridgeName(customObject) +
										" --service.kubernetes.cluster.namespace=" + keyv1.ClusterNamespace(customObject) +
										" --service.kubernetes.cluster.service=" + keyv1.WorkerID +
										" --service.kubernetes.inCluster=true" +
										" --service.kubernetes.pod.name=${POD_NAME}",
								},
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								Env: []apiv1.EnvVar{
									{
										Name: "POD_NAME",
										ValueFrom: &apiv1.EnvVarSource{
											FieldRef: &apiv1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.name",
											},
										},
									},
								},
							},
							{
								Name:            "k8s-kvm",
								Image:           customObject.Spec.KVM.K8sKVM.Docker.Image,
								ImagePullPolicy: apiv1.PullIfNotPresent,
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								Args: []string{
									keyv1.WorkerID,
								},
								Env: []apiv1.EnvVar{
									{
										Name:  "CORES",
										Value: fmt.Sprintf("%d", capabilities.CPUs),
									},
									{
										Name:  "DISK",
										Value: fmt.Sprintf("%.0fG", capabilities.Disk),
									},
									{
										Name: "HOSTNAME",
										ValueFrom: &apiv1.EnvVarSource{
											FieldRef: &apiv1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.name",
											},
										},
									},
									{
										Name:  "NETWORK_BRIDGE_NAME",
										Value: keyv1.NetworkBridgeName(customObject),
									},
									{
										Name:  "NETWORK_TAP_NAME",
										Value: keyv1.NetworkTapName(customObject),
									},
									{
										Name: "MEMORY",
										// TODO provide memory like disk as float64 and format here.
										Value: capabilities.Memory,
									},
									{
										Name:  "ROLE",
										Value: keyv1.WorkerID,
									},
									{
										Name:  "CLOUD_CONFIG_PATH",
										Value: "/cloudconfig/user_data",
									},
								},
								LivenessProbe: &apiv1.Probe{
									InitialDelaySeconds: keyv1.InitialDelaySeconds,
									TimeoutSeconds:      keyv1.TimeoutSeconds,
									PeriodSeconds:       keyv1.PeriodSeconds,
									FailureThreshold:    keyv1.FailureThreshold,
									SuccessThreshold:    keyv1.SuccessThreshold,
									Handler: apiv1.Handler{
										HTTPGet: &apiv1.HTTPGetAction{
											Path: keyv1.HealthEndpoint,
											Port: intstr.IntOrString{IntVal: keyv1.LivenessPort(customObject)},
											Host: keyv1.ProbeHost,
										},
									},
								},
								Resources: apiv1.ResourceRequirements{
									Requests: map[apiv1.ResourceName]resource.Quantity{
										apiv1.ResourceCPU:    cpuQuantity,
										apiv1.ResourceMemory: memoryQuantity,
									},
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      "cloud-config",
										MountPath: "/cloudconfig/",
									},
									{
										Name:      "images",
										MountPath: "/usr/code/images/",
									},
									{
										Name:      "rootfs",
										MountPath: "/usr/code/rootfs/",
									},
								},
							},
							{
								Name:            "k8s-kvm-health",
								Image:           keyv1.K8SKVMHealthDocker,
								ImagePullPolicy: apiv1.PullAlways,
								Env: []apiv1.EnvVar{
									{
										Name:  "LISTEN_ADDRESS",
										Value: keyv1.HealthListenAddress(customObject),
									},
									{
										Name:  "NETWORK_ENV_FILE_PATH",
										Value: keyv1.NetworkEnvFilePath(customObject),
									},
								},
								SecurityContext: &apiv1.SecurityContext{
									Privileged: &privileged,
								},
								VolumeMounts: []apiv1.VolumeMount{
									{
										Name:      "flannel",
										MountPath: keyv1.FlannelEnvPathPrefix,
									},
								},
							},
						},
					},
				},
			},
		}

		deployments = append(deployments, deployment)
	}

	return deployments, nil
}
