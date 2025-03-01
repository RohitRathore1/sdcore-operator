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
	AMFImage  = "registry.aetherproject.org/omecproject/5gc-amf:1.5.0"
	SMFImage  = "registry.aetherproject.org/omecproject/5gc-smf:1.5.0"
	UPFImage  = "registry.aetherproject.org/omecproject/upf-epc:1.5.0"
	NRFImage  = "registry.aetherproject.org/omecproject/5gc-nrf:1.5.0"
	AUSFImage = "registry.aetherproject.org/omecproject/5gc-ausf:1.5.0"
	UDMImage  = "registry.aetherproject.org/omecproject/5gc-udm:1.5.0"
	UDRImage  = "registry.aetherproject.org/omecproject/5gc-udr:1.5.0"
	PCFImage  = "registry.aetherproject.org/omecproject/5gc-pcf:1.5.0"
) 