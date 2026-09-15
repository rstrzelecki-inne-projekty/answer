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

import { FC, useEffect, useState } from 'react';
import { Button, Form, Modal } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import useSWR from 'swr';
import qs from 'qs';

import { Icon } from '@/components';
import { useToast } from '@/hooks';
import { useGetAllBadges, awardBadge } from '@/services';
import request from '@/utils/request';
import { loggedUserInfoStore } from '@/stores';

interface Props {
  /** fixed recipient; when omitted the modal asks for a user */
  username?: string;
  /** fixed badge; when omitted the modal asks for a badge */
  badgeId?: string;
  /** 'icon' renders a small inline trophy, 'button' a regular button */
  variant?: 'icon' | 'button';
  className?: string;
  onAwarded?: () => void;
}

/**
 * Admin-only control to award a badge to a user manually. Rendered as nothing for non-admins.
 */
const AwardBadgeButton: FC<Props> = ({
  username,
  badgeId,
  variant = 'button',
  className = '',
  onAwarded,
}) => {
  const { t } = useTranslation('translation', { keyPrefix: 'badges.award' });
  const roleId = loggedUserInfoStore((state) => state.user?.role_id);
  const Toast = useToast();
  const [show, setShow] = useState(false);
  const [user, setUser] = useState(username || '');
  const [query, setQuery] = useState('');
  const [badge, setBadge] = useState(badgeId || '');
  const [awardKey, setAwardKey] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const { data: groups } = useGetAllBadges();
  const { data: users } = useSWR<{ list: any[] }>(
    show && !username && query.length >= 2
      ? `/answer/admin/api/users/page?${qs.stringify({ page: 1, page_size: 8, query })}`
      : null,
    request.instance.get,
  );

  useEffect(() => {
    setUser(username || '');
    setBadge(badgeId || '');
  }, [username, badgeId, show]);

  if (roleId !== 2) {
    return null;
  }

  const handleSubmit = () => {
    if (!user || !badge) {
      return;
    }
    setSubmitting(true);
    awardBadge({
      badge_id: badge,
      username: user,
      award_key: awardKey.trim() || undefined,
    })
      .then(() => {
        Toast.onShow({
          msg: t('success', { username: user }),
          variant: 'success',
        });
        setShow(false);
        setAwardKey('');
        onAwarded?.();
      })
      .catch((err) => {
        Toast.onShow({ msg: err?.msg || t('failed'), variant: 'danger' });
      })
      .finally(() => setSubmitting(false));
  };

  const open = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setShow(true);
  };

  return (
    <>
      {variant === 'icon' ? (
        <Button
          variant="link"
          size="sm"
          className={`p-0 align-baseline text-secondary ${className}`}
          title={t('btn')}
          aria-label={t('btn')}
          onClick={open}>
          <Icon name="award" />
        </Button>
      ) : (
        <Button
          variant="outline-primary"
          size="sm"
          className={className}
          onClick={open}>
          <Icon name="award" className="me-1" />
          {t('btn')}
        </Button>
      )}
      <Modal
        show={show}
        onHide={() => setShow(false)}
        onClick={(e) => e.stopPropagation()}>
        <Modal.Header closeButton>
          <Modal.Title as="h5">{t('title')}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Form>
            <Form.Group className="mb-3" controlId="award_badge_user">
              <Form.Label>{t('user')}</Form.Label>
              {username ? (
                <Form.Control value={username} disabled />
              ) : (
                <>
                  <Form.Control
                    value={user || query}
                    placeholder={t('user_placeholder')}
                    autoComplete="off"
                    onChange={(e) => {
                      setQuery(e.target.value);
                      setUser('');
                    }}
                  />
                  {!user && users?.list?.length ? (
                    <div className="list-group mt-1">
                      {users.list.map((u) => (
                        <button
                          type="button"
                          key={u.user_id}
                          className="list-group-item list-group-item-action py-1"
                          onClick={() => {
                            setUser(u.username);
                            setQuery('');
                          }}>
                          {u.display_name}{' '}
                          <span className="text-secondary small">
                            @{u.username}
                          </span>
                        </button>
                      ))}
                    </div>
                  ) : null}
                </>
              )}
            </Form.Group>
            <Form.Group className="mb-3" controlId="award_badge_badge">
              <Form.Label>{t('badge')}</Form.Label>
              <Form.Select
                value={badge}
                disabled={!!badgeId}
                onChange={(e) => setBadge(e.target.value)}>
                <option value="">{t('badge_placeholder')}</option>
                {groups?.map((g) => (
                  <optgroup key={g.group_name} label={g.group_name}>
                    {g.badges.map((b) => (
                      <option key={b.id} value={b.id}>
                        {b.name}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </Form.Select>
            </Form.Group>
            <Form.Group controlId="award_badge_key">
              <Form.Label>{t('key')}</Form.Label>
              <Form.Control
                value={awardKey}
                placeholder={t('key_placeholder')}
                onChange={(e) => setAwardKey(e.target.value)}
              />
              <Form.Text className="text-secondary">{t('key_help')}</Form.Text>
            </Form.Group>
          </Form>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="link" onClick={() => setShow(false)}>
            {t('cancel', { keyPrefix: 'btns' })}
          </Button>
          <Button
            variant="primary"
            disabled={!user || !badge || submitting}
            onClick={handleSubmit}>
            {t('submit')}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default AwardBadgeButton;
