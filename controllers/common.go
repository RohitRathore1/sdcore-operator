/*
Copyright 2024 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

const (
	// Annotation used to store the ConfigMap version in the pods template
	ConfigMapVersionAnnotation = "sdcore.nephio.org/configmap-version"

	// Annotation used to store the network attachment definitions
	NetworksAnnotation = "k8s.v1.cni.cncf.io/networks"

	// Container images for each SDCore component
	// InitImage is the container image for init container
	InitImage = "omecproject/pod-init:rel-1.1.2"

	AMFImage  = "omecproject/5gc-amf:rel-1.6.4"
	SMFImage  = "omecproject/5gc-smf:rel-2.0.3"
	UPFImage  = "omecproject/5gc-upf:rel-1.0.0"
	NRFImage  = "omecproject/5gc-nrf:rel-1.6.3"
	AUSFImage = "omecproject/5gc-ausf:rel-1.6.2"
	UDMImage  = "omecproject/5gc-udm:rel-1.6.2"
	UDRImage  = "omecproject/5gc-udr:rel-1.6.3"
	PCFImage  = "omecproject/5gc-pcf:rel-1.6.2"

	// NSSFImage is the container image for NSSF
	NSSFImage = "omecproject/5gc-nssf:rel-1.6.2"

	// WEBUIImage is the container image for WebUI
	WEBUIImage = "omecproject/5gc-webui:rel-1.8.3"

	// SCTPLBImage is the container image for SCTP Load Balancer
	SCTPLBImage = "omecproject/sctplb:rel-1.6.1"

	// MetricFuncImage is the container image for Metric Function
	MetricFuncImage = "omecproject/metricfunc:rel-1.6.1"

	// UPFAdapterImage is the container image for UPF Adapter
	UPFAdapterImage = "omecproject/upfadapter:rel-2.0.2"
)
