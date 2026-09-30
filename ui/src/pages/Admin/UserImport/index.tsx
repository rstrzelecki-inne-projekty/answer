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

import { FC, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Form, ProgressBar, Table } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

import dayjs from 'dayjs';

import {
  postImportUsers,
  postUserInvite,
  ImportUsersResp,
  ImportUsersRow,
} from '@/services';

type InviteState = 'pending' | 'sent' | 'failed';

// Excel with the Polish locale saves CSV in Windows-1250; everything else is UTF-8
const decodeFile = (buf: ArrayBuffer) => {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buf);
  } catch {
    return new TextDecoder('windows-1250').decode(buf);
  }
};

const STATUS_VARIANT: Record<ImportUsersRow['status'], string> = {
  new: 'text-primary',
  created: 'text-success',
  exists: 'text-secondary',
  duplicate: 'text-warning',
  invalid: 'text-danger',
  error: 'text-danger',
};

// [cd] partners without the Google login: CSV/TSV -> password accounts -> invitations with a "set your password" link
const Index: FC = () => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.user_import',
  });
  const [content, setContent] = useState('');
  const [preview, setPreview] = useState<ImportUsersResp | null>(null);
  const [result, setResult] = useState<ImportUsersResp | null>(null);
  const [busy, setBusy] = useState(false);
  const [sendInvites, setSendInvites] = useState(true);
  const [inviteExisting, setInviteExisting] = useState(false);
  const [onlyProblems, setOnlyProblems] = useState(false);
  const [invites, setInvites] = useState<Record<string, InviteState>>({});
  const sending = useRef(false);

  const report = result || preview;
  const neverLoggedIn = (preview?.rows || []).filter(
    (r) => r.status === 'exists' && !r.last_login_at,
  ).length;
  const inviteValues = Object.values(invites);
  const sent = inviteValues.filter((s) => s === 'sent').length;
  const failed = inviteValues.filter((s) => s === 'failed').length;
  const inviteTotal = inviteValues.length;
  const inviting = inviteValues.some((s) => s === 'pending');

  useEffect(() => {
    if (!inviting) return undefined;
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [inviting]);

  const onFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    file.arrayBuffer().then((buf) => {
      setContent(decodeFile(buf));
      setPreview(null);
      setResult(null);
      setInvites({});
    });
  };

  const check = () => {
    setBusy(true);
    setResult(null);
    setInvites({});
    postImportUsers(content, true)
      .then((res) => {
        setPreview(res);
        setInviteExisting(false);
      })
      .finally(() => setBusy(false));
  };

  // one invitation at a time: shows progress and stays within the SMTP send rate
  const sendAll = async (userIds: string[]) => {
    if (sending.current || userIds.length === 0) return;
    sending.current = true;
    setInvites((prev) => {
      const next = { ...prev };
      userIds.forEach((id) => {
        next[id] = 'pending';
      });
      return next;
    });
    for (let i = 0; i < userIds.length; i += 1) {
      const id = userIds[i];
      let state: InviteState = 'sent';
      try {
        // eslint-disable-next-line no-await-in-loop
        await postUserInvite(id);
      } catch {
        state = 'failed';
      }
      setInvites((prev) => ({ ...prev, [id]: state }));
    }
    sending.current = false;
  };

  const runImport = () => {
    setBusy(true);
    postImportUsers(content, false)
      .then((res) => {
        setResult(res);
        if (!sendInvites) return;
        const ids = res.rows
          .filter(
            (r) =>
              r.user_id &&
              (r.status === 'created' ||
                (inviteExisting && r.status === 'exists' && !r.last_login_at)),
          )
          .map((r) => r.user_id as string);
        sendAll(ids);
      })
      .finally(() => setBusy(false));
  };

  const retry = () =>
    sendAll(
      Object.entries(invites)
        .filter(([, s]) => s === 'failed')
        .map(([id]) => id),
    );

  const rows = useMemo(
    () =>
      (report?.rows || []).filter(
        (r) =>
          !onlyProblems ||
          ['duplicate', 'invalid', 'error'].includes(r.status) ||
          (r.user_id && invites[r.user_id] === 'failed'),
      ),
    [report, onlyProblems, invites],
  );

  const statusText = (r: ImportUsersRow) => {
    if (r.status === 'duplicate') {
      return t('status.duplicate', { line: r.duplicate_of });
    }
    const base = t(`status.${r.status}`);
    if (r.status === 'invalid' && r.message) {
      const code = r.message.split(':')[0];
      return `${base}: ${t(`reason.${code}`)}`;
    }
    if (r.status === 'error' && r.message) return `${base}: ${r.message}`;
    if (r.status === 'exists') {
      return `${base} (${
        r.last_login_at
          ? t('last_login', {
              date: dayjs.unix(r.last_login_at).format('YYYY-MM-DD'),
            })
          : t('never_logged_in')
      })`;
    }
    return base;
  };

  const toCreate = preview?.statuses?.new || 0;
  const canImport =
    !!preview &&
    !result &&
    !busy &&
    (toCreate > 0 || (sendInvites && inviteExisting && neverLoggedIn > 0));

  return (
    <>
      <h3 className="mb-3">{t('title')}</h3>
      <p className="text-secondary" style={{ maxWidth: 820 }}>
        {t('intro')}
      </p>

      <Form.Group className="mb-3" controlId="user-import-file">
        <Form.Label>{t('file')}</Form.Label>
        <Form.Control
          type="file"
          accept=".csv,.tsv,.txt,text/csv,text/tab-separated-values,text/plain"
          onChange={onFile}
          disabled={busy || inviting}
        />
      </Form.Group>
      <Form.Group className="mb-3" controlId="user-import-content">
        <Form.Label>{t('paste')}</Form.Label>
        <Form.Control
          as="textarea"
          rows={8}
          className="font-monospace small"
          placeholder={t('paste_placeholder')}
          value={content}
          disabled={busy || inviting}
          onChange={(e) => {
            setContent(e.target.value);
            setPreview(null);
            setResult(null);
            setInvites({});
          }}
        />
      </Form.Group>

      <div className="d-flex flex-wrap align-items-center gap-3 mb-3">
        <Button
          variant="outline-primary"
          disabled={!content.trim() || busy || inviting}
          onClick={check}>
          {t('check')}
        </Button>
        {preview && !result ? (
          <Button variant="primary" disabled={!canImport} onClick={runImport}>
            {t('import', { count: toCreate })}
          </Button>
        ) : null}
      </div>

      {preview && !result ? (
        <div className="mb-3">
          <Form.Check
            id="user-import-send-invites"
            checked={sendInvites}
            onChange={(e) => setSendInvites(e.target.checked)}
            label={t('send_invites')}
          />
          {neverLoggedIn > 0 ? (
            <Form.Check
              id="user-import-invite-existing"
              checked={inviteExisting}
              disabled={!sendInvites}
              onChange={(e) => setInviteExisting(e.target.checked)}
              label={t('invite_existing', { count: neverLoggedIn })}
            />
          ) : null}
        </div>
      ) : null}

      {inviteTotal > 0 ? (
        <div className="mb-3" style={{ maxWidth: 820 }}>
          <ProgressBar
            now={((sent + failed) / inviteTotal) * 100}
            variant={failed ? 'warning' : 'success'}
            className="mb-2"
          />
          {inviting ? (
            <div className="small">
              {t('sending', { done: sent + failed, total: inviteTotal })}
            </div>
          ) : null}
        </div>
      ) : null}

      {result && !inviting ? (
        <Alert
          variant={failed ? 'warning' : 'success'}
          style={{ maxWidth: 820 }}>
          {t('done', {
            created: result.statuses?.created || 0,
            invited: sent,
            failed: failed + (result.statuses?.error || 0),
          })}
          {failed > 0 ? (
            <Button size="sm" variant="link" onClick={retry}>
              {t('retry')}
            </Button>
          ) : null}
        </Alert>
      ) : null}

      {report ? (
        <>
          <div className="small text-secondary mb-2">
            {t('delimiter', { delimiter: report.delimiter })}
            {' · '}
            {t('summary')}:{' '}
            {Object.entries(report.statuses || {})
              .map(([k, v]) => `${t(`status.${k}`, { line: '…' })}: ${v}`)
              .join(', ')}
            {' · '}
            {t('domains')}:{' '}
            {Object.entries(report.domains || {})
              .sort((a, b) => b[1] - a[1])
              .map(([k, v]) => `${k}: ${v}`)
              .join(', ')}
          </div>
          <Form.Check
            id="user-import-only-problems"
            className="mb-2 small"
            checked={onlyProblems}
            onChange={(e) => setOnlyProblems(e.target.checked)}
            label={t('only_problems')}
          />
          <Table responsive="md" size="sm" className="align-middle small">
            <thead>
              <tr>
                <th style={{ width: '7%' }}>{t('col.line')}</th>
                <th style={{ width: '24%' }}>{t('col.name')}</th>
                <th style={{ width: '30%' }}>{t('col.email')}</th>
                <th>{t('col.status')}</th>
                {inviteTotal > 0 ? (
                  <th style={{ width: '12%' }}>{t('col.invite')}</th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const inv = r.user_id ? invites[r.user_id] : undefined;
                return (
                  <tr key={`${r.line}-${r.email}`}>
                    <td>{r.line}</td>
                    <td className="text-break">{r.display_name}</td>
                    <td className="text-break">{r.email}</td>
                    <td className={STATUS_VARIANT[r.status]}>
                      {statusText(r)}
                    </td>
                    {inviteTotal > 0 ? (
                      <td
                        className={
                          inv === 'failed'
                            ? 'text-danger'
                            : inv === 'sent'
                              ? 'text-success'
                              : 'text-secondary'
                        }>
                        {inv ? t(`invite_state.${inv}`) : ''}
                      </td>
                    ) : null}
                  </tr>
                );
              })}
            </tbody>
          </Table>
        </>
      ) : null}
    </>
  );
};

export default Index;
