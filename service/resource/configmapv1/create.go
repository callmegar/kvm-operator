package configmapv1

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiv1 "k8s.io/client-go/pkg/api/v1"

	"github.com/giantswarm/kvm-operator/service/keyv1"
)

func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createChange interface{}) error {
	customObject, err := keyv1.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	configMapsToCreate, err := toConfigMaps(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	// Create the config maps in the Kubernetes API.
	if len(configMapsToCreate) != 0 {
		r.logger.LogCtx(ctx, "debug", "creating the config maps in the Kubernetes API")

		namespace := keyv1.ClusterNamespace(customObject)
		for _, configMap := range configMapsToCreate {
			_, err := r.k8sClient.CoreV1().ConfigMaps(namespace).Create(configMap)
			if apierrors.IsAlreadyExists(err) {
				// fall through
			} else if err != nil {
				return microerror.Mask(err)
			}
		}

		r.logger.LogCtx(ctx, "debug", "created the config maps in the Kubernetes API")
	} else {
		r.logger.LogCtx(ctx, "debug", "the config maps do not need to be created in the Kubernetes API")
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentConfigMaps, err := toConfigMaps(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredConfigMaps, err := toConfigMaps(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "debug", "finding out which config maps have to be created")

	var configMapsToCreate []*apiv1.ConfigMap

	for _, desiredConfigMap := range desiredConfigMaps {
		if !containsConfigMap(currentConfigMaps, desiredConfigMap) {
			configMapsToCreate = append(configMapsToCreate, desiredConfigMap)
		}
	}

	r.logger.LogCtx(ctx, "debug", fmt.Sprintf("found %d config maps that have to be created", len(configMapsToCreate)))

	return configMapsToCreate, nil
}
