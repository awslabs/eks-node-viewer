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

package pricing

import "github.com/awslabs/eks-node-viewer/pkg/model"

// Provider provides node prices for display in the node viewer
type Provider interface {
	NodePrice(n *model.Node) (float64, bool)
	OnUpdate(onUpdate func())
}
