package controller

import (
	"context"
	"go_heap/api/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AlertEventReconciler struct {
	Client client.Client
}

func (r *AlertEventReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := logf.FromContext(ctx)

	var alertEvent v1alpha1.AlertEvent
	if err := r.Client.Get(ctx, req.NamespacedName, &alertEvent); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Reconcilation triggered but AlertEvent CR not found", "name", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		log.Error(err, "Error while retrieving AlertEvent CR", "name", req.NamespacedName)
		return reconcile.Result{}, err
	}

	if alertEvent.Status.Phase == v1alpha1.PhasePending {
		//TODO: execute script
	} else if alertEvent.Status.Phase == v1alpha1.PhaseInProgress {
		log.Info("Action already in progress", "name", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}
