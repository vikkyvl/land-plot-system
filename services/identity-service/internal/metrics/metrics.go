package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Service struct {
	registry        *prometheus.Registry
	requestDuration *prometheus.HistogramVec
	profiles        prometheus.Counter
	newUsers        prometheus.Counter
}

func New(serviceName string) *Service {
	serviceLabel := prometheus.Labels{labelService: serviceName}

	service := &Service{
		registry:        prometheus.NewRegistry(),
		requestDuration: newRequestDuration(serviceLabel),
		profiles: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "identity_profiles_served_total",
			Help:        "Profiles returned for a verified token; the web app asks for it after every sign-in.",
			ConstLabels: serviceLabel,
		}),
		newUsers: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        "identity_users_created_total",
			Help:        "Users created just in time on their first verified token.",
			ConstLabels: serviceLabel,
		}),
	}

	service.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		service.requestDuration,
		service.profiles,
		service.newUsers,
	)

	return service
}

func (service *Service) Handler() http.Handler {
	return promhttp.HandlerFor(service.registry, promhttp.HandlerOpts{Registry: service.registry})
}

func (service *Service) ObserveRequests(next http.Handler) http.Handler {
	return observeRequests(service.requestDuration, next)
}

func (service *Service) ProfileServed(newUser bool) {
	service.profiles.Inc()

	if newUser {
		service.newUsers.Inc()
	}
}
