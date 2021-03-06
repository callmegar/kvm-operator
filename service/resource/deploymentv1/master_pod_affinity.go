package deploymentv1

import (
	"github.com/giantswarm/kvmtpr"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"

	"github.com/giantswarm/kvm-operator/service/keyv1"
)

func newMasterPodAfinity(customObject kvmtpr.CustomObject) *apiv1.Affinity {
	podAffinity := &apiv1.Affinity{
		PodAntiAffinity: &apiv1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []apiv1.PodAffinityTerm{
				{
					LabelSelector: &apismetav1.LabelSelector{
						MatchExpressions: []apismetav1.LabelSelectorRequirement{
							{
								Key:      "app",
								Operator: apismetav1.LabelSelectorOpIn,
								Values: []string{
									"master",
									"worker",
								},
							},
						},
					},
					TopologyKey: "kubernetes.io/hostname",
					Namespaces: []string{
						keyv1.ClusterID(customObject),
					},
				},
			},
		},
	}

	return podAffinity
}
