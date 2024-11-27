//go:build !ignore_autogenerated

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

// generated at 2024-11-06T01:00:11Z for cn-north-1

import ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

var InitialOnDemandPricesCN = map[string]map[ec2types.InstanceType]float64{
	"cn-north-1": {
		// c3 family
		"c3.2xlarge": 4.217000, "c3.4xlarge": 8.434000, "c3.8xlarge": 16.869000, "c3.large": 1.054000,
		"c3.xlarge": 2.109000,
		// c4 family
		"c4.2xlarge": 4.535000, "c4.4xlarge": 9.071000, "c4.8xlarge": 18.141000, "c4.large": 1.134000,
		"c4.xlarge": 2.268000,
		// c5 family
		"c5.12xlarge": 17.745000, "c5.18xlarge": 26.617000, "c5.24xlarge": 35.490000, "c5.2xlarge": 2.957000,
		"c5.4xlarge": 5.915000, "c5.9xlarge": 13.309000, "c5.large": 0.739000, "c5.metal": 35.490000,
		"c5.xlarge": 1.479000,
		// c5a family
		"c5a.12xlarge": 15.937000, "c5a.16xlarge": 21.250000, "c5a.24xlarge": 31.875000, "c5a.2xlarge": 2.656000,
		"c5a.4xlarge": 5.312000, "c5a.8xlarge": 10.625000, "c5a.large": 0.664000, "c5a.xlarge": 1.328000,
		// c5d family
		"c5d.12xlarge": 21.852000, "c5d.18xlarge": 32.779000, "c5d.24xlarge": 43.705000, "c5d.2xlarge": 3.642000,
		"c5d.4xlarge": 7.284000, "c5d.9xlarge": 16.389000, "c5d.large": 0.911000, "c5d.metal": 43.705000,
		"c5d.xlarge": 1.821000,
		// c6g family
		"c6g.12xlarge": 14.064400, "c6g.16xlarge": 18.752600, "c6g.2xlarge": 2.344100, "c6g.4xlarge": 4.688100,
		"c6g.8xlarge": 9.376300, "c6g.large": 0.586000, "c6g.medium": 0.293000, "c6g.metal": 19.390900,
		"c6g.xlarge": 1.172000,
		// c6gn family
		"c6gn.12xlarge": 17.869700, "c6gn.16xlarge": 23.826270, "c6gn.2xlarge": 2.978280, "c6gn.4xlarge": 5.956570,
		"c6gn.8xlarge": 11.913140, "c6gn.large": 0.744570, "c6gn.medium": 0.372290, "c6gn.xlarge": 1.489140,
		// c6i family
		"c6i.12xlarge": 17.744830, "c6i.16xlarge": 23.659780, "c6i.24xlarge": 35.489660, "c6i.2xlarge": 2.957470,
		"c6i.32xlarge": 47.319550, "c6i.4xlarge": 5.914940, "c6i.8xlarge": 11.829890, "c6i.large": 0.739370,
		"c6i.metal": 47.319550, "c6i.xlarge": 1.478740,
		// c7g family
		"c7g.12xlarge": 15.083100, "c7g.16xlarge": 20.110800, "c7g.2xlarge": 2.513900, "c7g.4xlarge": 5.027700,
		"c7g.8xlarge": 10.055400, "c7g.large": 0.628500, "c7g.medium": 0.314200, "c7g.metal": 20.110800,
		"c7g.xlarge": 1.256900,
		// d2 family
		"d2.2xlarge": 13.345000, "d2.4xlarge": 26.690000, "d2.8xlarge": 53.380000, "d2.xlarge": 6.673000,
		// g3 family
		"g3.16xlarge": 64.817900, "g3.4xlarge": 16.204500, "g3.8xlarge": 32.409000,
		// g3s family
		"g3s.xlarge": 11.282000,
		// g4dn family
		"g4dn.12xlarge": 38.849000, "g4dn.16xlarge": 43.218000, "g4dn.2xlarge": 7.468000, "g4dn.4xlarge": 11.956000,
		"g4dn.8xlarge": 21.609000, "g4dn.xlarge": 5.223000,
		// g5 family
		"g5.12xlarge": 53.640920, "g5.16xlarge": 38.736460, "g5.24xlarge": 77.018980, "g5.2xlarge": 11.462060,
		"g5.48xlarge": 154.037950, "g5.4xlarge": 15.358400, "g5.8xlarge": 23.151090, "g5.xlarge": 9.513890,
		// i2 family
		"i2.2xlarge": 20.407000, "i2.4xlarge": 40.815000, "i2.8xlarge": 81.630000, "i2.xlarge": 10.204000,
		// i3 family
		"i3.16xlarge": 49.948000, "i3.2xlarge": 6.244000, "i3.4xlarge": 12.487000, "i3.8xlarge": 24.974000,
		"i3.large": 1.561000, "i3.xlarge": 3.122000,
		// i3en family
		"i3en.12xlarge": 54.302000, "i3en.24xlarge": 108.605000, "i3en.2xlarge": 9.050000, "i3en.3xlarge": 13.576000,
		"i3en.6xlarge": 27.151000, "i3en.large": 2.263000, "i3en.xlarge": 4.525000,
		// i4i family
		"i4i.12xlarge": 43.665000, "i4i.16xlarge": 58.221000, "i4i.24xlarge": 87.330860, "i4i.2xlarge": 7.278000,
		"i4i.32xlarge": 116.441150, "i4i.4xlarge": 14.555000, "i4i.8xlarge": 29.110000, "i4i.large": 1.819000,
		"i4i.xlarge": 3.639000,
		// inf1 family
		"inf1.24xlarge": 47.342000, "inf1.2xlarge": 3.630000, "inf1.6xlarge": 11.835000, "inf1.xlarge": 2.288000,
		// m1 family
		"m1.small": 0.442000,
		// m3 family
		"m3.2xlarge": 6.942000, "m3.large": 1.735000, "m3.medium": 0.868000, "m3.xlarge": 3.471000,
		// m4 family
		"m4.10xlarge": 28.121000, "m4.16xlarge": 44.995000, "m4.2xlarge": 5.624000, "m4.4xlarge": 11.248000,
		"m4.large": 1.405000, "m4.xlarge": 2.815000,
		// m5 family
		"m5.12xlarge": 24.317000, "m5.16xlarge": 32.423000, "m5.24xlarge": 48.634000, "m5.2xlarge": 4.053000,
		"m5.4xlarge": 8.106000, "m5.8xlarge": 16.211000, "m5.large": 1.013000, "m5.metal": 48.634000,
		"m5.xlarge": 2.026000,
		// m5a family
		"m5a.12xlarge": 21.852000, "m5a.16xlarge": 29.137000, "m5a.24xlarge": 43.705000, "m5a.2xlarge": 3.642000,
		"m5a.4xlarge": 7.284000, "m5a.8xlarge": 14.568000, "m5a.large": 0.911000, "m5a.xlarge": 1.821000,
		// m5d family
		"m5d.12xlarge": 30.561000, "m5d.16xlarge": 40.747000, "m5d.24xlarge": 61.121000, "m5d.2xlarge": 5.093000,
		"m5d.4xlarge": 10.187000, "m5d.8xlarge": 20.374000, "m5d.large": 1.273000, "m5d.metal": 61.121000,
		"m5d.xlarge": 2.547000,
		// m6g family
		"m6g.12xlarge": 19.289300, "m6g.16xlarge": 25.719100, "m6g.2xlarge": 3.214900, "m6g.4xlarge": 6.429800,
		"m6g.8xlarge": 12.859500, "m6g.large": 0.803700, "m6g.medium": 0.401900, "m6g.metal": 26.486300,
		"m6g.xlarge": 1.607400,
		// m6i family
		"m6i.12xlarge": 24.316990, "m6i.16xlarge": 32.422660, "m6i.24xlarge": 48.633980, "m6i.2xlarge": 4.052830,
		"m6i.32xlarge": 64.845310, "m6i.4xlarge": 8.105660, "m6i.8xlarge": 16.211330, "m6i.large": 1.013210,
		"m6i.metal": 64.845310, "m6i.xlarge": 2.026420,
		// m7g family
		"m7g.12xlarge": 20.669400, "m7g.16xlarge": 27.559300, "m7g.2xlarge": 3.444900, "m7g.4xlarge": 6.889800,
		"m7g.8xlarge": 13.779600, "m7g.large": 0.861200, "m7g.medium": 0.430600, "m7g.metal": 27.559300,
		"m7g.xlarge": 1.722500,
		// p2 family
		"p2.16xlarge": 169.792000, "p2.8xlarge": 84.896000, "p2.xlarge": 10.612000,
		// p3 family
		"p3.16xlarge": 288.627000, "p3.2xlarge": 36.078000, "p3.8xlarge": 144.314000,
		// r3 family
		"r3.2xlarge": 9.803600, "r3.4xlarge": 19.607300, "r3.8xlarge": 39.214700, "r3.large": 2.450900,
		"r3.xlarge": 4.901800,
		// r4 family
		"r4.16xlarge": 62.746000, "r4.2xlarge": 7.842000, "r4.4xlarge": 15.683000, "r4.8xlarge": 31.373000,
		"r4.large": 1.959000, "r4.xlarge": 3.924000,
		// r5 family
		"r5.12xlarge": 29.246000, "r5.16xlarge": 38.995000, "r5.24xlarge": 58.492000, "r5.2xlarge": 4.874000,
		"r5.4xlarge": 9.749000, "r5.8xlarge": 19.497000, "r5.large": 1.219000, "r5.metal": 58.492000,
		"r5.xlarge": 2.437000,
		// r5a family
		"r5a.12xlarge": 26.322000, "r5a.16xlarge": 35.095000, "r5a.24xlarge": 52.643000, "r5a.2xlarge": 4.387000,
		"r5a.4xlarge": 8.774000, "r5a.8xlarge": 17.548000, "r5a.large": 1.097000, "r5a.xlarge": 2.193000,
		// r5d family
		"r5d.12xlarge": 35.490000, "r5d.16xlarge": 47.320000, "r5d.24xlarge": 70.979000, "r5d.2xlarge": 5.915000,
		"r5d.4xlarge": 11.830000, "r5d.8xlarge": 23.660000, "r5d.large": 1.479000, "r5d.metal": 70.979000,
		"r5d.xlarge": 2.957000,
		// r6g family
		"r6g.12xlarge": 23.199700, "r6g.16xlarge": 30.933000, "r6g.2xlarge": 3.866600, "r6g.4xlarge": 7.733200,
		"r6g.8xlarge": 15.466500, "r6g.large": 0.966700, "r6g.medium": 0.483300, "r6g.metal": 31.847000,
		"r6g.xlarge": 1.933300,
		// r6gd family
		"r6gd.12xlarge": 28.096000, "r6gd.16xlarge": 37.461300, "r6gd.2xlarge": 4.682700, "r6gd.4xlarge": 9.365300,
		"r6gd.8xlarge": 18.730700, "r6gd.large": 1.170700, "r6gd.medium": 0.585300, "r6gd.metal": 37.461300,
		"r6gd.xlarge": 2.341300,
		// r6i family
		"r6i.12xlarge": 29.246110, "r6i.16xlarge": 38.994820, "r6i.24xlarge": 58.492220, "r6i.2xlarge": 4.874350,
		"r6i.32xlarge": 77.989630, "r6i.4xlarge": 9.748700, "r6i.8xlarge": 19.497410, "r6i.large": 1.218590,
		"r6i.metal": 77.989630, "r6i.xlarge": 2.437180,
		// r7g family
		"r7g.12xlarge": 24.875600, "r7g.16xlarge": 33.167500, "r7g.2xlarge": 4.145900, "r7g.4xlarge": 8.291900,
		"r7g.8xlarge": 16.583800, "r7g.large": 1.036500, "r7g.medium": 0.518200, "r7g.metal": 33.167500,
		"r7g.xlarge": 2.073000,
		// t1 family
		"t1.micro": 0.221000,
		// t2 family
		"t2.2xlarge": 3.392000, "t2.large": 0.851000, "t2.medium": 0.426000, "t2.micro": 0.106000,
		"t2.nano": 0.060600, "t2.small": 0.212000, "t2.xlarge": 1.696000,
		// t3 family
		"t3.2xlarge": 2.103100, "t3.large": 0.525800, "t3.medium": 0.262900, "t3.micro": 0.065700,
		"t3.nano": 0.032900, "t3.small": 0.131400, "t3.xlarge": 1.051500,
		// t3a family
		"t3a.2xlarge": 1.892800, "t3a.large": 0.473200, "t3a.medium": 0.236600, "t3a.micro": 0.059100,
		"t3a.nano": 0.029600, "t3a.small": 0.118300, "t3a.xlarge": 0.946400,
		// t4g family
		"t4g.2xlarge": 1.621100, "t4g.large": 0.405300, "t4g.medium": 0.202600, "t4g.micro": 0.050700,
		"t4g.nano": 0.025300, "t4g.small": 0.101300, "t4g.xlarge": 0.810600,
		// u-12tb1 family
		"u-12tb1.112xlarge": 1074.685080,
		// u-6tb1 family
		"u-6tb1.112xlarge": 537.342540, "u-6tb1.56xlarge": 456.681190,
		// u-9tb1 family
		"u-9tb1.112xlarge": 805.979580,
		// x1 family
		"x1.16xlarge": 68.876000, "x1.32xlarge": 137.752000,
		// x2idn family
		"x2idn.16xlarge": 68.870760, "x2idn.24xlarge": 103.306140, "x2idn.32xlarge": 137.741520,
		"x2idn.metal": 137.741520,
		// x2iedn family
		"x2iedn.16xlarge": 137.741520, "x2iedn.24xlarge": 206.612280, "x2iedn.2xlarge": 17.217690,
		"x2iedn.32xlarge": 275.483040, "x2iedn.4xlarge": 34.435380, "x2iedn.8xlarge": 68.870760,
		"x2iedn.metal": 275.483040, "x2iedn.xlarge": 8.608850,
	},
}
