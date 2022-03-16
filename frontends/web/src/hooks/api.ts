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

import { useEffect, useState } from 'react'
import { SubscriptionCallback } from '../api/subscribe';
import { Unsubscribe } from '../utils/event';
import { useMountedRef } from './utils';

/**
 * useSubscribe is a hook to subscribe to a subscription function.
 * starts on first render, and returns undefined while there is no first response.
 * re-renders on every update.
 */
export const useSubscribe = <T>(
    subscription: ((callback: SubscriptionCallback<T>) => Unsubscribe)
): (T | undefined) => {
    const [respose, setResponse] = useState<T>();
    const mounted = useMountedRef();
    useEffect(
        () => (
            subscription((data) => {
                if (mounted.current) {
                    setResponse(data);
                }
            })
        ), // we pass no dependencies because it's only suscribed once
        [] // eslint-disable-line react-hooks/exhaustive-deps
    );
    return respose;
}

/**
 * useLoad is a hook to load a promise.
 * gets fired on first render, and returns undefined while loading.
 * if 'apiCall` is `null`, the default state is returned.
 */
export const useLoad = <T>(
    apiCall: (() => Promise<T>) | null
): (T | undefined) => {
    const [respose, setResponse] = useState<T>();
    const mounted = useMountedRef();
    useEffect(
        () => {
            if (apiCall !== null) {
                apiCall().then((data) => {
                    if (mounted.current) {
                        setResponse(data);
                    }
                });
            }
        }, // we pass no dependencies because it's only queried once
        [] // eslint-disable-line react-hooks/exhaustive-deps
    );
    return respose;
}

/**
 * useSync is a hook to load a promise and sync to a subscription function.
 * It is a combination of useLoad and useSubscribe.
 * gets fired on first render, and returns undefined while loading,
 * re-renders on every update.
 */
export const useSync = <T>(
    apiCall: () => Promise<T>,
    subscription: ((callback: SubscriptionCallback<T>) => Unsubscribe),
): (T | undefined) => {
    const [respose, setResponse] = useState<T>();
    const mounted = useMountedRef();
    const onData = (data: T) => {
        if (mounted.current) {
            setResponse(data);
        }
    };
    useEffect(
        () => {
            apiCall().then(onData);
            return subscription(onData);
        }, // we pass no dependencies because it's only queried once
    []); // eslint-disable-line react-hooks/exhaustive-deps
    return respose;
}
