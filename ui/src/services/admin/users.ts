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

import qs from 'qs';
import useSWR from 'swr';

import request from '@/utils/request';
import type * as Type from '@/common/interface';

export const changeUserStatus = (params) => {
  return request.put('/answer/admin/api/user/status', params);
};

export const useQueryUsers = (params) => {
  const apiUrl = `/answer/admin/api/users/page?${qs.stringify(params)}`;
  const { data, error, mutate } = useSWR<Type.ListResult, Error>(
    apiUrl,
    request.instance.get,
  );
  return {
    data,
    isLoading: !data && !error,
    error,
    mutate,
  };
};

export const getUserRoles = () => {
  return request.get('/answer/admin/api/roles');
};

export const changeUserRole = (params) => {
  return request.put('/answer/admin/api/user/role', params);
};

export const addUser = (params: {
  display_name: string;
  email: string;
  password: string;
}) => {
  return request.post('/answer/admin/api/user', params);
};

export const addUsers = (params: { users: string }) => {
  return request.post('/answer/admin/api/users', params);
};

export const updateUserPassword = (params: {
  password: string;
  user_id: string;
}) => {
  return request.put('/answer/admin/api/user/password', params);
};

export const updateUserProfile = (params: {
  display_name: string;
  username: string;
  email: string;
  user_id: string;
}) => {
  return request.put('/answer/admin/api/user/profile', params);
};

export const getUserActivation = (userId: string) => {
  const apiUrl = `/answer/admin/api/user/activation`;
  return request.get<{
    activation_url: string;
  }>(apiUrl, {
    params: {
      user_id: userId,
    },
  });
};

export const postUserActivation = (userId: string) => {
  const apiUrl = `/answer/admin/api/user/activation`;
  return request.post(apiUrl, {
    user_id: userId,
  });
};

// [cd] invitation with a "set your password" link (valid USER_INVITE_LINK_DAYS days)
export const postUserInvite = (userId: string) => {
  return request.post<{ expires_at: number }>('/answer/admin/api/user/invite', {
    user_id: userId,
  });
};

export interface ImportUsersRow {
  line: number;
  display_name: string;
  email: string;
  status: 'new' | 'created' | 'exists' | 'duplicate' | 'invalid' | 'error';
  message?: string;
  duplicate_of?: number;
  user_id?: string;
  last_login_at: number;
}

export interface ImportUsersResp {
  delimiter: string;
  rows: ImportUsersRow[];
  statuses: Record<string, number>;
  domains: Record<string, number>;
}

// [cd] create password accounts from a CSV/TSV; dry_run only checks the rows
export const postImportUsers = (content: string, dryRun: boolean) => {
  // creating hundreds of accounts (bcrypt per row) takes longer than the default 10 s timeout
  return request.post<ImportUsersResp>(
    '/answer/admin/api/users/import',
    { content, dry_run: dryRun },
    { timeout: 600000 },
  );
};

export interface KBImportRow {
  line: number;
  question: string;
  answer: string;
  tags: string[];
  status: 'new' | 'exists' | 'duplicate' | 'invalid';
  message?: string;
  duplicate_of?: number;
  question_id?: string;
}

export interface KBImportCheckResp {
  delimiter: string;
  rows: KBImportRow[];
  statuses: Record<string, number>;
  unknown_tags: string[];
  username: string;
  display_name: string;
  account_found: boolean;
  base_tag: string;
  base_tag_found: boolean;
}

// [cd] knowledge base import: check the pairs, then publish them one by one
export const postKBImportCheck = (content: string) => {
  return request.post<KBImportCheckResp>(
    '/answer/admin/api/kb/import/check',
    { content },
    { timeout: 120000 },
  );
};

export const postKBImportRow = (params: {
  question: string;
  answer: string;
  tags: string[];
  source: string;
}) => {
  return request.post<{
    question_id: string;
    url_title?: string;
    status: 'published' | 'pending' | 'exists';
  }>('/answer/admin/api/kb/import/row', params, { timeout: 60000 });
};

export const useAdminUsersSettings = () => {
  const apiUrl = `/answer/admin/api/siteinfo/users-settings`;
  const { data, error } = useSWR<
    {
      default_avatar: string;
      gravatar_base_url: string;
    },
    Error
  >(apiUrl, request.instance.get);
  return { data, isLoading: !data && !error, error };
};

export const updateAdminUsersSettings = (params: {
  default_avatar: string;
  gravatar_base_url: string;
}) => {
  return request.put('/answer/admin/api/siteinfo/users-settings', params);
};
