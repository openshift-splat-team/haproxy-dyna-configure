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

package parsers

import (
	"github.com/haproxytech/config-parser/common"
	"github.com/haproxytech/config-parser/errors"
	"github.com/haproxytech/config-parser/types"
)

type Daemon struct {
	data *types.Enabled
}

func (d *Daemon) Parse(line string, parts, previousParts []string, comment string) (changeState string, err error) {
	if parts[0] == "daemon" {
		d.data = &types.Enabled{
			Comment: comment,
		}
		return "", nil
	}
	return "", &errors.ParseError{Parser: "Daemon", Line: line}
}

func (d *Daemon) Result() ([]common.ReturnResultLine, error) {
	if d.data == nil {
		return nil, errors.ErrFetch
	}
	return []common.ReturnResultLine{
		common.ReturnResultLine{
			Data:    "daemon",
			Comment: d.data.Comment,
		},
	}, nil
}
