package controller

import (
	"context"
	"go_heap/api/v1alpha1"
	"os"
	"os/exec"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type AlertEventReconciler struct {
	Client     client.Client
	RestConfig *rest.Config
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

	if alertEvent.Status.Phase != v1alpha1.PhasePending {
		if alertEvent.Status.Phase == v1alpha1.PhaseInProgress {
			log.Info("Action already in progress", "name", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		log.Info("Action already executed", "name", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	alertEvent.Status.Phase = v1alpha1.PhaseInProgress
	if err := r.Client.Status().Update(ctx, &alertEvent); err != nil {
		log.Error(err, "Error updating status to InProgress for AlertEvent CR", "name", req.NamespacedName)
		return reconcile.Result{}, err
	}

	//TODO write a checkpoint to ensure that the script alertEvent.Spec.Action exists. error and return if not

	switch alertEvent.Spec.ExecuteFrom {
	case "self":
		cmd := exec.CommandContext(ctx, alertEvent.Spec.Action, alertEvent.Spec.TargetNamespace, alertEvent.Spec.Pod)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Error(err, "Error executing action for AlertEvent", "name", req.NamespacedName, "error", err)
			alertEvent.Status.Phase = v1alpha1.PhaseFailed
			alertEvent.Status.Message = err.Error() + ": " + string(output)
			if err1 := r.Client.Status().Update(ctx, &alertEvent); err1 != nil {
				log.Error(err1, "Error updating status message for AlertEvent", "name", req.NamespacedName)
				return reconcile.Result{}, err1
			}
			return reconcile.Result{}, err
		}
		log.Info("Action successfully executed for AlertEvent", "name", req.NamespacedName, "output", string(output))
		alertEvent.Status.Phase = v1alpha1.PhaseCompleted
		alertEvent.Status.Message = string(output)
		if err := r.Client.Status().Update(ctx, &alertEvent); err != nil {
			log.Error(err, "Error updating status to Completed for AlertEvent", "name", req.NamespacedName, "error", err)
			return reconcile.Result{}, err
		}

	case "targetPod":
		clientset, err := kubernetes.NewForConfig(r.RestConfig)
		if err != nil {
			log.Error(err, "failed to create clientset", "name", req.NamespacedName)
			return reconcile.Result{}, err
		}
		execReq := clientset.CoreV1().RESTClient().Post().
			Resource("pods").
			Name(alertEvent.Spec.Pod).
			Namespace(alertEvent.Spec.TargetNamespace).
			SubResource("exec")

		cmd := []string{alertEvent.Spec.Action, alertEvent.Spec.TargetNamespace, alertEvent.Spec.Pod}

		// Container field is omitted.
		option := corev1.PodExecOptions{
			Command: cmd,
			Stdout:  true,
			Stderr:  true,
		}

		execReq = execReq.VersionedParams(&option, scheme.ParameterCodec)

		exec, err := remotecommand.NewWebSocketExecutor(r.RestConfig, "POST", execReq.URL().String())
		if err != nil {
			log.Error(err, "failed to create executor", "name", req.NamespacedName)
			return reconcile.Result{}, err
		}

		err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: os.Stdout, Stderr: os.Stderr})
		if err != nil {
			log.Error(err, "failed to stream exec", "name", req.NamespacedName)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
