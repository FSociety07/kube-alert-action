package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// AlertPayload is the expected shape of an incoming alert.
type AlertPayload struct {
	Namespace string `json:"namespace"`
	Container string `json:"container"`
	Pod       string `json:"pod"`
	Metric    string `json:"metric"`
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
		log.Info("Invalid method: ", r.Method)
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

	// TODO 3: validate — what counts as "invalid"? empty namespace/pod/metric?
	//         decide what status code to return for bad payloads (400?)
	if alertpayLoad.Container == "" || alertpayLoad.Namespace == "" || alertpayLoad.Pod == "" || alertpayLoad.Metric == "" {
		log.Info("Alert Payload is incomplete", "namespace", alertpayLoad.Namespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)
		http.Error(w, "Alert Payload is incomplete: "+fmt.Sprintf("%+v\n", alertpayLoad), 400)
		return
	}
	// TODO 4: on success, log the parsed alert and respond 200/202
	//         (actual CRD matching + AlertEvent creation comes in the next step —
	//         don't wire that in yet)

	w.WriteHeader(202)
	log.Info("received alert", "namespace", alertpayLoad.Namespace, "container", alertpayLoad.Container, "pod", alertpayLoad.Pod, "metric", alertpayLoad.Metric)
}
