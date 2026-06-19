/*
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
package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestPricingAPIRegion(t *testing.T) {
	cases := map[string]string{
		"us-east-1":      "us-east-1",
		"us-west-2":      "us-east-1",
		"ap-south-1":     "ap-south-1",
		"ap-northeast-1": "ap-south-1",
		"cn-north-1":     "cn-northwest-1",
		"eu-west-1":      "eu-central-1",
		"":               "us-east-1",
	}
	for in, want := range cases {
		if got := pricingAPIRegion(in); got != want {
			t.Errorf("pricingAPIRegion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewPricingClientPreservesConfig(t *testing.T) {
	cfg := aws.Config{Region: "eu-west-1"}

	if client := NewPricingClient(cfg); client == nil {
		t.Fatal("NewPricingClient returned nil")
	}

	// The pricing client must not mutate the caller's config: it should copy
	// before overriding the region, leaving the original credentials/region in
	// place for the EC2 client.
	if cfg.Region != "eu-west-1" {
		t.Errorf("NewPricingClient mutated the input config region to %q", cfg.Region)
	}
}
