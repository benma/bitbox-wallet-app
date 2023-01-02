/**
 * Copyright 2018 Shift Devices AG
 * Copyright 2022 Shift Crypto AG
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

import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { useLoad } from '../../hooks/api';
import * as accountApi from '../../api/account';
import { syncAddressesCount } from '../../api/accountsync';
import { TDevices } from '../../api/devices';
import { isMoonpayBuySupported } from '../../api/backend';
import { useSDCard } from '../../hooks/sdcard';
import { unsubscribe, UnsubscribeList } from '../../utils/subscriptions';
import { statusChanged, syncdone } from '../../api/subscribe-legacy';
import { alertUser } from '../../components/alert/Alert';
import { Balance } from '../../components/balance/balance';
import { AccountGuide } from './guide';
import { HeadersSync } from '../../components/headerssync/headerssync';
import { Header } from '../../components/layout';
import { Info } from '../../components/icon';
import { Spinner } from '../../components/spinner/Spinner';
import Status from '../../components/status/status';
import { Transactions } from '../../components/transactions/transactions';
import { apiGet } from '../../utils/request';
import { BuyCTA } from './info/buyCTA';
import { isBitcoinBased } from './utils';
import { ActionButtons } from './actionButtons';
import style from './account.module.css';

type Props = {
  accounts: accountApi.IAccount[];
  code: string;
  devices: TDevices;
};

export function Account({
  accounts,
  code,
  devices,
}: Props) {
  const { t } = useTranslation();

  const [balance, setBalance] = useState<accountApi.IBalance>(null);
  const [status, setStatus] = useState<accountApi.IStatus>();
  const [syncedAddressesCount, setSyncedAddressesCount] = useState<number>();
  const [transactions, setTransactions] = useState<accountApi.ITransaction[]>();
  const [usesProxy, setUsesProxy] = useState<boolean>();
  const [stateCode, setStateCode] = useState<string>();

  useEffect(() => {
    apiGet('config').then(({ backend }) => setUsesProxy(backend.proxy.useProxy));
  }, []);

  const hasCard = useSDCard(devices, [code]);

  const onAccountChanged = useCallback((code: string, status: accountApi.IStatus | undefined) => {
    if (!code || status === undefined || status.fatalError) {
      return;
    }
    if (status.synced && status.offlineError === null) {
      const currentCode = code;
      Promise.all([
        accountApi.getBalance(currentCode).then(newBalance => {
          if (currentCode !== code) {
            // Results came in after the account was switched. Ignore.
            return;
          }
          setBalance(newBalance);
        }),
        accountApi.getTransactionList(code).then(newTransactions => {
          if (currentCode !== code) {
            // Results came in after the account was switched. Ignore.
            return;
          }
          setTransactions(newTransactions);
        })
      ])
        .catch(console.error);
    } else {
      setBalance(null);
      setTransactions(undefined);
    }
  }, []);

  const onStatusChanged = useCallback(() => {
    const currentCode = code;
    if (!currentCode) {
      return;
    }
    accountApi.getStatus(currentCode).then(async status => {
      if (currentCode !== code) {
        // Results came in after the account was switched. Ignore.
        return;
      }
      setStatus(status);
      if (!status.disabled && !status.synced) {
        await accountApi.init(currentCode).catch(console.error);
      }
      onAccountChanged(code, status);
    })
      .catch(console.error);
  }, [onAccountChanged, code]);

  const subscriptions = useRef<UnsubscribeList>([]);
  useEffect(() => {
    unsubscribe(subscriptions.current);
    subscriptions.current.push(
      syncAddressesCount(code, (givenCode, addressesSynced) => {
        if (givenCode === code) {
          setSyncedAddressesCount(addressesSynced);
        }
      }),
      statusChanged(code, () => onStatusChanged()),
      syncdone(code, () => onAccountChanged(code, status)),
    );
    const unsubscribeList = subscriptions.current;
    return () => unsubscribe(unsubscribeList);
  }, [code, onAccountChanged, onStatusChanged, status]);

  const moonpayBuySupported = useLoad(isMoonpayBuySupported(code));

  function exportAccount() {
    if (status === undefined || status.fatalError) {
      return;
    }
    accountApi.exportAccount(code)
      .then(result => {
        if (result !== null && !result.success) {
          alertUser(result.errorMessage);
        }
      })
      .catch(console.error);
  }

  useEffect(() => {
    setStateCode(code);
    setBalance(null);
    setStatus(undefined);
    setSyncedAddressesCount(0);
    setTransactions(undefined);
    onStatusChanged();
  }, [code, onStatusChanged]);

  const hasDataLoaded = balance !== null && transactions !== undefined;

  const account = accounts && accounts.find(acct => acct.code === code);
  if (stateCode !== code) {
    // Sync code property with stateCode to work around a re-render that
    // happens briefly before `setStatus(undefined)` stops rendering again below.
    return null;
  }
  if (!account || status === undefined) {
    return null;
  }

  const canSend = balance !== null && balance.hasAvailable;

  const initializingSpinnerText =
    (syncedAddressesCount !== undefined && syncedAddressesCount > 1) ? (
      '\n' + t('account.syncedAddressesCount', {
        count: syncedAddressesCount.toString(),
        defaultValue: 0,
      } as any)
    ) : '';

  const offlineErrorTextLines: string[] = [];
  if (status.offlineError !== null) {
    offlineErrorTextLines.push(t('account.reconnecting'));
    offlineErrorTextLines.push(status.offlineError);
    if (usesProxy) {
      offlineErrorTextLines.push(t('account.maybeProxyError'));
    }
  }

  const showBuyButton = moonpayBuySupported
    && balance !== null
    && !balance.hasAvailable
    && !balance.hasIncoming
    && transactions && transactions.length === 0;

  const actionButtonsProps = {
    code,
    canSend,
    moonpayBuySupported
  };

  return (
    <div className="contentWithGuide">
      <div className="container">
        <Status hidden={!hasCard} type="warning">
          {t('warning.sdcard')}
        </Status>
        <Header
          title={<h2><span>{account.name}</span></h2>}>
          <Link to={`/account/${code}/info`} title={t('accountInfo.title')} className="flex flex-row flex-items-center">
            <Info className={style.accountIcon} />
            <span>{t('accountInfo.label')}</span>
          </Link>
        </Header>
        {status.synced && hasDataLoaded && isBitcoinBased(account.coinCode) && (
          <HeadersSync coinCode={account.coinCode} />
        )}
        <div className="innerContainer scrollableContainer">
          <div className="content padded">
            { showBuyButton && (
              <BuyCTA
                code={code}
                unit={balance.available.unit} />
            )}

            <div className="flex flex-column flex-reverse-mobile">
              <label className="labelXLarge flex-self-start-mobile hide-on-small">
                {t('accountSummary.availableBalance')}
              </label>
              <div className="flex flex-row flex-between flex-item-center flex-column-mobile flex-reverse-mobile">
                <Balance balance={balance} />
                <label className="labelXLarge flex-self-start-mobile show-on-small">
                  {t('accountSummary.availableBalance')}
                </label>
                <ActionButtons {...actionButtonsProps} />
              </div>
            </div>

            { !status.synced || offlineErrorTextLines.length || !hasDataLoaded || status.fatalError ? (
              <Spinner text={
                (status.fatalError && t('account.fatalError'))
                  || offlineErrorTextLines.join('\n')
                  || (!status.synced &&
                      t('account.initializing')
                      + initializingSpinnerText
                  )
                  || ''
              } />
            ) : (
              <Transactions
                accountCode={code}
                handleExport={exportAccount}
                explorerURL={account.blockExplorerTxPrefix}
                transactions={transactions}
              />
            ) }
          </div>
        </div>
      </div>
      <AccountGuide
        account={account}
        unit={balance?.available.unit}
        hasIncomingBalance={balance !== null && balance.hasIncoming}
        hasTransactions={transactions !== undefined && transactions.length > 0}
        hasNoBalance={balance !== null && balance.available.amount === '0'} />
    </div>
  );
}
