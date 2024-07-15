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

import { ReactNode, useContext, createContext, useEffect, useState, useCallback } from 'react';
import { AppContext } from './AppContext';

export type THandler = () => boolean;

type TProps = {
  pushHandler: (handler: THandler) => void;
  popHandler: () => void;
  callTopHandler: () => boolean;
}

export const BackButtonStackContext = createContext<TProps>({
  pushHandler: () => {
    console.log('pushHandler');
    return true;
  },
  popHandler: () => {
    console.log('popHandler');
    return true;
  },
  callTopHandler: () => {
    console.log('callTopHandler');
    return true;
  },
});

type TProviderProps = {
    children: ReactNode;
}

export const BackButtonStackProvider = ({ children }: TProviderProps) => {
  const [stack, setStack] = useState<THandler[]>([]);

  const { guideShown, setGuideShown } = useContext(AppContext);

  const pushHandler = useCallback((handler: THandler) => {
    setStack((prevStack) => [...prevStack, handler]);
  }, []);

  const popHandler = useCallback(() => {
    setStack((prevStack) => prevStack.slice(0, -1));
  }, []);

  const callTopHandler = useCallback(() => {
    // If guide is shown, back button should close that first.
    if (guideShown) {
      setGuideShown(false);
      return false;
    }

    if (stack.length > 0) {
      const topHandler = stack[stack.length - 1];
      return topHandler();
    }
    return true;
  }, [stack, guideShown, setGuideShown]);


  // Install back button callback that is called from Android.
  useEffect(() => {
    window.onBackButtonPressed = callTopHandler;
    return () => {
      delete window.onBackButtonPressed;
    };
  }, [callTopHandler]);

  return (
    <BackButtonStackContext.Provider value={{ pushHandler, popHandler, callTopHandler }}>
      {children}
    </BackButtonStackContext.Provider>
  );
};
