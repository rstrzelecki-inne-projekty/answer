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

import { FC } from 'react';
import { Row, Col, Button } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import * as Type from '@/common/interface';
import { CardBadge, AwardBadgeButton } from '@/components';
import { useToast } from '@/hooks';
import { revokeBadge } from '@/services';
import { loggedUserInfoStore } from '@/stores';

interface IProps {
  data: Type.BadgeListItem[];
  username: string;
  visible: boolean;
  /** re-fetch the list after an admin awards or revokes a badge */
  onChange?: () => void;
}

const Index: FC<IProps> = ({ data, visible, username, onChange }) => {
  const { t } = useTranslation('translation', { keyPrefix: 'badges.award' });
  const isAdmin = loggedUserInfoStore((state) => state.user?.role_id) === 2;
  const Toast = useToast();
  if (!visible) {
    return null;
  }
  const handleRevoke = (badgeId: string) => {
    if (!window.confirm(t('revoke_confirm'))) {
      return;
    }
    revokeBadge({ badge_id: badgeId, username })
      .then(() => {
        Toast.onShow({ msg: t('revoked'), variant: 'success' });
        onChange?.();
      })
      .catch((err) => Toast.onShow({ msg: err?.msg, variant: 'danger' }));
  };
  return (
    <>
      {isAdmin && (
        <div className="d-flex justify-content-end mb-3">
          <AwardBadgeButton username={username} onAwarded={onChange} />
        </div>
      )}
      <Row>
        {data.map((item) => {
          return (
            <Col sm={6} md={4} lg={3} key={item.id} className="mb-4">
              <CardBadge
                data={item}
                urlSearchParams={`username=${username}`}
                badgePillType="count"
              />
              {isAdmin && (
                <div className="text-center mt-1 mb-2">
                  <Button
                    variant="link"
                    size="sm"
                    className="text-secondary p-0"
                    onClick={() => handleRevoke(item.id)}>
                    {t('revoke')}
                  </Button>
                </div>
              )}
            </Col>
          );
        })}
      </Row>
    </>
  );
};

export default Index;
