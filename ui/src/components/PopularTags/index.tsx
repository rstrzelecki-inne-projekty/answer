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
import { Button } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';

import classNames from 'classnames';

import { useQueryPopularTags } from '@/services';
import { formatCount } from '@/utils';

// [cd] the periods are computed in the browser, so "today" is the viewer's own midnight
const RANGES: { key: string; from: () => number }[] = [
  {
    key: 'today',
    from: () => {
      const d = new Date();
      d.setHours(0, 0, 0, 0);
      return Math.floor(d.getTime() / 1000);
    },
  },
  { key: 'days_7', from: () => Math.floor(Date.now() / 1000) - 7 * 86400 },
  { key: 'days_14', from: () => Math.floor(Date.now() / 1000) - 14 * 86400 },
  { key: 'days_30', from: () => Math.floor(Date.now() / 1000) - 30 * 86400 },
  { key: 'always', from: () => 0 },
];

const PopularTags: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'popular_tags' });
  const navigate = useNavigate();
  const [urlSearchParams] = useSearchParams();
  const [rangeKey, setRangeKey] = useState('always');
  const range = RANGES.find((r) => r.key === rangeKey) || RANGES[4];
  const { data } = useQueryPopularTags(range.from());
  const activeTag = urlSearchParams.get('tag') || '';

  if (!data?.length && !activeTag) {
    return null;
  }

  return (
    <div className="mt-4 pt-3 border-top">
      <div className="d-flex align-items-center justify-content-between mb-2">
        <h3 className="h6 mb-0 text-body-secondary">{t('title')}</h3>
        {activeTag ? (
          <Button
            variant="link"
            size="sm"
            className="p-0 text-decoration-none"
            onClick={() => navigate('/questions')}>
            {t('reset')}
          </Button>
        ) : null}
      </div>

      <div className="d-flex flex-wrap gap-1 mb-2">
        {RANGES.map((r) => (
          <Button
            key={r.key}
            size="sm"
            variant={r.key === rangeKey ? 'secondary' : 'outline-secondary'}
            className="py-0 px-2 lh-base"
            style={{ fontSize: '12px' }}
            onClick={() => setRangeKey(r.key)}>
            {t(r.key)}
          </Button>
        ))}
      </div>

      <ul className="list-unstyled mb-0 small">
        {data?.map((tag) => (
          <li key={tag.slug_name} className="mb-1">
            <Link
              to={`/questions?tag=${encodeURIComponent(tag.slug_name)}`}
              className={classNames(
                'd-flex justify-content-between align-items-center gap-2 text-decoration-none',
                tag.slug_name === activeTag
                  ? 'fw-bold text-body-emphasis'
                  : 'text-body-secondary',
              )}
              title={t('tag_title', {
                questions: tag.question_count,
                views: tag.view_count,
              })}>
              <span className="text-truncate">{tag.display_name}</span>
              <span className="flex-shrink-0">
                {formatCount(tag.view_count)}
              </span>
            </Link>
          </li>
        ))}
      </ul>

      {data && data.length === 0 ? (
        <p className="text-body-secondary small mb-0">{t('empty')}</p>
      ) : null}
    </div>
  );
};

export default PopularTags;
