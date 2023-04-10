/*
Copyright 2019 HAProxy Technologies

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

package actions

import (
	"fmt"
	"strings"
)

type InspectDelay struct {
	Timeout string
	Comment string
}

func (f *InspectDelay) Parse(parts []string, comment string) error {
	if comment != "" {
		f.Comment = comment
	}
	if len(parts) >= 3 {
		if len(parts) > 1 {
			f.Timeout = parts[2]
		} else {
			return fmt.Errorf("not enough params")
		}
		return nil
	}
	return fmt.Errorf("not enough params")
}

func (f *InspectDelay) String() string {
	var result strings.Builder
	result.WriteString("inspect-delay ")
	result.WriteString(f.Timeout)

	return result.String()
}

func (f *InspectDelay) GetComment() string {
	return f.Comment
}
