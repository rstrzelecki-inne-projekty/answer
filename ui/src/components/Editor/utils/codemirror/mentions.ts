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

import {
  autocompletion,
  CompletionContext,
  CompletionResult,
} from '@codemirror/autocomplete';

import request from '@/utils/request';
import type * as Type from '@/common/interface';

// a word character right before "@" means an e-mail address or a word, not a mention
const notBoundary = /[\p{L}\p{N}_[/@]/u;

/**
 * [cd] 26: "@" + two or more characters of a name, login or e-mail (one space allowed, for "Jan Kowalski")
 * lists people from the server; the picked one is written as a profile link "[@Imię Nazwisko](/users/login)",
 * which the preview renders at once and the server keeps as the mention.
 */
async function mentionSource(
  context: CompletionContext,
): Promise<CompletionResult | null> {
  const word = context.matchBefore(/@[^\s@]*(?: [^\s@]*)?/u);
  if (!word) {
    return null;
  }
  const before = context.state.sliceDoc(Math.max(0, word.from - 1), word.from);
  if (before && notBoundary.test(before)) {
    return null;
  }
  const q = word.text.slice(1).trim();
  if (q.length < 2) {
    return null;
  }
  let users: Type.MentionUser[] = [];
  try {
    users = await request.get(
      `/answer/api/v1/user/mention/search?q=${encodeURIComponent(q)}`,
    );
  } catch {
    return null;
  }
  if (!users?.length) {
    return null;
  }
  return {
    from: word.from,
    filter: false,
    options: users.map((u) => ({
      label: u.display_name,
      detail: u.e_mail || `@${u.username}`,
      apply: `[@${u.display_name}](/users/${u.username}) `,
    })),
  };
}

export const mentionCompletion = () =>
  autocompletion({
    override: [mentionSource],
    icons: false,
    activateOnTyping: true,
  });
