/**
 * Copyright 2024 Shift Crypto AG
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

import { useContext, useEffect } from 'react';
import { BackButtonStackContext, THandler } from '@/contexts/BackButtonStackContext';

/*
 * Installs a handler that is called when the Android back button is pressed.
 * If the handler returns true, the default behavior is invoked: go back in history if possible,
 * otherwise prompt the user if they want to quit the app. If the handler returns false, the default
 * behavior is prevented.
 *
 * Example:
 * useBackButton(useCallback(() => {
 *   // Do something when the back button is pressed in Android.
 *   return false; // prevent default behavior
 * }));
 */
export const useBackButton = (handler: THandler): void => {
  const { pushHandler, popHandler } = useContext(BackButtonStackContext);
  useEffect(() => {
    pushHandler(handler);
    return popHandler;
  }, [handler, pushHandler, popHandler]);
};
