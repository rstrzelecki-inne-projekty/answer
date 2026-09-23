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

import { FC, Fragment, memo } from 'react';
import { useTranslation } from 'react-i18next';

import classNames from 'classnames';

import { formatCount } from '@/utils';
import type * as Type from '@/common/interface';

interface Props {
  /** reputation score */
  rank?: number;
  prestige?: Type.UserPrestige | null;
  /** sm hides the rank name and the badge counter, keeping the insignia and the reputation */
  size?: 'sm' | 'md';
  className?: string;
}

const LEVEL_CLASS = {
  3: 'border-warning-subtle bg-warning-subtle',
  2: 'border-secondary-subtle bg-secondary-subtle',
};

const AwardIcon = () => (
  <svg
    viewBox="0 0 16 16"
    width="12"
    height="12"
    fill="currentColor"
    aria-hidden="true">
    <path d="M9.669.864 8 0 6.331.864l-1.858.282-.842 1.68-1.337 1.32L2.6 6l-.306 1.854 1.337 1.32.842 1.68 1.858.282L8 12l1.669-.864 1.858-.282.842-1.68 1.337-1.32L13.4 6l.306-1.854-1.337-1.32-.842-1.68zm1.196 1.193.684 1.365 1.086 1.072L12.387 6l.248 1.506-1.086 1.072-.684 1.365-1.51.229L8 10.874l-1.355-.702-1.51-.229-.684-1.365-1.086-1.072L3.614 6l-.25-1.506 1.087-1.072.684-1.365 1.51-.229L8 1.126z" />
    <path d="M4 11.794V16l4-1 4 1v-4.206l-2.018.306L8 13.126 6.018 12.1z" />
  </svg>
);

const YellowCardIcon = () => (
  <svg viewBox="0 0 16 16" width="9" height="12" aria-hidden="true">
    <rect
      x="2.5"
      y="1"
      width="11"
      height="14"
      rx="2"
      fill="#ffd400"
      stroke="#8a6d00"
      strokeWidth="1.2"
    />
  </svg>
);

const Index: FC<Props> = ({ rank, prestige, size = 'md', className }) => {
  const { t } = useTranslation('translation', { keyPrefix: 'prestige' });
  const compact = size === 'sm';
  const rankBadgeIcon = prestige?.rank_badge_icon;
  const rankBadgeName = prestige?.rank_badge_name;
  const badgeCount = prestige?.badge_count || 0;
  const yellowCards = prestige?.yellow_cards || 0;
  const showReputation = typeof rank === 'number';

  if (!rankBadgeIcon && !showReputation && !badgeCount && !yellowCards) {
    return null;
  }

  // the pill reads "insignia | rank | reputation | badges", with a hairline between the parts
  const segments: { key: string; node: JSX.Element }[] = [];
  if (!compact && rankBadgeName) {
    segments.push({
      key: 'rank',
      node: (
        <span className="fw-semibold text-body-emphasis">{rankBadgeName}</span>
      ),
    });
  }
  if (showReputation) {
    segments.push({
      key: 'reputation',
      node: (
        <span className="fw-bold text-body-emphasis" title={t('reputation')}>
          {formatCount(rank)}
        </span>
      ),
    });
  }
  if (!compact && badgeCount > 0) {
    segments.push({
      key: 'badges',
      node: (
        <span
          className="d-inline-flex align-items-center gap-1 text-body-secondary"
          title={t('badges', { num: badgeCount })}>
          <AwardIcon />
          {formatCount(badgeCount)}
        </span>
      ),
    });
  }

  const levelClass =
    LEVEL_CLASS[prestige?.rank_badge_level || 0] ||
    'border-secondary-subtle bg-body-tertiary';

  return (
    <span className={classNames('d-inline-flex align-items-center', className)}>
      <span
        className={classNames(
          'd-inline-flex align-items-center gap-1 border rounded-pill lh-1',
          levelClass,
          compact ? 'px-1 py-1' : 'px-2 py-1',
        )}>
        {rankBadgeIcon ? (
          <img
            src={rankBadgeIcon}
            width={17}
            height={17}
            alt={rankBadgeName || t('rank')}
            title={
              prestige?.rank_badge_amount
                ? t('rank_title', {
                    name: rankBadgeName,
                    num: prestige.rank_badge_amount,
                  })
                : rankBadgeName
            }
          />
        ) : null}
        {segments.map((segment, index) => (
          <Fragment key={segment.key}>
            {index > 0 ? (
              <span
                className="border-start opacity-50"
                style={{ height: '12px' }}
                aria-hidden="true"
              />
            ) : null}
            {segment.node}
          </Fragment>
        ))}
      </span>
      {yellowCards > 0 ? (
        <span
          className="d-inline-flex align-items-center gap-1 border border-warning-subtle bg-warning-subtle text-warning-emphasis rounded-pill lh-1 ms-1 px-2 py-1 fw-semibold"
          title={t('yellow_card', { num: yellowCards })}>
          <YellowCardIcon />
          {yellowCards}
        </span>
      ) : null}
    </span>
  );
};

export default memo(Index);
