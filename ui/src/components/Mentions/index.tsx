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

import React, { useEffect, useRef, useState, FC } from 'react';
import { Dropdown } from 'react-bootstrap';

import uniqBy from 'lodash/uniqBy';

import { useSearchMentionUsers } from '@/services';
import * as Types from '@/common/interface';
import { registerMentionUser } from '@/utils';

import './index.scss';

interface IProps {
  children: React.ReactNode;
  pageUsers;
  onSelected: (val: string) => void;
}

interface MentionItem {
  displayName: string;
  userName: string;
  email?: string;
}

const MAX_RECODE = 8;

// a word character right before "@" means an e-mail address or a word, not a mention
const notBoundary = /[\p{L}\p{N}_[/@]/u;

/**
 * [cd] 24: suggestions come from the server (every active user, by name, login or e-mail — the e-mail only for
 * people from MENTION_EMAIL_DOMAINS) and from the people commenting on this page; the picked person is written as
 * "@Display Name" and remembered, so the comment is sent with "[@Display Name](/users/login)".
 */
const Mentions: FC<IProps> = ({ children, pageUsers, onSelected }) => {
  const menuRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [val, setValue] = useState('');
  const [cursor, setCursor] = useState(0);
  const { data: found = [] } = useSearchMentionUsers(val);

  const query = val.trim().toLowerCase();
  const fromPage: MentionItem[] = query
    ? ((pageUsers || []) as Types.PageUser[])
        .filter(
          (u) =>
            u.userName &&
            (u.displayName?.toLowerCase().includes(query) ||
              u.userName.toLowerCase().includes(query)),
        )
        .map((u) => ({ displayName: u.displayName, userName: u.userName }))
    : [];
  const fromServer: MentionItem[] = query
    ? (found || []).map((u) => ({
        displayName: u.display_name,
        userName: u.username,
        email: u.e_mail,
      }))
    : [];
  const items = uniqBy([...fromServer, ...fromPage], 'userName').slice(
    0,
    MAX_RECODE,
  );

  const searchUser = () => {
    const element = dropdownRef.current?.children[0] as HTMLTextAreaElement;
    if (!element) {
      return;
    }
    const { value, selectionStart = 0 } = element;
    const before = value.substring(0, selectionStart);
    const at = before.lastIndexOf('@');
    if (at < 0 || (at > 0 && notBoundary.test(before.charAt(at - 1)))) {
      setValue('');
      return;
    }
    const q = before.substring(at + 1);
    // a name is at most two words ("Jan Kowalski"); a line break or a longer text means the mention is over
    if (q.includes('\n') || q.split(' ').length > 2 || q.length > 40) {
      setValue('');
      return;
    }
    setValue(q);
    setCursor(0);
  };

  useEffect(() => {
    const element = dropdownRef.current?.children[0] as HTMLTextAreaElement;

    if (element) {
      element.addEventListener('input', searchUser);
    }
    return () => {
      element?.removeEventListener('input', searchUser);
    };
  }, [dropdownRef]);

  const handleClick = (item: MentionItem) => {
    const element = dropdownRef.current?.children[0] as HTMLTextAreaElement;

    const { value, selectionStart = 0 } = element;

    if (!selectionStart) {
      return;
    }
    const before = value.substring(0, selectionStart);
    const at = before.lastIndexOf('@');
    if (at < 0) {
      return;
    }
    registerMentionUser(item.displayName, item.userName);
    const text = `@${item.displayName} `;
    onSelected(
      `${before.substring(0, at)}${text}${value.substring(selectionStart)}`,
    );
    setValue('');
  };

  const handleKeyDown = (e) => {
    const { keyCode } = e;
    if (items.length === 0) {
      return;
    }
    if (keyCode === 38 && cursor > 0) {
      e.preventDefault();
      setCursor(cursor - 1);
    }
    if (keyCode === 40 && cursor < items.length - 1) {
      e.preventDefault();
      setCursor(cursor + 1);
    }
    if (keyCode === 27) {
      setValue('');
    }
    if (keyCode === 13 && cursor > -1 && cursor <= items.length - 1) {
      e.preventDefault();
      handleClick(items[cursor]);
      setCursor(0);
    }
  };

  return (
    <Dropdown
      ref={dropdownRef}
      className="mentions-wrap"
      show={items.length > 0}
      onKeyDown={handleKeyDown}>
      {children}
      <Dropdown.Menu
        className={items.length > 0 ? 'visible' : 'invisible'}
        ref={menuRef}>
        {items.map((item, index) => {
          return (
            <Dropdown.Item
              className={`${cursor === index ? 'bg-gray-200' : ''}`}
              key={item.userName}
              onClick={() => handleClick(item)}>
              <span className="link-dark me-1">{item.displayName}</span>
              <small className="link-secondary">
                {item.email || `@${item.userName}`}
              </small>
            </Dropdown.Item>
          );
        })}
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default Mentions;
