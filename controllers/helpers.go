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

import (
	"encoding/json"
	"errors"
	
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

// Parameter represents a key-value parameter
type Parameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetParameterValue extracts a parameter value from NFDeployment using json unmarshal
func GetParameterValue(nf *nephiov1alpha1.NFDeployment, paramName string) (string, error) {
	// Convert the NFDeployment to raw json
	rawJson, err := json.Marshal(nf)
	if err != nil {
		return "", err
	}

	// Parse the json into a map
	var data map[string]interface{}
	if err := json.Unmarshal(rawJson, &data); err != nil {
		return "", err
	}

	// Navigate to spec.parameterValues
	spec, ok := data["spec"].(map[string]interface{})
	if !ok {
		return "", errors.New("spec not found or invalid format")
	}

	paramValues, ok := spec["parameterValues"].([]interface{})
	if !ok {
		return "", errors.New("parameterValues not found or invalid format")
	}

	// Find the parameter by name
	for _, param := range paramValues {
		paramMap, ok := param.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := paramMap["name"].(string)
		if !ok {
			continue
		}

		if name == paramName {
			value, ok := paramMap["value"].(string)
			if !ok {
				return "", errors.New("parameter value is not a string")
			}
			return value, nil
		}
	}

	return "", errors.New("parameter not found")
}

// GetCapacitySize returns the capacity size from the parameterValues
func GetCapacitySize(nf *nephiov1alpha1.NFDeployment) (string, error) {
	return GetParameterValue(nf, "capacity")
}

// GetDNSIP returns the DNS IP from the parameterValues
func GetDNSIP(nf *nephiov1alpha1.NFDeployment) (string, error) {
	return GetParameterValue(nf, "dns")
} 