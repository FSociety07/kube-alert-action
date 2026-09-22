package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"go_heap/api/v1alpha1"
	"go_heap/internal/notify"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// alert body
type AlertPayload struct {
	TargetNamespace string `json:"targetNamespace"`
	Container       string `json:"container"`
	Pod             string `json:"pod"`
	Metric          string `json:"metric"`
}

// Server implements manager.Runnable so it can be registered with mgr.Add().
type Server struct {
	Client   client.Client
	Addr     string
	Notifier notify.Notifier
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/alert", s.handleAlert)

	srv := &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		d := time.Now().Add(15 * time.Second)
		ctxD, cancel := context.WithDeadline(context.Background(), d)
		defer cancel()
		err := srv.Shutdown(ctxD)
		return err
	case err := <-errCh:
		return err
	}

}

func (s *Server) handleAlert(w http.ResponseWriter, r *http.Request) {
	log := logf.FromContext(r.Context())

	if r.Method != http.MethodPost {
		log.Error(nil, "Invalid method:", "method", r.Method)
		http.Error(w, "Invalid method", 405)
		return
	}

	var alertpayLoad AlertPayload
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&alertpayLoad); err != nil {
		log.Error(err, "failed to decode alert payload")
		http.Error(w, "Malformed alert: "+err.Error(), 400)
		return
	}

	GChatThreadKey := notify.ThreadKey(alertpayLoad.Container, alertpayLoad.Metric)

	log.Info("received alert", "targetNamespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)
	msg := fmt.Sprintf("*Alert received*: %s/%s metric=%s", alertpayLoad.TargetNamespace, alertpayLoad.Pod, alertpayLoad.Metric)
	if err := s.Notifier.SendMessage(r.Context(), msg, GChatThreadKey); err != nil {
		log.Error(err, "Failed to send Chat notification")
	}

	if alertpayLoad.Container == "" || alertpayLoad.TargetNamespace == "" || alertpayLoad.Pod == "" || alertpayLoad.Metric == "" {
		log.Error(nil, "Alert Payload is incomplete", "targetNamespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)
		http.Error(w, "Alert Payload is incomplete: "+fmt.Sprintf("%+v\n", alertpayLoad), 400)
		return
	}

	matchFound, action, executeFrom, err := s.matchRule(r.Context(), alertpayLoad.Metric)
	if err != nil {
		log.Error(err, "Unable to reach the cluster")
		http.Error(w, "Unable to reach the cluster: "+err.Error(), 500)
		return
	}

	if !matchFound {
		log.Info("no match found for metric", "metric", alertpayLoad.Metric)

	} else {
		log.Info("AlertEvent CR to be created", "action", action, "executeFrom", executeFrom)
		msg := fmt.Sprintf("*Match found for alert metric* metric:%s action:%s executeFrom:%s", alertpayLoad.Metric, action, executeFrom)
		if err := s.Notifier.SendMessage(r.Context(), msg, GChatThreadKey); err != nil {
			log.Error(err, "Failed to send Chat notification")
		}
		var alertEvent v1alpha1.AlertEvent
		if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: alertpayLoad.TargetNamespace, Name: alertpayLoad.Container + "-" + alertpayLoad.Metric}, &alertEvent); err == nil {
			log.Info("AlertEvent CR already exists", "namespace", alertpayLoad.TargetNamespace, "name", alertpayLoad.Container+"-"+alertpayLoad.Metric)
			msg := fmt.Sprintf("*AlertEvent CR already exists* namespace:%s name:%s", alertpayLoad.TargetNamespace, alertpayLoad.Container+"-"+alertpayLoad.Metric)
			if err := s.Notifier.SendMessage(r.Context(), msg, GChatThreadKey); err != nil {
				log.Error(err, "Failed to send Chat notification")
			}
		} else if apierrors.IsNotFound(err) {
			log.Info("Creating AlertEvent CR", "namespace", alertpayLoad.TargetNamespace, "name", alertpayLoad.Container+"-"+alertpayLoad.Metric)
			CreateAlertEvent := &v1alpha1.AlertEvent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      alertpayLoad.Container + "-" + alertpayLoad.Metric,
					Namespace: alertpayLoad.TargetNamespace,
				},
				Spec: v1alpha1.AlertEventSpec{
					TargetNamespace: alertpayLoad.TargetNamespace,
					Container:       alertpayLoad.Container,
					Pod:             alertpayLoad.Pod,
					Metric:          alertpayLoad.Metric,
					Action:          action,
					ExecuteFrom:     executeFrom,
				},
				Status: v1alpha1.AlertEventStatus{ //does not work consistently. status is set below after CR creation
					Phase: v1alpha1.PhasePending,
				},
			}
			if err := s.Client.Create(r.Context(), CreateAlertEvent); err != nil {
				log.Error(err, "Unable to create AlertEvent CR")
				http.Error(w, "Unable to create AlertEvent CR: "+err.Error(), 500)
				return
			} else {
				CreateAlertEvent.Status.Phase = v1alpha1.PhasePending
				if err := s.Client.Status().Update(r.Context(), CreateAlertEvent); err != nil {
					log.Error(err, "Unable to set initial status for AlertEvent CR")
					http.Error(w, "Unable to set initial status: "+err.Error(), 500)
					return
				}
				log.Info("AlertEvent CR created", "namespace", alertpayLoad.TargetNamespace, "name", alertpayLoad.Container+"-"+alertpayLoad.Metric)
				msg := fmt.Sprintf("*AlertEvent CR created* namespace:%s name:%s", alertpayLoad.TargetNamespace, alertpayLoad.Container+"-"+alertpayLoad.Metric)
				if err := s.Notifier.SendMessage(r.Context(), msg, GChatThreadKey); err != nil {
					log.Error(err, "Failed to send Chat notification")
				}
			}
		} else {
			log.Error(err, "Unable to query the cluster on listing AlertEvents")
			http.Error(w, "Unable to query the cluster on listing AlertEvents: "+err.Error(), 500)
			return
		}
	}

	w.WriteHeader(202)
}

func (s *Server) matchRule(ctx context.Context, payLoadMetric string) (bool, string, string, error) {
	log := logf.FromContext(ctx)
	var actionMapList v1alpha1.ActionMapList
	if err := s.Client.List(ctx, &actionMapList); err != nil {
		return false, "", "", err
	}

	var action string
	var executeFrom string
	matchFound := false

out:
	for _, v := range actionMapList.Items {
		for _, rule := range v.Spec.Rules {
			if payLoadMetric == rule.Metric {
				log.Info("alert metric matched", "metric", payLoadMetric, "action", rule.Action, "executeFrom", rule.ExecuteFrom)
				action = rule.Action
				executeFrom = rule.ExecuteFrom
				matchFound = true
				break out
			}
		}
	}

	if !matchFound {
		return false, "", "", nil
	}

	return true, action, executeFrom, nil

}
