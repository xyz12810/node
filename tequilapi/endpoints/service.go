/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// swagger:model ServiceRequestDTO
type serviceRequest struct {
	// provider identity
	// required: true
	// example: 0x0000000000000000000000000000000000000002
	ProviderID string `json:"providerId"`

	// provider identity passphrase
	// required: true
	Passphrase string `json:"passphrase"`

	// service type. Possible values are "openvpn", "wireguard" and "noop"
	// required: false
	// default: openvpn
	// example: openvpn
	ServiceType string `json:"serviceType"`

	Options json.RawMessage `json:"options"`
}

// swagger:model ServiceListDTO
type serviceList []serviceInfo

// swagger:model ServiceInfoDTO
type serviceInfo struct {
	// example: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
	ID       string      `json:"id"`
	Proposal proposalRes `json:"proposal"`
	// example: Running
	Status  string         `json:"status"`
	Options serviceOptions `json:"options"`
}

type serviceOptions struct {
	// example: UDP
	Protocol string `json:"protocol"`
	// example: 1190
	Port int `json:"port"`
}

// ServiceEndpoint struct represents /service resource and it's sub-resources
type ServiceEndpoint struct {
	serviceManager  ServiceManager
	identityManager identity.Manager
	optionsParser   map[string]func(json.RawMessage) (service.Options, error)
}

// NewServiceEndpoint creates and returns service endpoint
func NewServiceEndpoint(serviceManager ServiceManager, identityManager identity.Manager, optionsParser map[string]func(json.RawMessage) (service.Options, error)) *ServiceEndpoint {
	return &ServiceEndpoint{
		serviceManager:  serviceManager,
		optionsParser:   optionsParser,
		identityManager: identityManager,
	}
}

// ServiceList provides a list of running services on the node.
// swagger:operation GET /services Service serviceList
// ---
// summary: List of services
// description: ServiceList provides a list of running services on the node.
// responses:
//   200:
//     description: List of running services
//     schema:
//       "$ref": "#/definitions/ServiceListDTO"
func (se *ServiceEndpoint) ServiceList(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	instances := se.serviceManager.List()
	statusResponse := toServiceListResponse(instances)
	utils.WriteAsJSON(statusResponse, resp)
}

// ServiceGet provides info for requested service on the node.
// swagger:operation GET /services/:id Service serviceGet
// ---
// summary: Information about service
// description: ServiceGet provides info for requested service on the node.
// responses:
//   200:
//     description: Service detailed information
//     schema:
//       "$ref": "#/definitions/ServiceInfoDTO"
//   404:
//     description: Service not found
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceGet(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	for _, instance := range se.serviceManager.List() {
		if instance.ID() == service.ID(params.ByName("id")) {
			statusResponse := toServiceInfoResponse(*instance)
			utils.WriteAsJSON(statusResponse, resp)
			return
		}
	}

	utils.SendErrorMessage(resp, "Requested service not found", http.StatusNotFound)
}

// ServiceStart starts requested service on the node.
// swagger:operation POST /services Service serviceStart
// ---
// summary: Starts service
// description: Provider starts serving new service to consumers
// parameters:
//   - in: body
//     name: body
//     description: Parameters in body (providerID) required for starting new service
//     schema:
//       $ref: "#/definitions/ServiceRequestDTO"
// responses:
//   201:
//     description: Initiates service start
//     schema:
//       "$ref": "#/definitions/ServiceInfoDTO"
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   409:
//     description: Conflict. Service is already running
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   499:
//     description: Service was cancelled
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceStart(resp http.ResponseWriter, req *http.Request, params httprouter.Params) {
	sr, err := toServiceRequest(req)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	errorMap := validateServiceRequest(sr)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}

	optionsParser, ok := se.optionsParser[sr.ServiceType]
	if !ok {
		utils.SendErrorMessage(resp, "Invalid service type", http.StatusBadRequest)
		return
	}

	options, err := optionsParser(sr.Options)
	if err != nil {
		utils.SendErrorMessage(resp, "Invalid options", http.StatusBadRequest)
		return
	}

	for _, instance := range se.serviceManager.List() {
		proposal := instance.Proposal()
		if proposal.ProviderID == sr.ProviderID && proposal.ServiceType == sr.ServiceType {
			utils.SendErrorMessage(resp, "Service already running", http.StatusConflict)
			return
		}
	}

	if err := se.identityManager.Unlock(sr.ProviderID, sr.Passphrase); err != nil {
		utils.SendErrorMessage(resp, "Failed to unlock identity", http.StatusBadRequest)
		return
	}

	instance, err := se.serviceManager.Start(identity.FromAddress(sr.ProviderID), sr.ServiceType, options)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	statusResponse := toServiceInfoResponse(instance)
	statusResponse.Status = string(service.Starting)
	utils.WriteAsJSON(statusResponse, resp)
}

// ServiceStop stops service on the node.
// swagger:operation DELETE /services/:id Service serviceStop
// ---
// summary: Stops service
// description: Initiates service stop
// responses:
//   202:
//     description: Service Stop initiated
//   404:
//     description: No service exists
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (se *ServiceEndpoint) ServiceStop(resp http.ResponseWriter, _ *http.Request, params httprouter.Params) {
	for _, instance := range se.serviceManager.List() {
		if instance.ID() == service.ID(params.ByName("id")) {
			if err := se.serviceManager.Stop(instance); err != nil {
				utils.SendError(resp, err, http.StatusInternalServerError)
				return
			}
			resp.WriteHeader(http.StatusAccepted)
			return
		}
	}
	utils.SendErrorMessage(resp, "Service not found", http.StatusNotFound)
}

// AddRoutesForService adds service routes to given router
func AddRoutesForService(router *httprouter.Router, serviceManager ServiceManager, identityManager identity.Manager, optionsParser map[string]func(json.RawMessage) (service.Options, error)) {
	serviceEndpoint := NewServiceEndpoint(serviceManager, identityManager, optionsParser)

	router.GET("/services", serviceEndpoint.ServiceList)
	router.POST("/services", serviceEndpoint.ServiceStart)
	router.GET("/services/:id", serviceEndpoint.ServiceGet)
	router.DELETE("/services/:id", serviceEndpoint.ServiceStop)
}

func toServiceRequest(req *http.Request) (*serviceRequest, error) {
	var serviceRequest = serviceRequest{}
	err := json.NewDecoder(req.Body).Decode(&serviceRequest)

	return &serviceRequest, err
}

func toServiceInfoResponse(instance service.Instance) serviceInfo {
	return serviceInfo{
		Status:   string(service.Running),
		Proposal: proposalToRes(instance.Proposal()),
		ID:       string(instance.ID()),
	}
}

func toServiceListResponse(instances []*service.Instance) (res serviceList) {
	for _, instance := range instances {
		res = append(res, serviceInfo{
			Status:   string(service.Running),
			Proposal: proposalToRes(instance.Proposal()),
			ID:       string(instance.ID()),
		})
	}
	return res
}

func validateServiceRequest(cr *serviceRequest) *validation.FieldErrorMap {
	errors := validation.NewErrorMap()
	if len(cr.ProviderID) == 0 {
		errors.ForField("providerId").AddError("required", "Field is required")
	}
	if cr.ServiceType == "" {
		errors.ForField("serviceType").AddError("required", "Field is required")
	}
	return errors
}

// ServiceManager represents service manager that will be used for manipulation node services.
type ServiceManager interface {
	Start(providerID identity.Identity, serviceType string, options service.Options) (service.Instance, error)
	Stop(instance *service.Instance) error
	Kill() error
	List() []*service.Instance
}
