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
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

type Data struct {
	Version    string
	DarwinAll  string
	LinuxArm   string
	LinuxX64   string
	WindowsX64 string
}

// Example Usage:
// go run hack/homebrew.go --version 0.6.0 > ../aws-homebrew-tap/bottle-configs/eks-node-viewer.json

func main() {
	var data Data
	flag.StringVar(&data.Version, "version", "", "version to generate a homebrew config for")
	flag.Parse()
	if data.Version == "" {
		log.Fatalf("version must be supplied")
	}

	bconfig, err := template.New("bottle-config").Parse(`{
    "name": "eks-node-viewer",
    "version": "{{.Version}}",
    "bin": "eks-node-viewer",
    "bottle": {
        "root_url": "https://github.com/awslabs/eks-node-viewer/releases/download/v{{.Version}}/eks-node-viewer",
        "sha256": {
            "sierra": "{{.DarwinAll}}",
            "linux": "{{.LinuxX64}}",
            "linux_arm": "{{.LinuxArm}}"
        }
    }
}
`)
	if err != nil {
		log.Fatalf("unable to parse template, %s", err)
	}

	// fetch and parse the checksums
	req, err := http.Get(fmt.Sprintf(`https://github.com/awslabs/eks-node-viewer/releases/download/v%s/eks-node-viewer_%s_sha256_checksums.txt`, data.Version, data.Version))
	if err != nil {
		log.Fatalf("fetching checksums, %s", err)
	}
	defer req.Body.Close()
	sc := bufio.NewScanner(req.Body)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			log.Fatalf("unavble to parse line, %q", sc.Text())
		}
		hash := fields[0]
		bin := fields[1]
		switch bin {
		case "eks-node-viewer_Darwin_all":
			data.DarwinAll = hash
		case "eks-node-viewer_Linux_arm64":
			data.LinuxArm = hash
		case "eks-node-viewer_Linux_x86_64":
			data.LinuxX64 = hash
		case "eks-node-viewer_Windows_x86_64.exe":
			data.WindowsX64 = hash
		default:
			log.Fatalf("unsupported bin, %s", bin)
		}
	}

	if err := bconfig.Execute(os.Stdout, data); err != nil {
		log.Fatalf("executing template, %s", err)
	}
}
