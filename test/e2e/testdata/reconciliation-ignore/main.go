package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jenkinsci/kubernetes-operator/test/e2e/testdata/reconciliation-ignore/identitywebhook"
	admissionv1 "k8s.io/api/admission/v1"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/mutate", func(w http.ResponseWriter, r *http.Request) {
		var review admissionv1.AdmissionReview
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&review); err != nil || review.Request == nil {
			http.Error(w, "invalid admission review", http.StatusBadRequest)
			return
		}
		review.Response = identitywebhook.Mutate(*review.Request)
		review.Request = nil
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(review); err != nil {
			log.Print(err)
		}
	})
	server := &http.Server{Addr: ":8443", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Fatal(server.ListenAndServeTLS("/tls/tls.crt", "/tls/tls.key"))
}
