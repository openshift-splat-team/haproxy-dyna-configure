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

// GetStickRules returns configuration version and an array of
// configured stick rules in the specified backend. Returns error on fail.
func (c *Client) GetStickRules(backend string, transactionID string) (int64, models.StickRules, error) {
	p, err := c.GetParser(transactionID)
	if err != nil {
		return 0, nil, err
	}

	v, err := c.GetVersion(transactionID)
	if err != nil {
		return 0, nil, err
	}

	sRules, err := c.parseStickRules(backend, p)
	if err != nil {
		return v, nil, c.handleError("", "backend", backend, "", false, err)
	}

	return v, sRules, nil
}

// GetStickRule returns configuration version and a requested stick rule
// in the specified backend. Returns error on fail or if stick rule does not exist.
func (c *Client) GetStickRule(id int64, backend string, transactionID string) (int64, *models.StickRule, error) {
	p, err := c.GetParser(transactionID)
	if err != nil {
		return 0, nil, err
	}

	v, err := c.GetVersion(transactionID)
	if err != nil {
		return 0, nil, err
	}

	data, err := p.GetOne(parser.Backends, backend, "stick", int(id))
	if err != nil {
		return v, nil, c.handleError(strconv.FormatInt(id, 10), "backend", backend, "", false, err)
	}

	sRule := parseStickRule(data.(types.Stick))
	sRule.ID = &id

	return v, sRule, nil
}

// DeleteStickRule deletes a stick rule in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) DeleteStickRule(id int64, backend string, transactionID string, version int64) error {
	p, t, err := c.loadDataForChange(transactionID, version)
	if err != nil {
		return err
	}

	if err := p.Delete(parser.Backends, backend, "stick", int(id)); err != nil {
		return c.handleError(strconv.FormatInt(id, 10), "backend", backend, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}
	return nil
}

// CreateStickRule creates a stick rule in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) CreateStickRule(backend string, data *models.StickRule, transactionID string, version int64) error {
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

	if err := p.Insert(parser.Backends, backend, "stick", serializeStickRule(*data), int(*data.ID)); err != nil {
		return c.handleError(strconv.FormatInt(*data.ID, 10), "backend", backend, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}
	return nil
}

// EditStickRule edits a stick rule in configuration. One of version or transactionID is
// mandatory. Returns error on fail, nil on success.
func (c *Client) EditStickRule(id int64, backend string, data *models.StickRule, transactionID string, version int64) error {
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

	if _, err := p.GetOne(parser.Backends, backend, "stick", int(id)); err != nil {
		return c.handleError(strconv.FormatInt(*data.ID, 10), "backend", backend, t, transactionID == "", err)
	}

	if err := p.Set(parser.Backends, backend, "stick", serializeStickRule(*data), int(id)); err != nil {
		return c.handleError(strconv.FormatInt(*data.ID, 10), "backend", backend, t, transactionID == "", err)
	}

	if err := c.saveData(p, t, transactionID == ""); err != nil {
		return err
	}
	return nil
}

func (c *Client) parseStickRules(backend string, p *parser.Parser) (models.StickRules, error) {
	sr := models.StickRules{}

	data, err := p.Get(parser.Backends, backend, "stick", false)
	if err != nil {
		if err == parser_errors.ErrFetch {
			return sr, nil
		}
		return nil, err
	}

	sRules := data.([]types.Stick)
	for i, sRule := range sRules {
		id := int64(i)
		s := parseStickRule(sRule)
		if s != nil {
			s.ID = &id
			sr = append(sr, s)
		}
	}
	return sr, nil
}

func parseStickRule(s types.Stick) *models.StickRule {
	return &models.StickRule{
		Type:     s.Type,
		Table:    s.Table,
		Pattern:  s.Pattern,
		Cond:     s.Cond,
		CondTest: s.CondTest,
	}
}

func serializeStickRule(sRule models.StickRule) types.Stick {
	sr := types.Stick{
		Type:     sRule.Type,
		Table:    sRule.Table,
		Pattern:  sRule.Pattern,
		Cond:     sRule.Cond,
		CondTest: sRule.CondTest,
	}
	return sr
}
