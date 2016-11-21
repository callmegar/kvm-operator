package controller

import (
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
)

var (
	reconcilliationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconcilliation_total",
			Help: "Number of times we have performed reconcilliation for a namespace",
		},
		[]string{"namespace"},
	)
	reconicilliationTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reconcilliation_milliseconds",
			Help: "Time taken to handle resource reconcilliation for a namespace, in milliseconds",
		},
		[]string{"namespace"},
	)

	reconcilliationResourceModificationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconcilliation_resource_modification_total",
			Help: "Number of times a resource has been modified during reconcilliation",
		},
		[]string{"namespace", "kind", "action"},
	)
	reconcilliationResourceModificationTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "reconcilliation_resource_modification_time",
			Help: "Time taken to handle resource modification for reconcilliation, in milliseconds",
		},
		[]string{"namespace", "kind", "action"},
	)
)

func init() {
	prometheus.MustRegister(reconcilliationTotal)
	prometheus.MustRegister(reconicilliationTime)
	prometheus.MustRegister(reconcilliationResourceModificationTotal)
	prometheus.MustRegister(reconcilliationResourceModificationTime)
}

// reconcileResourceState takes a list of Kubernetes resources, and makes sure
// that these resources exist.
func (c *controller) reconcileResourceState(namespaceName string, resources []runtime.Object) error {
	start := time.Now()
	reconcilliationTotal.WithLabelValues(namespaceName).Inc()

	log.Println("starting reconcilliation for namespace:", namespaceName)

	for _, resource := range resources {
		switch r := resource.(type) {
		case *v1.Service:
			start := time.Now()
			reconcilliationResourceModificationTotal.WithLabelValues(namespaceName, "service", "created").Inc()

			log.Println("creating service:", r.Name)

			if _, err := c.clientset.Core().Services(namespaceName).Create(r); err != nil && !errors.IsAlreadyExists(err) {
				return err
			}

			reconcilliationResourceModificationTime.WithLabelValues(namespaceName, "service", "created").Set(float64(time.Since(start) / time.Millisecond))
		}
	}

	log.Println("finished reconcilliation for namespace:", namespaceName)

	reconicilliationTime.WithLabelValues(namespaceName).Set(float64(time.Since(start) / time.Millisecond))

	return nil
}
