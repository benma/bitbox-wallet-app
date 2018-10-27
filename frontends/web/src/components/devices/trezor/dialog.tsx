/**
 * Copyright 2018 Shift Devices AG
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
import { Dialog } from '../../dialog/dialog';
import { apiGet, apiPost } from '../../../utils/request';
import { apiWebsocket } from '../../../utils/websocket';

interface Props {
}

interface State {
    activeDialog: boolean;

    deviceStatus: string;
    pin: string;
    passphrase: string;
    deviceID: string;
}

class TrezorDialog extends Component<Props, State> {
    unsubscribe: (() => void) | null = null

    state = {
        activeDialog: false,
        deviceStatus: '',
        pin: '',
        passphrase: '',
        deviceID: '',
    }

    public componentDidMount() {
        this.unsubscribe = apiWebsocket(({ type, data, deviceID, productName }) => {
            console.log(productName);
            if (type === 'device' && productName === 'trezor') {
                if (data === 'statusChanged') {
                    this.onDeviceStatusChanged(deviceID);
                }
            }
        });
    }

    public componentWillUnmount() {
        if (this.unsubscribe) {
            this.unsubscribe();
        }
    }

    private onDeviceStatusChanged = (deviceID: string) => {
        apiGet('devices/trezor/' + deviceID + '/status').then(deviceStatus => {
            if (deviceStatus === '' || deviceStatus === 'pinRequired' || deviceStatus === 'passphraseRequired') {
                this.setState({ activeDialog: deviceStatus !== '', deviceID, deviceStatus });
            }
        });
    }

    private submitPIN = (event) => {
        event.preventDefault();
        apiPost('devices/trezor/' + this.state.deviceID + '/pin', this.state.pin);
    }

    private submitPassphrase = (event) => {
        event.preventDefault();
        apiPost('devices/trezor/' + this.state.deviceID + '/passphrase', this.state.passphrase);
    }


    private abort = () => {
        this.setState({
            activeDialog: false,
        });
    }

    private enterDigit = digit => {
        this.setState({ pin: this.state.pin + digit });
    }

    public render({
    }: RenderableProps<Props>, {
        activeDialog,
        deviceStatus,
        pin,
        passphrase,
    }: State) {
        if (!activeDialog) {
            return null;
        }
        return (
            <Dialog
                title={'TREZOR'}
                onClose={this.abort}>
                {
                    deviceStatus === 'pinRequired' && (
                        <span>
                            <form onSubmit={this.submitPIN}>
                                PIN: <input type="password" value={pin} />
                                <br/>
                                <button type="button" onClick={() => this.enterDigit('7')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('8')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('9')}>x</button>
                                <br/>
                                <button type="button" onClick={() => this.enterDigit('4')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('5')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('6')}>x</button>
                                <br/>
                                <button type="button" onClick={() => this.enterDigit('1')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('2')}>x</button>
                                <button type="button" onClick={() => this.enterDigit('3')}>x</button>
                                <br/>
                                <button>submit</button>
                            </form>
                        </span>
                    )
                }
                {
                    deviceStatus === 'passphraseRequired' && (
                        <span>
                            <form onSubmit={this.submitPassphrase}>
                                Passphrase: <input type="password" onChange={e => this.setState({ passphrase: (e.target as HTMLInputElement).value })} value={passphrase} />
                                <button>submit</button>
                            </form>
                        </span>
                    )
                }
            </Dialog>
        );
    }
}

export { TrezorDialog };
