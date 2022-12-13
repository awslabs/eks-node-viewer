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
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const apacheLicense = `/*
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
`

func main() {
	for _, path := range os.Args[1:] {
		if err := filepath.WalkDir(path, addLicense); err != nil {
			log.Printf("processing %s, %s", path, err)
		}
	}
}

func addLicense(path string, d fs.DirEntry, err error) error {
	if !strings.HasSuffix(path, ".go") {
		return nil
	}

	srcFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s, %w", srcFile.Name(), err)
	}
	defer srcFile.Close()
	buf := make([]byte, len(apacheLicense)+50)
	_, err = srcFile.Read(buf)
	if err != nil {
		return fmt.Errorf("reading  %s, %w", srcFile.Name(), err)
	}
	if bytes.Index(buf, []byte(`http://www.apache.org/licenses/LICENSE-2.0`)) != -1 {
		return nil
	}
	log.Println("adding license to", path)

	tmp, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("creating temp file, %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	// write the license to the file
	fmt.Fprint(tmp, apacheLicense)
	if _, err := srcFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking, %w", err)
	}

	// followed by the source file contents
	_, err = io.Copy(tmp, srcFile)
	if err != nil {
		return fmt.Errorf("creating source, %w", err)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("removing %s, %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("moving %s => %s, %w", tmp.Name(), path, err)
	}
	return nil
}
