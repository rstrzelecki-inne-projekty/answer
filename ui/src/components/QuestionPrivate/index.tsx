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

import { loggedUserInfoStore } from '@/stores';
import { setQuestionPrivate } from '@/services';
import { useToast } from '@/hooks';

interface Props {
  questionId: string;
  /** the person who asked */
  authorId?: string;
  isPrivate?: boolean;
  onChange?: (isPrivate: boolean) => void;
  className?: string;
}

// [cd] The PRIV badge plus the switch. The switch belongs to the person who asked and to the staff;
// the server checks the same thing, the button only decides what is worth showing.
const QuestionPrivate: FC<Props> = ({
  questionId,
  authorId,
  isPrivate = false,
  onChange,
  className = '',
}) => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'private_question',
  });
  const { user } = loggedUserInfoStore();
  const toast = useToast();
  const [saving, setSaving] = useState(false);

  const isStaff = user?.role_id === 2 || user?.role_id === 3;
  const canSwitch = Boolean(user?.id) && (isStaff || user?.id === authorId);

  if (!isPrivate && !canSwitch) {
    return null;
  }

  const handleSwitch = () => {
    const next = !isPrivate;
    setSaving(true);
    setQuestionPrivate(questionId, next)
      .then(() => {
        toast.onShow({
          msg: next ? t('turned_on') : t('turned_off'),
          variant: 'success',
        });
        onChange?.(next);
      })
      .finally(() => setSaving(false));
  };

  return (
    <span className={`d-inline-flex align-items-center gap-2 ${className}`}>
      {isPrivate ? (
        <span
          className="badge text-bg-secondary"
          title={t('badge_title')}
          aria-label={t('badge_title')}>
          {t('badge')}
        </span>
      ) : null}
      {canSwitch ? (
        <Button
          variant="link"
          size="sm"
          className="p-0 text-decoration-none"
          disabled={saving}
          onClick={handleSwitch}>
          {isPrivate ? t('make_public') : t('make_private')}
        </Button>
      ) : null}
    </span>
  );
};

export default QuestionPrivate;
