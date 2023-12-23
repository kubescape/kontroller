package restapihandler

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/kubescape/go-logger"
	"github.com/kubescape/operator/config"
	"github.com/kubescape/operator/docs"
	"github.com/panjf2000/ants/v2"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
)

type HTTPHandler struct {
	keyPair    *tls.Certificate
	pool       *ants.PoolWithFunc
	config     config.IConfig
	sendReport bool
}

func NewHTTPHandler(pool *ants.PoolWithFunc, config config.IConfig) *HTTPHandler {
	return &HTTPHandler{
		keyPair:    nil,
		pool:       pool,
		config:     config,
		sendReport: config.EventReceiverURL() != "",
	}
}

// SetupHTTPListener set up listening http servers
func (resthandler *HTTPHandler) SetupHTTPListener(port string) error {
	err := resthandler.loadTLSKey()
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
	}
	if resthandler.keyPair != nil {
		server.TLSConfig = &tls.Config{Certificates: []tls.Certificate{*resthandler.keyPair}}
	}
	rtr := mux.NewRouter()
	rtr.Use(otelmux.Middleware("operator-http"))
	rtr.HandleFunc("/v1/triggerAction", resthandler.ActionRequest)

	openAPIUIHandler := docs.NewOpenAPIUIHandler()
	rtr.PathPrefix(docs.OpenAPIV2Prefix).Methods("GET").Handler(openAPIUIHandler)

	server.Handler = rtr

	logger.L().Info("Waiting for REST API to receive notifications, port: " + port)

	// listen
	if resthandler.keyPair != nil {
		return server.ListenAndServeTLS("", "")
	} else {
		return server.ListenAndServe()
	}
}
