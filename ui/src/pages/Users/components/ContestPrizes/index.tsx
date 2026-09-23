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
import { Button, Modal, Table } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';

// [cd] §4 of the contest rules: the places, their prizes and what has to be reached for them.
// The prize column carries amounts, so it is data rather than translated copy.
const PRIZES: {
  place: string;
  prize: string;
  points: number;
  solutionsKey: string;
}[] = [
  { place: '1', prize: '1 000 zł', points: 350, solutionsKey: 'solutions_24' },
  { place: '2', prize: '600 zł', points: 220, solutionsKey: 'solutions_15' },
  { place: '3', prize: '400 zł', points: 150, solutionsKey: 'solutions_10' },
  { place: '4–5', prize: '200 zł', points: 80, solutionsKey: 'solutions_6' },
  { place: '6–10', prize: '100 zł', points: 40, solutionsKey: 'solutions_3' },
  { place: '11–20', prize: '', points: 18, solutionsKey: 'solutions_2' },
];

const ContestPrizes: FC = () => {
  const { t } = useTranslation('translation', { keyPrefix: 'contest' });
  const [show, setShow] = useState(false);

  return (
    <>
      <Button
        variant="outline-secondary"
        size="sm"
        onClick={() => setShow(true)}>
        {t('prizes')}
      </Button>

      <Modal show={show} onHide={() => setShow(false)} scrollable>
        <Modal.Header closeButton>
          <Modal.Title as="h5">{t('prizes_title')}</Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <Table responsive size="sm" className="align-middle">
            <thead>
              <tr className="text-secondary small">
                <th>{t('col_place')}</th>
                <th>{t('col_prize')}</th>
                <th className="text-end">{t('col_threshold')}</th>
                <th>{t('col_solutions')}</th>
              </tr>
            </thead>
            <tbody>
              {PRIZES.map((row) => (
                <tr key={row.place}>
                  <td className="text-nowrap">{row.place}</td>
                  <td className="text-nowrap">
                    {row.prize || t('prize_in_kind')}
                  </td>
                  <td className="text-end fw-bold text-nowrap">
                    {row.points} {t('points')}
                  </td>
                  <td className="small">{t(row.solutionsKey)}</td>
                </tr>
              ))}
            </tbody>
          </Table>
          <p className="text-secondary small mb-0">{t('prizes_note')}</p>
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

export default ContestPrizes;
