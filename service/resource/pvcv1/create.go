package pvcv1

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
	pvcsToCreate, err := toPVCs(createChange)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(pvcsToCreate) != 0 {
		r.logger.LogCtx(ctx, "debug", "creating the PVCs in the Kubernetes API")

		namespace := keyv1.ClusterNamespace(customObject)
		for _, PVC := range pvcsToCreate {
			_, err := r.k8sClient.Core().PersistentVolumeClaims(namespace).Create(PVC)
			if apierrors.IsAlreadyExists(err) {
				// fall through
			} else if err != nil {
				return microerror.Mask(err)
			}
		}

		r.logger.LogCtx(ctx, "debug", "created the PVCs in the Kubernetes API")
	} else {
		r.logger.LogCtx(ctx, "debug", "the PVCs do not need to be created in the Kubernetes API")
	}

	return nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	currentPVCs, err := toPVCs(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredPVCs, err := toPVCs(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "debug", "finding out which PVCs have to be created")

	var pvcsToCreate []*apiv1.PersistentVolumeClaim

	for _, desiredPVC := range desiredPVCs {
		if !containsPVC(currentPVCs, desiredPVC) {
			pvcsToCreate = append(pvcsToCreate, desiredPVC)
		}
	}

	r.logger.LogCtx(ctx, "debug", fmt.Sprintf("found %d PVCs that have to be created", len(pvcsToCreate)))

	return pvcsToCreate, nil
}
