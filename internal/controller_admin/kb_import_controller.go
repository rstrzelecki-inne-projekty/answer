/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package controller_admin

import (
	"github.com/apache/answer/internal/base/handler"
	"github.com/apache/answer/internal/base/middleware"
	"github.com/apache/answer/internal/schema"
	"github.com/apache/answer/internal/service/kb_import"
	"github.com/gin-gonic/gin"
)

// KBImportController [cd] knowledge base import (question–answer pairs) in the admin panel
type KBImportController struct {
	service *kb_import.KBImportService
}

// NewKBImportController new controller
func NewKBImportController(service *kb_import.KBImportService) *KBImportController {
	return &KBImportController{service: service}
}

// Check parse the pairs and report what would be published
// @Summary check a TSV with question–answer pairs
// @Security ApiKeyAuth
// @Tags admin
// @Produce json
// @Param data body schema.KBImportCheckReq true "KBImportCheckReq"
// @Success 200 {object} handler.RespBody{data=schema.KBImportCheckResp}
// @Router /answer/admin/api/kb/import/check [post]
func (c *KBImportController) Check(ctx *gin.Context) {
	req := &schema.KBImportCheckReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	resp, err := c.service.Check(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}

// Publish one pair as the knowledge-base account
// @Summary publish one question–answer pair to the knowledge base
// @Security ApiKeyAuth
// @Tags admin
// @Produce json
// @Param data body schema.KBImportRowReq true "KBImportRowReq"
// @Success 200 {object} handler.RespBody{data=schema.KBImportRowResp}
// @Router /answer/admin/api/kb/import/row [post]
func (c *KBImportController) Publish(ctx *gin.Context) {
	req := &schema.KBImportRowReq{}
	if handler.BindAndCheck(ctx, req) {
		return
	}
	req.LoginUserID = middleware.GetLoginUserIDFromContext(ctx)
	resp, err := c.service.Publish(ctx, req)
	handler.HandleResponse(ctx, err, resp)
}
