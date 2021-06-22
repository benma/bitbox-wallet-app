/**
 * Copyright 2021 Shift Crypto AG
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

import { AccountCode, CoinCode } from './account';
import { apiGet, apiPost } from '../utils/request';

export interface ICoin {
    coinCode: CoinCode;
    name: string;
    canAddAccount: boolean;
    suggestedAccountName: string;
}

export interface ISuccess {
    success: boolean;
    errorMessage?: string;
    errorCode?: string;
}

export const getSupportedCoins = (): Promise<ICoin[]> => {
    return apiGet('supported-coins');
};

export const setAccountActive = (accountCode: string, active: boolean): Promise<ISuccess> => {
    return apiPost('set-account-active', { accountCode, active });
};

export const setTokenActive = (accountCode: string, tokenCode: string, active: boolean): Promise<ISuccess> => {
    return apiPost('set-token-active', { accountCode, tokenCode, active });
};

export const renameAccount = (accountCode: string, name: string): Promise<ISuccess> => {
    return apiPost('rename-account', { accountCode, name });
};

export const reinitializeAccounts = (): Promise<null> => {
    return apiPost('accounts/reinitialize');
};

export interface AoppAccount {
    name: string;
    code: AccountCode;
}

export interface Aopp {
    // See backend/aopp.go fo ra description of the states.x
    state: '' | 'error' | 'awaiting-keystore' | 'choosing-account' | 'syncing' | 'signing' | 'success';
    accounts: AoppAccount[];
    // See backend/errors.go for a description of the errors.
    errorCode: '' | 'aoppUnsupportedAsset' | 'aoppVersion' | 'aoppInvalidRequest' | 'aoppNoAccounts' | 'aoppCannotSign' | 'aoppUnknown' | 'aoppSigningAborted' | 'aoppCallback';
    address: string;
    callbackHost: string;
}

export const aoppCancel = (): Promise<null> => {
    return apiPost('aopp/cancel');
};

export const aoppChooseAccount = (accountCode: string): Promise<null> => {
    return apiPost('aopp/choose-account', { accountCode });
};
