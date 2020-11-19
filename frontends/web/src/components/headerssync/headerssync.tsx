
/**
 * Copyright 2020 Shift Crypto AG
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
import { subscribe } from '../../decorators/subscribe';
import { translate, TranslateProps } from '../../decorators/translate';
import { CoinCode } from '../../routes/account/account';
import Spinner from '../spinner/ascii';
import * as style from './headerssync.css';

interface HeadersSyncProps {
    coinCode: CoinCode;
}

interface StatusInterface {
    targetHeight: number;
    tip: number;
    tipAtInitTime: number;
    tipHashHex: string;
}

interface SubscribedHeadersSyncProps {
    status?: StatusInterface;
}

interface State {
    show: number;
}

type Props = SubscribedHeadersSyncProps & HeadersSyncProps & TranslateProps;

class HeadersSync extends Component<Props, State> {
    public readonly state: State = {
        show: 0,
    }

    componentDidUpdate(prevProps) {
        const { status } = this.props;
        if (status && prevProps.status && status.tip !== prevProps.status.tip) {
            this.setState({ show: status.tip });
            if (status.tip === status.targetHeight) {
                setTimeout(() => this.setState(state => state.show === status.tip ? { show: 0 } : null), 4000);
            }
        }
    }

    render({
        t,
        status,
    }: RenderableProps<Props>, {
        show,
    }: State) {
        if (!status || !show) {
            return null;
        }
        const total = status.targetHeight - status.tipAtInitTime;
        const value = 100 * (status.tip - status.tipAtInitTime) / total;
        const loaded = !total || value >= 100;
        let formatted = status.tip.toString();
        let position = formatted.length - 3;
        while (position > 0) {
            formatted = formatted.slice(0, position) + "'" + formatted.slice(position);
            position = position - 3;
        }

        return (
            <div class={style.syncContainer}>
                <div class={style.syncMessage}>
                    <div class={style.syncText}>
                        {t('headerssync.blocksSynced', { blocks: formatted })} { !loaded && `(${Math.ceil(value)}%)` }
                    </div>
                    { !loaded ? (<Spinner />) : null }
                </div>
                <div class={style.progressBar}>
                    <div class={style.progressValue} style={{ width: `${value}%` }}></div>
                </div>
            </div>
        );
    }
}

const subscribeHOC = subscribe<SubscribedHeadersSyncProps, HeadersSyncProps & TranslateProps>(({ coinCode }) => ({
    status: `coins/${coinCode}/headers/status`,
}), false, true)(HeadersSync);

const HOC = translate<HeadersSyncProps>()(subscribeHOC);
export { HOC as HeadersSync };
