/**
 * Copyright 2023 Shift Crypto AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import style from './amount.module.css';

type TProps = {
  amount: string;
  unit: string;
  removeBtcTrailingZeroes?: boolean;
};

export const Amount = ({ amount, unit, removeBtcTrailingZeroes }: TProps) => {
  const formatSats = (amount: string): JSX.Element => {
    const blocks: JSX.Element[] = [];
    const len = amount.length;
    const blockSize = 3;

    for (let i = 0; i > -len ; i -= blockSize) {
      blocks.push(
        <span className={i > -len ? style.space : ''}>
          {i === 0 ? amount.slice(i - blockSize) : amount.slice(i - blockSize, i)}
        </span>);
    }
    return <>{blocks.reverse()}</>;
  };

  switch (unit) {
  case 'BTC':
  case 'TBTC':
  case 'LTC':
  case 'TLTC':
    if (removeBtcTrailingZeroes && amount.includes('.')) {
      return <>{amount.replace(/\.?0+$/, '')}</>;
    }
    break;
  case 'sat':
  case 'tsat':
    return formatSats(amount);
  }
  return <>{amount}</>;

};
