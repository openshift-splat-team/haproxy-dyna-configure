// Copyright 2019 HAProxy Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package configuration

import (
	"strconv"

	strfmt "github.com/go-openapi/strfmt"
	parser "github.com/haproxytech/config-parser"
	parser_errors "github.com/haproxytech/config-parser/errors"
	"github.com/haproxytech/config-parser/types"
	"github.com/haproxytech/models"
)

// GetLogTargets returns configuration version and an array of
// configured log targets in the specified parent. Returns error on fail.
func (c *Client) GetLogTargets(parentType, parentName string, transactionID string) (int64, models.LogTargets, error) {
	p, err := c.GetParser(transactionID)
	if err != nil {
		return 0, nil, err
	}

	v, err := c.GetVersion(transactionID)
	if err != nil {
		return 0, nil, err
	}

	logTargets, err := c.parseLogTargets(parentType, parentName, p)
	if err != nil {
		return v, nil, c.handleError("", parentType, parentName, "", false, err)
	}

	return v, logTargets, nil
}

// GetLogTarget returns configuration version and a requested log target
// in the specified parent. Returns error on fail or if log target does not exist.
func (c *Client) GetLogTarget(id int64, parentType, parentName string, transactionID string) (int64, *models.LogTarget, error) {
	p, err := c.GetParser(transactionID)
	if err != nil {
		return 0, nil, err
	}

	v, err := c.GetVersion(transactionID)
	if err != nil {
		return 0, nil, err
	}

	var section parser.Section
	if parentType == "backend" {
		section = parser.Backends
	} else if parentType == "frontend" {
		section = parser.Frontends
	}

	data, err := p.GetOne(section, parentName, "log", int(id))
	if err != nil {
		return v, nil, c.handleError(strconv.FormatInt(id, 10), parentType, parentName, "", false, err)
	}

	logTarget := parseLogTarget(data.(types.Log))
	logTarget.ID = &id

	return v, logTarget, nil
}

// DeleteLogTarget deletes a log target in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) DeleteLogTarget(id int64, parentType string, parentName string, transactionID string, version int64) error {
	p, t, err := c.loadDataForChange(transactionID, version)
	if err != nil {
		return err
	}

	var section parser.Section
	if parentType == "backend" {
		section = parser.Backends
	} else if parentType == "frontend" {
		section = parser.Frontends
	}

	if err := p.Delete(section, parentName, "log", int(id)); err != nil {
		return c.handleError(strconv.FormatInt(id, 10), parentType, parentName, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}

	return nil
}

// CreateLogTarget creates a log target in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) CreateLogTarget(parentType string, parentName string, data *models.LogTarget, transactionID string, version int64) error {
	if c.UseValidation {
		validationErr := data.Validate(strfmt.Default)
		if validationErr != nil {
			return NewConfError(ErrValidationError, validationErr.Error())
		}
	}

	p, t, err := c.loadDataForChange(transactionID, version)
	if err != nil {
		return err
	}

	var section parser.Section
	if parentType == "backend" {
		section = parser.Backends
	} else if parentType == "frontend" {
		section = parser.Frontends
	}

	if err := p.Insert(section, parentName, "log", serializeLogTarget(*data), int(*data.ID)); err != nil {
		return c.handleError(strconv.FormatInt(*data.ID, 10), parentType, parentName, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}
	return nil
}

// EditLogTarget edits a log target in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) EditLogTarget(id int64, parentType string, parentName string, data *models.LogTarget, transactionID string, version int64) error {
	if c.UseValidation {
		validationErr := data.Validate(strfmt.Default)
		if validationErr != nil {
			return NewConfError(ErrValidationError, validationErr.Error())
		}
	}
	p, t, err := c.loadDataForChange(transactionID, version)
	if err != nil {
		return err
	}

	var section parser.Section
	if parentType == "backend" {
		section = parser.Backends
	} else if parentType == "frontend" {
		section = parser.Frontends
	}

	if _, err := p.GetOne(section, parentName, "log", int(id)); err != nil {
		return c.handleError(strconv.FormatInt(id, 10), parentType, parentName, t, transactionID == "", err)
	}

	if err := p.Set(section, parentName, "log", serializeLogTarget(*data), int(id)); err != nil {
		return c.handleError(strconv.FormatInt(id, 10), parentType, parentName, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}
	return nil
}

func (c *Client) parseLogTargets(t, pName string, p *parser.Parser) (models.LogTargets, error) {
	var section parser.Section
	if t == "backend" {
		section = parser.Backends
	} else if t == "frontend" {
		section = parser.Frontends
	}

	logTargets := models.LogTargets{}
	data, err := p.Get(section, pName, "log", false)
	if err != nil {
		if err == parser_errors.ErrFetch {
			return logTargets, nil
		}
		return nil, err
	}

	targets := data.([]types.Log)
	for i, l := range targets {
		id := int64(i)
		logTarget := parseLogTarget(l)
		if logTarget != nil {
			logTarget.ID = &id
			logTargets = append(logTargets, logTarget)
		}
	}
	return logTargets, nil
}

func parseLogTarget(l types.Log) *models.LogTarget {
	return &models.LogTarget{
		Address:  l.Address,
		Facility: l.Facility,
		Format:   l.Format,
		Global:   l.Global,
		Length:   l.Length,
		Level:    l.Level,
		Minlevel: l.MinLevel,
		Nolog:    l.NoLog,
	}
}

func serializeLogTarget(l models.LogTarget) types.Log {
	return types.Log{
		Address:  l.Address,
		Facility: l.Facility,
		Format:   l.Format,
		Global:   l.Global,
		Length:   l.Length,
		Level:    l.Level,
		MinLevel: l.Minlevel,
		NoLog:    l.Nolog,
	}
}
