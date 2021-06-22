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

import { Component, h, RenderableProps } from 'preact';
import { AccountCode } from '../../api/account';
import * as backendAPI from '../../api/backend';
import { subscribe } from '../../decorators/subscribe';
import { translate, TranslateProps } from '../../decorators/translate';
import { Button, Select } from '../forms';

interface State {
}

interface AoppProps {
}

interface SubscribedProps {
    aopp?: backendAPI.Aopp;
}

type Props = SubscribedProps & AoppProps & TranslateProps;

class Aopp extends Component<Props, State> {
    public readonly state: State = {
        aopp: {
            state: '',
            accounts: [],
            errorCode: '',
            address: '',
        },
    };

    private chooseAccount = (code: AccountCode) => {
        backendAPI.aoppChooseAccount(code);
    }

    public render(
        { t, aopp }: RenderableProps<Props>,
    ) {
        if (!aopp) {
            return null;
        }
        switch (aopp.state) {
            case '':
                // Inactive, waiting for action.
                return null;
            case 'error':
                return (
                    <div>
                        <p>{ t('aopp.addressRequested', { host: aopp.callbackHost }) }</p>
                        <p>{ t(`error.${aopp.errorCode}`, { host: aopp.callbackHost }) }</p>
                        <Button primary onclick={backendAPI.aoppCancel}>Dismiss</Button>
                    </div>
                );
            case 'awaiting-keystore':
                return (
                    <div>
                        <p>{ t('aopp.addressRequested', { host: aopp.callbackHost }) }</p>
                        <p>{ t('aopp.awaitingKeystore') }</p>
                        <Button primary onclick={backendAPI.aoppCancel}>Cancel</Button>
                    </div>
                );
            case 'choosing-account': {
                const options = aopp.accounts.map(account => {
                    return {
                        text: account.name,
                        value: account.code,
                    };
                });
                return (
                    <div>
                        <Select
                            options={[{
                                text: t('buy.info.selectLabel'),
                                disabled: true,
                                value: 'choose',
                            }, ...options]
                            }
                            defaultValue={'choose'}
                            onChange={e => this.chooseAccount(e.target.value)}
                            id="account"
                        />
                        <Button primary onclick={backendAPI.aoppCancel}>Cancel</Button>
                    </div>
                );
            }
            case 'syncing':
                return <div>Syncing the account, please stand by.</div>;
            case 'signing':
                return (
                    <div>
                        <p>Address: {aopp.address}</p>
                        <p>Please confirm on your BitBox.</p>
                    </div>
                );
            case 'success':
                return (
                    <div>
                        <p>Successfully delivered a fresh address to the third party.</p>
                        <Button primary onclick={backendAPI.aoppCancel}>Dismiss</Button>
                    </div>
                );
        }
    }
}

const subscribeHOC = subscribe<SubscribedProps, AoppProps & TranslateProps>(
    { aopp: 'aopp' },
    false,
    false,
)(Aopp);

const translateHOC = translate<AoppProps>()(subscribeHOC);
export { translateHOC as Aopp };
