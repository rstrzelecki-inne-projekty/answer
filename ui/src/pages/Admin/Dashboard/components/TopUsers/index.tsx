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

import { FC, useState } from 'react';
import { Button, ButtonGroup, Card, Form, Table } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import dayjs from 'dayjs';

import { BaseUserCard } from '@/components';
import { useQueryActivityLogTopUsers } from '@/services';

// [cd] AA-51: the 50 most active users in a date range (activity = questions + answers + comments);
// every number opens the activity log filtered to that user, range and action
const COLUMNS: { key: string; actions: string }[] = [
  { key: 'questions', actions: 'question.create' },
  { key: 'answers', actions: 'answer.create' },
  { key: 'comments', actions: 'comment.create,comment.reply' },
  { key: 'reviews', actions: 'review.queued' },
  { key: 'badges', actions: 'badge.award' },
  { key: 'views', actions: 'page.view' },
];
const PRESETS = [1, 7, 14, 30];

const TopUsers: FC = () => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.dashboard.top_users',
  });
  const today = dayjs().format('YYYY-MM-DD');
  const [from, setFrom] = useState(today);
  const [to, setTo] = useState(today);
  const fromUnix = dayjs(from).startOf('day').unix();
  const toUnix = dayjs(to).add(1, 'day').startOf('day').unix();
  const { data } = useQueryActivityLogTopUsers({
    from: fromUnix,
    to: toUnix,
    limit: 50,
  });

  const applyPreset = (days: number) => {
    setFrom(
      dayjs()
        .subtract(days - 1, 'day')
        .format('YYYY-MM-DD'),
    );
    setTo(today);
  };
  const activePreset = PRESETS.find(
    (d) =>
      to === today &&
      from ===
        dayjs()
          .subtract(d - 1, 'day')
          .format('YYYY-MM-DD'),
  );

  const logLink = (username: string, actions: string) =>
    `/admin/activity-log?range=custom&from=${from}&to=${to}&username=${encodeURIComponent(username)}${
      actions ? `&action=${encodeURIComponent(actions)}` : ''
    }`;

  return (
    <Card className="mb-4">
      <Card.Body>
        <div className="d-flex flex-wrap justify-content-between align-items-center gap-2 mb-3">
          <h6 className="mb-0">{t('title')}</h6>
          <div className="d-flex flex-wrap align-items-center gap-2">
            <ButtonGroup size="sm">
              {PRESETS.map((d) => (
                <Button
                  key={d}
                  variant={activePreset === d ? 'primary' : 'outline-secondary'}
                  onClick={() => applyPreset(d)}>
                  {d === 1 ? t('today') : t('last_days', { count: d })}
                </Button>
              ))}
            </ButtonGroup>
            <Form.Control
              size="sm"
              type="date"
              value={from}
              max={to}
              onChange={(e) => e.target.value && setFrom(e.target.value)}
              style={{ width: '10rem' }}
            />
            <span className="text-secondary">–</span>
            <Form.Control
              size="sm"
              type="date"
              value={to}
              min={from}
              max={today}
              onChange={(e) => e.target.value && setTo(e.target.value)}
              style={{ width: '10rem' }}
            />
          </div>
        </div>
        <Table size="sm" responsive className="mb-0 align-middle">
          <thead>
            <tr>
              <th style={{ width: '3rem' }}>#</th>
              <th>{t('user')}</th>
              {COLUMNS.map((c) => (
                <th key={c.key} className="text-end">
                  {t(c.key)}
                </th>
              ))}
              <th className="text-end">{t('total')}</th>
            </tr>
          </thead>
          <tbody>
            {(data || []).map((row, i) => (
              <tr key={row.user.id}>
                <td className="text-secondary">{i + 1}</td>
                <td>
                  <BaseUserCard
                    data={row.user}
                    showReputation={false}
                    nameMaxWidth="200px"
                  />
                </td>
                {COLUMNS.map((c) => {
                  const n = Number(row[c.key] || 0);
                  return (
                    <td key={c.key} className="text-end">
                      {n > 0 ? (
                        <a
                          href={logLink(row.user.username, c.actions)}
                          target="_blank"
                          rel="noreferrer"
                          title={t('open_log')}>
                          {n}
                        </a>
                      ) : (
                        <span className="text-secondary">0</span>
                      )}
                    </td>
                  );
                })}
                <td className="text-end fw-bold">
                  <a
                    href={logLink(row.user.username, '')}
                    target="_blank"
                    rel="noreferrer"
                    title={t('open_log')}>
                    {row.total}
                  </a>
                </td>
              </tr>
            ))}
            {data && data.length === 0 ? (
              <tr>
                <td
                  colSpan={COLUMNS.length + 3}
                  className="text-secondary text-center py-3">
                  {t('empty')}
                </td>
              </tr>
            ) : null}
          </tbody>
        </Table>
        <div className="small text-secondary mt-2">{t('hint')}</div>
      </Card.Body>
    </Card>
  );
};

export default TopUsers;
