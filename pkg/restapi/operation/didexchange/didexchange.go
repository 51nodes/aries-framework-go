/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package didexchange

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/hyperledger/aries-framework-go/pkg/client/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/dispatcher"
	didexchange2 "github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/internal/common/support"
	"github.com/hyperledger/aries-framework-go/pkg/restapi/operation"
	"github.com/hyperledger/aries-framework-go/pkg/restapi/operation/didexchange/models"
	"github.com/hyperledger/aries-framework-go/pkg/wallet"
)

var logger = log.New("aries-framework/did-exchange")

const (
	operationID           = "/connections"
	createInvitationPath  = operationID + "/create-invitation"
	receiveInvtiationPath = operationID + "/receive-invitation"
	acceptInvitationPath  = operationID + "/{id}/accept-invitation"
	connections           = operationID
	connectionsByID       = operationID + "/{id}"
	acceptExchangeRequest = operationID + "/{id}/accept-request"
)

// provider contains dependencies for the Exchange protocol and is typically created by using aries.Context()
type provider interface {
	Service(id string) (interface{}, error)
	CryptoWallet() wallet.Crypto
}

// New returns new DID Exchange rest client protocol instance
func New(ctx provider) (*Operation, error) {

	didExchange, err := didexchange.New(ctx)
	if err != nil {
		return nil, err
	}

	service, err := ctx.Service(didexchange2.DIDExchange)
	if err != nil {
		return nil, err
	}

	didexchangeSvc, ok := service.(dispatcher.Service)
	if !ok {
		return nil, errors.New("failed to lookup didexchange service from context")
	}

	svc := &Operation{ctx: ctx, client: didExchange, service: didexchangeSvc}
	svc.registerHandler()

	return svc, nil
}

// Operation is controller REST service controller for DID Exchange
type Operation struct {
	ctx      provider
	client   *didexchange.Client
	service  dispatcher.Service
	handlers []operation.Handler
}

