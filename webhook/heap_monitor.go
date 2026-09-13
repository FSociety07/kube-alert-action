package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"go_heap/api/v1alpha1"
	"net/http"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// AlertPayload is the expected shape of an incoming alert.
type AlertPayload struct {
	TargetNamespace string `json:"targetNamespace"`
	Container       string `json:"container"`
	Pod             string `json:"pod"`
	Metric          string `json:"metric"`
}

// Server implements manager.Runnable so it can be registered with mgr.Add().
type Server struct {
	Client client.Client // shared controller-runtime client, injected at construction
	Addr   string        // e.g. ":8080"
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

	// TODO 1: reject anything that isn't POST (http.MethodPost)
	if r.Method != http.MethodPost {
		log.Info("Invalid method:", "method", r.Method)
		http.Error(w, "Invalid method", 405)
		return
	}

	// TODO 2: decode JSON body into AlertPayload using json.NewDecoder(r.Body).Decode(&payload)
	var alertpayLoad AlertPayload
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&alertpayLoad); err != nil {
		log.Error(err, "failed to decode alert payload")
		http.Error(w, "Malformed alert: "+err.Error(), 400)
		return
	}

	// TODO 3: validate — what counts as "invalid"? empty TargetNamespace/pod/metric?
	//         decide what status code to return for bad payloads (400?)
	if alertpayLoad.Container == "" || alertpayLoad.TargetNamespace == "" || alertpayLoad.Pod == "" || alertpayLoad.Metric == "" {
		log.Info("Alert Payload is incomplete", "targetNamespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)
		http.Error(w, "Alert Payload is incomplete: "+fmt.Sprintf("%+v\n", alertpayLoad), 400)
		return
	}
	// TODO 4: on success, log the parsed alert and respond 200/202
	//         (actual CRD matching + AlertEvent creation comes in the next step —
	//         don't wire that in yet)

	log.Info("received alert", "targetNamespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)

	matchFound, action, executeFrom, err := s.matchRule(r.Context(), alertpayLoad.Metric)
	if err != nil {
		http.Error(w, "Unable to reach the cluster: "+err.Error(), 500)
		return
	}

	if !matchFound {
		log.Info("no match found for metric", "metric", alertpayLoad.Metric)

	} else {
		log.Info("AlertEvent CRD to be created", "action", action, "executeFrom", executeFrom)
		var alertEvent v1alpha1.AlertEvent
		if err := s.Client.Get(r.Context(), client.ObjectKey{Namespace: alertpayLoad.TargetNamespace, Name: alertpayLoad.Container + "-" + alertpayLoad.Metric}, &alertEvent); err == nil {
			log.Info("CRD already exists", "namespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "metric", alertpayLoad.Metric)
		} else if apierrors.IsNotFound(err) {
			log.Info("Creating CRD", "namespace", alertpayLoad.TargetNamespace, "container", alertpayLoad.Container, "metric", alertpayLoad.Metric)
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
			}
			if err := s.Client.Create(r.Context(), CreateAlertEvent); err != nil {
				http.Error(w, "Unable to create AlertEvent: "+err.Error(), 500)
				log.Error(err, "Unable to create AlertEvent")
				return
			}
		} else {
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
				log.Info("alert metric matched", "metric", payLoadMetric, "action", rule.Action)
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
