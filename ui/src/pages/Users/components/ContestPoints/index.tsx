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
import { Button, Modal, Table, Alert } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

import dayjs from 'dayjs';

import { loggedUserInfoStore } from '@/stores';
import { useQueryContestMyPoints } from '@/services';

// [cd] §10.2 of the contest rules: every participant can see their own points and how they were counted
const KIND_KEY = {
  solution: 'kind_solution',
  answer_upvote: 'kind_answer_upvote',
  question_upvote: 'kind_question_upvote',
};

const ContestPoints: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'contest' });
  const { user } = loggedUserInfoStore();
  const [show, setShow] = useState(false);
  const { data, isLoading } = useQueryContestMyPoints(show);

  if (!user?.username) {
    return null;
  }

  const day = (unix?: number) =>
    unix ? dayjs.unix(unix).format('D MMM YYYY') : '';

  return (
    <>
      <Button
        variant="outline-secondary"
        size="sm"
        onClick={() => setShow(true)}>
        {t('my_points')}
      </Button>

      <Modal show={show} onHide={() => setShow(false)} size="lg" scrollable>
        <Modal.Header closeButton>
          <Modal.Title as="h5">{t('title')}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          {isLoading && !data ? <p className="mb-0">{t('loading')}</p> : null}

          {data ? (
            <>
              <p className="text-secondary small">
                {t('period', {
                  from: day(data.period_start),
                  to: day(data.period_end),
                })}
              </p>

              {data.excluded ? (
                <Alert variant="secondary" className="small">
                  {t('excluded')}
                </Alert>
              ) : null}

              <div className="d-flex flex-wrap gap-3 mb-3">
                <div>
                  <div className="fs-4 fw-bold">{data.total}</div>
                  <div className="text-secondary small">{t('total')}</div>
                </div>
                <div>
                  <div className="fs-5">{data.answer_points}</div>
                  <div className="text-secondary small">{t('answers')}</div>
                </div>
                <div>
                  <div className="fs-5">{data.question_points}</div>
                  <div className="text-secondary small">{t('questions')}</div>
                </div>
                <div>
                  <div className="fs-5">{data.solved_count}</div>
                  <div className="text-secondary small">{t('solved')}</div>
                </div>
              </div>

              {data.question_points_lost > 0 ? (
                <Alert variant="warning" className="small">
                  {t('questions_lost', { points: data.question_points_lost })}
                </Alert>
              ) : null}

              {data.items.length === 0 ? (
                <p className="text-secondary">{t('empty')}</p>
              ) : (
                <Table responsive size="sm" className="align-middle">
                  <thead>
                    <tr className="text-secondary small">
                      <th>{t('col_date')}</th>
                      <th>{t('col_post')}</th>
                      <th>{t('col_reason')}</th>
                      <th className="text-end">{t('col_points')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((item) => (
                      <tr
                        key={`${item.question_id}-${item.kind}-${item.created_at}`}>
                        <td className="text-secondary small text-nowrap">
                          {day(item.created_at)}
                        </td>
                        <td>
                          <Link
                            to={`/questions/${item.question_id}`}
                            onClick={() => setShow(false)}>
                            {item.title || item.question_id}
                          </Link>
                        </td>
                        <td className="small">
                          {t(KIND_KEY[item.kind] || 'kind_other')}
                          {item.halved ? (
                            <span className="text-secondary">
                              {' · '}
                              {t('halved')}
                            </span>
                          ) : null}
                        </td>
                        <td className="text-end fw-bold text-nowrap">
                          {item.points}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
              )}

              <p className="text-secondary small mb-0">{t('note')}</p>
            </>
          ) : null}
        </Modal.Body>
        <Modal.Footer>
          <Button variant="link" onClick={() => setShow(false)}>
            {t('close', { keyPrefix: 'btns' })}
          </Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default ContestPoints;
