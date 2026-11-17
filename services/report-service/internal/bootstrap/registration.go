package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/config"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/discovery/consul"
	"github.com/vleshchenko/land-plot-system/services/report-service/internal/logging"
)

type registration struct {
	registry      *consul.Registry
	serviceConfig config.Config
	logger        *slog.Logger
}

func register(serviceConfig config.Config, logger *slog.Logger) (*registration, error) {
	consulContext, cancel := context.WithTimeout(context.Background(), serviceConfig.ConsulTimeout)
	defer cancel()

	address, err := consul.AdvertiseAddress(consulContext, serviceConfig.ServiceAddress)
	if err != nil {
		return nil, fmt.Errorf(wrapDetermineServiceAddress, err)
	}

	registry := consul.NewRegistry(serviceConfig.ConsulAddress, &http.Client{Timeout: serviceConfig.ConsulTimeout})

	err = registry.Register(consulContext, consul.Registration{
		ID:                serviceConfig.InstanceID,
		Name:              config.ServiceName,
		Address:           address,
		Port:              serviceConfig.HTTPPort,
		HealthCheckTarget: consul.HealthURL(address, serviceConfig.HTTPPort),
	})
	if err != nil {
		return nil, fmt.Errorf(wrapRegisterInConsul, err)
	}

	logger.Info(
		logServiceRegistered,
		logging.KeyInstanceID, serviceConfig.InstanceID,
		logging.KeyAddress, address,
		logging.KeyPort, serviceConfig.HTTPPort,
	)

	return &registration{registry: registry, serviceConfig: serviceConfig, logger: logger}, nil
}

func (consulRegistration *registration) deregister() {
	consulContext, cancel := context.WithTimeout(context.Background(), consulRegistration.serviceConfig.ConsulTimeout)
	defer cancel()

	if err := consulRegistration.registry.Deregister(
		consulContext,
		consulRegistration.serviceConfig.InstanceID,
	); err != nil {
		consulRegistration.logger.Error(
			logDeregisterService,
			logging.KeyInstanceID, consulRegistration.serviceConfig.InstanceID,
			logging.KeyError, err,
		)

		return
	}

	consulRegistration.logger.Info(
		logServiceDeregistered,
		logging.KeyInstanceID, consulRegistration.serviceConfig.InstanceID,
	)
}
