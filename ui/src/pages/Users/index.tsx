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

import { Row, Col } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Fragment } from 'react';

import { usePageTags } from '@/hooks';
import { useQueryContributeUsers } from '@/services';
import { Avatar, UserPrestige } from '@/components';
import type * as Type from '@/common/interface';

import ContestPoints from './components/ContestPoints';

const Users = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'users' });

  // [cd] every section ranks by something else, so each one prints its own measure
  const countLabel = (key: string, user: Type.User) => {
    if (key === 'users_with_the_most_vote') {
      return `${user.vote_count} ${t('votes')}`;
    }
    if (key === 'most_active_users') {
      return `${user.activity_count} ${t('activities')}`;
    }
    if (key === 'most_viewing_users') {
      return `${user.view_count} ${t('views')}`;
    }
    if (key === 'contest_ranking') {
      return `${user.contest_points} ${t('points')} · ${user.solved_count} ${t('solutions')}`;
    }
    return `${user.rank} ${t('reputation')}`;
  };

  const { data: users } = useQueryContributeUsers();

  usePageTags({
    title: t('users', { keyPrefix: 'page_title' }),
  });

  if (!users) {
    return null;
  }

  const keys = Object.keys(users);
  return (
    <Row className="py-4 mb-4 d-flex justify-content-center">
      <Col xxl={12}>
        <h3 className="mb-4">{t('title')}</h3>
      </Col>

      <Col xxl={12}>
        {keys.map((key, index) => {
          // [cd] the contest section stays on the page while it is still empty: it carries the
          // rules and the button that shows a participant their own points
          const isContest = key === 'contest_ranking';
          if (users[key]?.length === 0 && !isContest) {
            return null;
          }
          return (
            <Fragment key={key}>
              <Row className="mb-4">
                <Col>
                  <div className="d-flex flex-wrap align-items-center gap-2">
                    <h6 className="mb-0">{t(key)}</h6>
                    {key === 'contest_ranking' && <ContestPoints />}
                  </div>
                  {key === 'contest_ranking' && (
                    <div className="text-secondary small mt-1">
                      {t('contest_ranking_note')}
                    </div>
                  )}
                </Col>
              </Row>
              <Row className={index === keys.length - 1 ? '' : 'mb-4'}>
                {isContest && users[key]?.length === 0 ? (
                  <Col>
                    <p className="text-secondary">{t('contest_empty')}</p>
                  </Col>
                ) : null}
                {users[key]?.map((user) => (
                  <Col
                    key={user.username}
                    xl={3}
                    lg={4}
                    md={4}
                    sm={6}
                    xs={12}
                    className="mb-4">
                    <div className="d-flex">
                      <Link to={`/users/${user.username}`}>
                        <Avatar
                          size="48px"
                          avatar={user?.avatar}
                          searchStr="s=96"
                          alt={user.display_name}
                        />
                      </Link>
                      <div className="ms-2">
                        <Link
                          className="text-break"
                          to={`/users/${user.username}`}>
                          {user.display_name}
                        </Link>
                        <div className="text-secondary small">
                          {countLabel(key, user)}
                        </div>
                        <UserPrestige
                          prestige={user.prestige}
                          className="mt-1 small"
                        />
                      </div>
                    </div>
                  </Col>
                ))}
              </Row>
            </Fragment>
          );
        })}
      </Col>
    </Row>
  );
};

export default Users;
