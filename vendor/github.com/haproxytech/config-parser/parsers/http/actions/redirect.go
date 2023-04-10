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

	"github.com/haproxytech/config-parser/common"
	"github.com/haproxytech/config-parser/errors"
)

type Redirect struct { // http-request redirect location <loc> [code <code>] [<option>] [<condition>]
	Type     string
	Value    string
	Code     string
	Option   string
	Cond     string
	CondTest string
	Comment  string
}

func (f *Redirect) Parse(parts []string, comment string) error {
	if comment != "" {
		f.Comment = comment
	}
	/*
	  redirect location <loc> [code <code>] <option> [{if | unless} <condition>]
	  redirect prefix   <pfx> [code <code>] <option> [{if | unless} <condition>]
	  redirect scheme   <sch> [code <code>] <option> [{if | unless} <condition>]
	*/
	if len(parts) >= 4 {
		command, condition := common.SplitRequest(parts[2:])
		if len(command) < 2 {
			return errors.ErrInvalidData
		}
		f.Type = command[0]
		f.Value = command[1]
		index := 2
		if index < len(command) && command[index] == "code" {
			index++
			if index == len(command) {
				return fmt.Errorf("not enough params")
			}
			f.Code = command[index]
			index++
		}
		if index < len(command) {
			f.Option = command[index]
		}
		if len(condition) > 1 {
			f.Cond = condition[0]
			f.CondTest = strings.Join(condition[1:], " ")
		}
		return nil
	}
	return fmt.Errorf("not enough params")
}

func (f *Redirect) String() string {
	var result strings.Builder
	result.WriteString("redirect ")
	result.WriteString(f.Type)
	result.WriteString(" ")
	result.WriteString(f.Value)
	if f.Code != "" {
		result.WriteString(" code ")
		result.WriteString(f.Code)
	}
	if f.Option != "" {
		result.WriteString(" ")
		result.WriteString(f.Option)
	}
	if f.Cond != "" {
		result.WriteString(" ")
		result.WriteString(f.Cond)
		result.WriteString(" ")
		result.WriteString(f.CondTest)
	}
	return result.String()
}

func (f *Redirect) GetComment() string {
	return f.Comment
}
