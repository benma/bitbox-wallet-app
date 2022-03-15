/**
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

import { FunctionComponent } from 'react';
import { useTranslation } from 'react-i18next';
import { TProductName } from '../../../../api/devices';
import { Button } from '../../../../components/forms';

type Props = JSX.IntrinsicElements['button'] & {
    device?: TProductName;
    forceVerification: boolean;
}

export const useVerfiyLabel = (device?: TProductName): string => {
    const { t } = useTranslation();
    if (device === 'bitbox') {
        return t('receive.verifyBitBox01');
    } else if (device === 'bitbox02') {
        return t('receive.verifyBitBox02');
    }
    return t('receive.verify'); // fallback
};

export const VerifyButton: FunctionComponent<Props> = ({
    device,
    forceVerification,
    ...props
}) => {
    const { t } = useTranslation();
    const verifyLabel = useVerfiyLabel(device);
    return (
        <Button primary {...props}>
            { forceVerification ? t('receive.showFull') : verifyLabel }
        </Button>
    );
};
