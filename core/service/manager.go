/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package service

import (
	"encoding/json"
	"errors"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	discovery_registry "github.com/mysteriumnetwork/node/market/proposals/registry"
	"github.com/mysteriumnetwork/node/session"
)

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
	// ErrUnsupportedServiceType indicates that manager tried to create an unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type")
)

// Service interface represents pluggable Mysterium service
type Service interface {
	Serve(providerID identity.Identity) error
	Stop() error
	ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error)
}

// DialogWaiterFactory initiates communication channel which waits for incoming dialogs
type DialogWaiterFactory func(providerID identity.Identity, serviceType string) (communication.DialogWaiter, error)

// DialogHandlerFactory initiates instance which is able to handle incoming dialogs
type DialogHandlerFactory func(market.ServiceProposal, session.ConfigNegotiator) communication.DialogHandler

// DiscoveryFactory initiates instance which is able announce service discoverability
type DiscoveryFactory func() *discovery_registry.Discovery

// NewManager creates new instance of pluggable instances manager
func NewManager(
	serviceRegistry *Registry,
	dialogWaiterFactory DialogWaiterFactory,
	dialogHandlerFactory DialogHandlerFactory,
	discoveryFactory DiscoveryFactory,
) *Manager {
	return &Manager{
		serviceRegistry:      serviceRegistry,
		servicePool:          NewPool(),
		dialogWaiterFactory:  dialogWaiterFactory,
		dialogHandlerFactory: dialogHandlerFactory,
		discoveryFactory:     discoveryFactory,
	}
}

// Manager entrypoint which knows how to start pluggable Mysterium instances
type Manager struct {
	dialogWaiterFactory  DialogWaiterFactory
	dialogHandlerFactory DialogHandlerFactory

	serviceRegistry *Registry
	servicePool     *Pool

	discoveryFactory DiscoveryFactory
}

// Start starts an instance of the given service type if knows one in service registry.
// It passes the options to the start method of the service.
// If an error occurs in the underlying service, the error is then returned.
func (manager *Manager) Start(providerID identity.Identity, serviceType string, options Options) (instance Instance, err error) {
	service, proposal, err := manager.serviceRegistry.Create(serviceType, options)
	if err != nil {
		return Instance{}, err
	}

	dialogWaiter, err := manager.dialogWaiterFactory(providerID, proposal.ServiceType)
	if err != nil {
		return Instance{}, err
	}
	providerContact, err := dialogWaiter.Start()
	if err != nil {
		return Instance{}, err
	}
	proposal.SetProviderContact(providerID, providerContact)

	dialogHandler := manager.dialogHandlerFactory(proposal, service)
	if err = dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return Instance{}, err
	}

	discovery := manager.discoveryFactory()
	discovery.Start(providerID, proposal)

	id, err := generateID()
	if err != nil {
		return Instance{}, err
	}

	instance = Instance{
		id:           id,
		service:      service,
		proposal:     proposal,
		dialogWaiter: dialogWaiter,
		discovery:    discovery,
	}

	manager.servicePool.Add(&instance)

	go func() {
		err = service.Serve(providerID)
		if err != nil {
			log.Error("Service serve failed: ", err)
		}

		discovery.Wait()
	}()
	return instance, nil
}

func (manager *Manager) List() []*Instance {
	return manager.servicePool.List()
}

// Kill stops all services
func (manager *Manager) Kill() error {
	return manager.servicePool.StopAll()
}

// Stop stops the service
func (manager *Manager) Stop(instance *Instance) error {
	return manager.servicePool.Stop(instance)
}
