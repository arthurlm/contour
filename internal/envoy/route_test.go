// Copyright © 2018 Heptio
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"testing"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	"github.com/gogo/protobuf/types"
	"github.com/google/go-cmp/cmp"
	"github.com/heptio/contour/internal/dag"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestRouteRoute(t *testing.T) {
	s1 := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kuard",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Name:       "http",
				Protocol:   "TCP",
				Port:       8080,
				TargetPort: intstr.FromInt(8080),
			}},
		},
	}
	tests := map[string]struct {
		route    *dag.Route
		services []*dag.Service
		want     route.Route_Route
	}{
		"simple": {
			route: &dag.Route{
				Prefix: "/",
			},
			services: []*dag.Service{{
				Object:      s1,
				ServicePort: &s1.Spec.Ports[0],
			}},
			want: route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					RequestHeadersToAdd: headers(
						appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%"),
					),
				},
			},
		},
		"websocket": {
			route: &dag.Route{
				Prefix:    "/",
				Websocket: true,
			},
			services: []*dag.Service{{
				Object:      s1,
				ServicePort: &s1.Spec.Ports[0],
			}},
			want: route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
						Cluster: "default/kuard/8080/da39a3ee5e",
					},
					RequestHeadersToAdd: headers(
						appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%"),
					),
					UseWebsocket: &types.BoolValue{Value: true},
				},
			},
		},
		"multiple": {
			route: &dag.Route{
				Prefix: "/",
			},
			services: []*dag.Service{{
				Object:      s1,
				ServicePort: &s1.Spec.Ports[0],
				Weight:      90,
			}, {
				Object:      s1, // it's valid to mention the same service several times per route.
				ServicePort: &s1.Spec.Ports[0],
			}},
			want: route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: &route.WeightedCluster{
							Clusters: []*route.WeightedCluster_ClusterWeight{{
								Name:                "default/kuard/8080/da39a3ee5e",
								Weight:              u32(0),
								RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
							}, {
								Name:                "default/kuard/8080/da39a3ee5e",
								Weight:              u32(90),
								RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
							}},
							TotalWeight: u32(90),
						},
					},
				},
			},
		},
		"multiple websocket": {
			route: &dag.Route{
				Prefix:    "/",
				Websocket: true,
			},
			services: []*dag.Service{{
				Object:      s1,
				ServicePort: &s1.Spec.Ports[0],
				Weight:      90,
			}, {
				Object:      s1, // it's valid to mention the same service several times per route.
				ServicePort: &s1.Spec.Ports[0],
			}},
			want: route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_WeightedClusters{
						WeightedClusters: &route.WeightedCluster{
							Clusters: []*route.WeightedCluster_ClusterWeight{{
								Name:                "default/kuard/8080/da39a3ee5e",
								Weight:              u32(0),
								RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
							}, {
								Name:                "default/kuard/8080/da39a3ee5e",
								Weight:              u32(90),
								RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
							}},
							TotalWeight: u32(90),
						},
					},
					UseWebsocket: &types.BoolValue{Value: true},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := RouteRoute(tc.route, tc.services)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestWeightedClusters(t *testing.T) {
	tests := map[string]struct {
		services []*dag.Service
		want     *route.WeightedCluster
	}{
		"multiple services w/o weights": {
			services: []*dag.Service{{
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
			}, {
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
			}},
			want: &route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{{
					Name:                "default/kuard/8080/da39a3ee5e",
					Weight:              u32(1),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}, {
					Name:                "default/nginx/8080/da39a3ee5e",
					Weight:              u32(1),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}},
				TotalWeight: u32(2),
			},
		},
		"multiple weighted services": {
			services: []*dag.Service{{
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
				Weight: 80,
			}, {
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
				Weight: 20,
			}},
			want: &route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{{
					Name:                "default/kuard/8080/da39a3ee5e",
					Weight:              u32(80),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}, {
					Name:                "default/nginx/8080/da39a3ee5e",
					Weight:              u32(20),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}},
				TotalWeight: u32(100),
			},
		},
		"multiple weighted services and one with no weight specified": {
			services: []*dag.Service{{
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kuard",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
				Weight: 80,
			}, {
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "nginx",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
				Weight: 20,
			}, {
				Object: &v1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "notraffic",
						Namespace: "default",
					},
				},
				ServicePort: &v1.ServicePort{
					Port: 8080,
				},
			}},
			want: &route.WeightedCluster{
				Clusters: []*route.WeightedCluster_ClusterWeight{{
					Name:                "default/kuard/8080/da39a3ee5e",
					Weight:              u32(80),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}, {
					Name:                "default/nginx/8080/da39a3ee5e",
					Weight:              u32(20),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}, {
					Name:                "default/notraffic/8080/da39a3ee5e",
					Weight:              u32(0),
					RequestHeadersToAdd: headers(appendHeader("x-request-start", "t=%START_TIME(%s.%3f)%")),
				}},
				TotalWeight: u32(100),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := weightedClusters(tc.services)
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestVirtualHost(t *testing.T) {
	tests := map[string]struct {
		hostname string
		port     int
		want     route.VirtualHost
	}{
		"default hostname": {
			hostname: "*",
			port:     9999,
			want: route.VirtualHost{
				Name:    "*",
				Domains: []string{"*"},
			},
		},
		"www.example.com": {
			hostname: "www.example.com",
			port:     9999,
			want: route.VirtualHost{
				Name:    "www.example.com",
				Domains: []string{"www.example.com", "www.example.com:9999"},
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := VirtualHost(tc.hostname, tc.port)
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestPrefixMatch(t *testing.T) {
	const prefix = "/kang"
	got := PrefixMatch(prefix)
	want := route.RouteMatch{
		PathSpecifier: &route.RouteMatch_Prefix{
			Prefix: prefix,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestUpgradeHTTPS(t *testing.T) {
	got := UpgradeHTTPS()
	want := &route.Route_Redirect{
		Redirect: &route.RedirectAction{
			HttpsRedirect: true,
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatal(diff)
	}
}