// CreateInvitation swagger:route GET /connections/create-invitation did-exchange createInvitation
//
// Creates a new connection invitation....
//
// Responses:
//    default: genericError
//        200: createInvitationResponse
func (c *Operation) CreateInvitation(rw http.ResponseWriter, req *http.Request) {

	logger.Debugf("Creating connection invitation ")
	// call didexchange client
	response, err := c.client.CreateInvitation()
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	err = json.NewEncoder(rw).Encode(&models.CreateInvitationResponse{Payload: response})
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// ReceiveInvitation swagger:route POST /connections/receive-invitation did-exchange receiveInvitation
//
// Receive a new connection invitation....
//
// Responses:
//    default: genericError
//        200: receiveInvitationResponse
func (c *Operation) ReceiveInvitation(rw http.ResponseWriter, req *http.Request) {

	logger.Debugf("Receiving connection invitation ")

	var request models.ReceiveInvitationRequest
	err := json.NewDecoder(req.Body).Decode(&request.Params)
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	payload, err := json.Marshal(request.Params)
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	err = c.service.Handle(dispatcher.DIDCommMsg{Type: request.Params.Type, Payload: payload})
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	// TODO returning sample response since listener on DID exchange service is still need to be implemented
	sampleResponse := models.ReceiveInvitationResponse{
		ConnectionID:  "f52024c4-04e7-4aeb-8486-1040155c6764",
		DID:           "TAaW9Dmxa93B8e5x6iLwFJ",
		State:         "requested",
		CreateTime:    time.Now(),
		UpdateTime:    time.Now(),
		Accept:        "auto",
		Initiator:     "external",
		InvitationKey: "none",
		InviterLabel:  "other party",
		Mode:          "none",
		RequestID:     "678ad4b6-4e2b-40a1-804e-8ba504945e26",
		RoutingState:  "none",
	}

	err = json.NewEncoder(rw).Encode(sampleResponse)
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// AcceptInvitation swagger:route GET /connections/{id}/accept-invitation did-exchange acceptInvitation
//
// Accept a stored connection invitation....
//
// Responses:
//    default: genericError
//        200: acceptInvitationResponse
func (c *Operation) AcceptInvitation(rw http.ResponseWriter, req *http.Request) {

	params := mux.Vars(req)
	logger.Debugf("Accepting connection invitation for id[%s]", params["id"])

	// TODO returning sample response since event listening/handling with DID exchange service needs to be implemented
	response := models.AcceptInvitationResponse{
		ConnectionID:  params["id"],
		DID:           "TAaW9Dmxa93B8e5x6iLwFJ",
		State:         "requested",
		CreateTime:    time.Now(),
		UpdateTime:    time.Now(),
		Accept:        "auto",
		Initiator:     "external",
		InvitationKey: "none",
		InviterLabel:  "other party",
		Mode:          "none",
		RequestID:     "678ad4b6-4e2b-40a1-804e-8ba504945e26",
		RoutingState:  "none",
	}

	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// QueryConnections swagger:route GET /connections did-exchange queryConnections
//
// query agent to agent connections.
//
// Responses:
//    default: genericError
//        200: queryConnectionsResponse
func (c *Operation) QueryConnections(rw http.ResponseWriter, req *http.Request) {
	logger.Debugf("Querying connection invitations ")

	var request models.QueryConnections
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	results, err := c.client.QueryConnections(&request.QueryConnectionsParams)
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	response := models.QueryConnectionsResponse{
		Body: struct {
			Results []*didexchange.QueryConnectionResult `json:"results"`
		}{
			Results: results,
		},
	}

	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// QueryConnectionByID swagger:route GET /connections/{id} did-exchange getConnection
//
// Fetch a single connection record.
//
// Responses:
//    default: genericError
//        200: queryConnectionResponse
func (c *Operation) QueryConnectionByID(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	logger.Debugf("Querying connection invitation for id [%s]", params["id"])

	result, err := c.client.QueryConnectionByID(params["id"])
	if err != nil {
		c.writeGenericError(rw, err)
		return
	}

	response := models.QueryConnectionResponse{
		Result: result,
	}

	err = json.NewEncoder(rw).Encode(response)
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// AcceptExchangeRequest swagger:route GET /connections/{id}/accept-request did-exchange acceptRequest
//
// Accepts a stored connection request.
//
// Responses:
//    default: genericError
//        200: acceptExchangeResponse
func (c *Operation) AcceptExchangeRequest(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	logger.Debugf("Accepting connection request for id [%s]", params["id"])

	// TODO returning sample response below, Accept Exchange Request to be added using events & callback (#198 & #238)
	result := &models.ExchangeResponse{
		ConnectionID: uuid.New().String(), CreatedTime: time.Now(),
	}

	response := models.AcceptExchangeResult{Result: result}

	err := json.NewEncoder(rw).Encode(response)
	if err != nil {
		logger.Errorf("Unable to write response, %s", err)
	}
}

// writeGenericError writes given error to writer as generic error response
func (c *Operation) writeGenericError(rw io.Writer, err error) {
	errResponse := models.GenericError{
		Body: struct {
			Code    int32  `json:"code"`
			Message string `json:"message"`
		}{
			// TODO implement error codes, below is sample error code
			Code:    1,
			Message: err.Error(),
		},
	}
	err = json.NewEncoder(rw).Encode(errResponse)
	if err != nil {
		logger.Errorf("Unable to send error response, %s", err)
	}
}

// GetRESTHandlers get all controller API handler available for this protocol service
func (c *Operation) GetRESTHandlers() []operation.Handler {
	return c.handlers
}

// registerHandler register handlers to be exposed from this protocol service as REST API endpoints
func (c *Operation) registerHandler() {
	// Add more protocol endpoints here to expose them as controller API endpoints
	c.handlers = []operation.Handler{
		support.NewHTTPHandler(createInvitationPath, http.MethodGet, c.CreateInvitation),
		support.NewHTTPHandler(receiveInvtiationPath, http.MethodPost, c.ReceiveInvitation),
		support.NewHTTPHandler(acceptInvitationPath, http.MethodGet, c.AcceptInvitation),
		support.NewHTTPHandler(connections, http.MethodGet, c.QueryConnections),
		support.NewHTTPHandler(connectionsByID, http.MethodGet, c.QueryConnectionByID),
		support.NewHTTPHandler(acceptExchangeRequest, http.MethodGet, c.AcceptExchangeRequest),
	}
}
