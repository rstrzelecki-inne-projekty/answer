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
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { pathFactory } from '@/router/pathFactory';
import {
  postKBImportCheck,
  postKBImportRow,
  KBImportCheckResp,
  KBImportRow,
} from '@/services';

type Result = {
  state: 'waiting' | 'published' | 'pending' | 'exists' | 'failed';
  questionId?: string;
  urlTitle?: string;
};

// Excel with the Polish locale saves text files in Windows-1250; everything else is UTF-8
const decodeFile = (buf: ArrayBuffer) => {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(buf);
  } catch {
    return new TextDecoder('windows-1250').decode(buf);
  }
};

const STATUS_CLASS: Record<KBImportRow['status'], string> = {
  new: 'text-primary',
  exists: 'text-secondary',
  duplicate: 'text-warning',
  invalid: 'text-danger',
};

const RESULT_CLASS: Record<Result['state'], string> = {
  waiting: 'text-secondary',
  published: 'text-success',
  pending: 'text-warning',
  exists: 'text-secondary',
  failed: 'text-danger',
};

// [cd] ready question–answer pairs -> questions with an accepted answer by the knowledge-base account
const Index: FC = () => {
  const { t } = useTranslation('translation', {
    keyPrefix: 'admin.kb_import',
  });
  const [content, setContent] = useState('');
  const [source, setSource] = useState('');
  const [check, setCheck] = useState<KBImportCheckResp | null>(null);
  const [busy, setBusy] = useState(false);
  const [onlyProblems, setOnlyProblems] = useState(false);
  const [results, setResults] = useState<Record<number, Result>>({});
  const running = useRef(false);

  const values = Object.values(results);
  const count = (s: Result['state']) =>
    values.filter((r) => r.state === s).length;
  const publishing = values.some((r) => r.state === 'waiting');
  const finished = values.length - count('waiting');

  useEffect(() => {
    if (!publishing) return undefined;
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [publishing]);

  const reset = () => {
    setCheck(null);
    setResults({});
  };

  const onFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    file.arrayBuffer().then((buf) => {
      setContent(decodeFile(buf));
      if (!source) setSource(file.name);
      reset();
    });
  };

  const runCheck = () => {
    setBusy(true);
    setResults({});
    postKBImportCheck(content)
      .then(setCheck)
      .finally(() => setBusy(false));
  };

  // one row at a time: progress per row, the reviewer and the vector index keep up
  const publish = async (rows: KBImportRow[]) => {
    if (running.current || rows.length === 0) return;
    running.current = true;
    setResults((prev) => {
      const next = { ...prev };
      rows.forEach((r) => {
        next[r.line] = { state: 'waiting' };
      });
      return next;
    });
    for (let i = 0; i < rows.length; i += 1) {
      const r = rows[i];
      let res: Result;
      try {
        // eslint-disable-next-line no-await-in-loop
        const out = await postKBImportRow({
          question: r.question,
          answer: r.answer,
          tags: r.tags,
          source: source.trim(),
        });
        res = {
          state: out.status,
          questionId: out.question_id,
          urlTitle: out.url_title,
        };
      } catch {
        res = { state: 'failed' };
      }
      setResults((prev) => ({ ...prev, [r.line]: res }));
    }
    running.current = false;
  };

  const toPublish = (check?.rows || []).filter((r) => r.status === 'new');
  const retry = () =>
    publish(
      (check?.rows || []).filter((r) => results[r.line]?.state === 'failed'),
    );

  const rows = useMemo(
    () =>
      (check?.rows || []).filter(
        (r) =>
          !onlyProblems ||
          ['duplicate', 'invalid'].includes(r.status) ||
          r.tags.some((tag) => check?.unknown_tags.includes(tag)) ||
          results[r.line]?.state === 'failed',
      ),
    [check, onlyProblems, results],
  );

  const statusText = (r: KBImportRow) => {
    if (r.status === 'duplicate') {
      return t('status.duplicate', { line: r.duplicate_of });
    }
    if (r.status === 'invalid' && r.message) {
      return `${t('status.invalid')}: ${t(`reason.${r.message.split(':')[0]}`)}`;
    }
    return t(`status.${r.status}`);
  };

  const canPublish =
    !!check &&
    check.account_found &&
    toPublish.length > 0 &&
    values.length === 0 &&
    !busy;

  return (
    <>
      <h3 className="mb-3">{t('title')}</h3>
      <p className="text-secondary" style={{ maxWidth: 820 }}>
        {t('intro')}
      </p>

      <Form.Group className="mb-3" controlId="kb-import-file">
        <Form.Label>{t('file')}</Form.Label>
        <Form.Control
          type="file"
          accept=".tsv,.csv,.txt,text/tab-separated-values,text/csv,text/plain"
          onChange={onFile}
          disabled={busy || publishing}
        />
      </Form.Group>
      <Form.Group className="mb-3" controlId="kb-import-content">
        <Form.Label>{t('paste')}</Form.Label>
        <Form.Control
          as="textarea"
          rows={8}
          className="font-monospace small"
          placeholder={t('paste_placeholder')}
          value={content}
          disabled={busy || publishing}
          onChange={(e) => {
            setContent(e.target.value);
            reset();
          }}
        />
      </Form.Group>
      <Form.Group
        className="mb-3"
        controlId="kb-import-source"
        style={{ maxWidth: 520 }}>
        <Form.Label>{t('source')}</Form.Label>
        <Form.Control
          type="text"
          maxLength={200}
          placeholder={t('source_placeholder')}
          value={source}
          disabled={publishing}
          onChange={(e) => setSource(e.target.value)}
        />
      </Form.Group>

      <div className="d-flex flex-wrap align-items-center gap-3 mb-3">
        <Button
          variant="outline-primary"
          disabled={!content.trim() || busy || publishing}
          onClick={runCheck}>
          {t('check')}
        </Button>
        {check && values.length === 0 ? (
          <Button
            variant="primary"
            disabled={!canPublish}
            onClick={() => publish(toPublish)}>
            {t('publish', { count: toPublish.length })}
          </Button>
        ) : null}
      </div>

      {check ? (
        <div style={{ maxWidth: 820 }}>
          {check.account_found ? (
            <Alert variant="light" className="py-2 small">
              {t('account', {
                name: check.display_name,
                username: check.username,
                tag: check.base_tag,
              })}
            </Alert>
          ) : (
            <Alert variant="danger" className="py-2 small">
              {t('account_missing', { username: check.username })}
            </Alert>
          )}
          {!check.base_tag_found ? (
            <Alert variant="warning" className="py-2 small">
              {t('base_tag_missing', { tag: check.base_tag })}
            </Alert>
          ) : null}
          {check.unknown_tags.length > 0 ? (
            <Alert variant="warning" className="py-2 small">
              {t('unknown_tags', { tags: check.unknown_tags.join(', ') })}
            </Alert>
          ) : null}
        </div>
      ) : null}

      {values.length > 0 ? (
        <div className="mb-3" style={{ maxWidth: 820 }}>
          <ProgressBar
            now={(finished / values.length) * 100}
            variant={count('failed') ? 'warning' : 'success'}
            className="mb-2"
          />
          {publishing ? (
            <div className="small">
              {t('publishing', { done: finished, total: values.length })}
            </div>
          ) : (
            <Alert
              variant={count('failed') ? 'warning' : 'success'}
              className="mt-2">
              {t('done', {
                published: count('published'),
                pending: count('pending'),
                failed: count('failed'),
              })}
              {count('failed') > 0 ? (
                <Button size="sm" variant="link" onClick={retry}>
                  {t('retry')}
                </Button>
              ) : null}
            </Alert>
          )}
        </div>
      ) : null}

      {check ? (
        <>
          <div className="small text-secondary mb-2">
            {t('delimiter', { delimiter: check.delimiter })}
            {' · '}
            {t('summary')}:{' '}
            {Object.entries(check.statuses || {})
              .map(([k, v]) => `${t(`status.${k}`, { line: '…' })}: ${v}`)
              .join(', ')}
          </div>
          <Form.Check
            id="kb-import-only-problems"
            className="mb-2 small"
            checked={onlyProblems}
            onChange={(e) => setOnlyProblems(e.target.checked)}
            label={t('only_problems')}
          />
          <Table responsive="md" size="sm" className="align-middle small">
            <thead>
              <tr>
                <th style={{ width: '6%' }}>{t('col.line')}</th>
                <th style={{ width: '26%' }}>{t('col.question')}</th>
                <th>{t('col.answer')}</th>
                <th style={{ width: '14%' }}>{t('col.tags')}</th>
                <th style={{ width: '14%' }}>{t('col.status')}</th>
                {values.length > 0 ? (
                  <th style={{ width: '12%' }}>{t('col.result')}</th>
                ) : null}
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => {
                const res = results[r.line];
                const qid = res?.questionId || r.question_id;
                return (
                  <tr key={r.line}>
                    <td>{r.line}</td>
                    <td className="text-break">
                      {qid ? (
                        <Link
                          to={pathFactory.questionLanding(qid, res?.urlTitle)}
                          target="_blank">
                          {r.question}
                        </Link>
                      ) : (
                        r.question
                      )}
                    </td>
                    <td className="text-break text-secondary">
                      {r.answer.length > 160
                        ? `${r.answer.slice(0, 160)}…`
                        : r.answer}
                    </td>
                    <td>
                      {r.tags.map((tag) => (
                        <span
                          key={tag}
                          className={`me-1 ${
                            check.unknown_tags.includes(tag)
                              ? 'text-danger text-decoration-line-through'
                              : ''
                          }`}>
                          {tag}
                        </span>
                      ))}
                    </td>
                    <td className={STATUS_CLASS[r.status]}>{statusText(r)}</td>
                    {values.length > 0 ? (
                      <td className={res ? RESULT_CLASS[res.state] : ''}>
                        {res ? t(`result.${res.state}`) : ''}
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
