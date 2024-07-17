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
import { usePrevious } from '@/hooks/previous';
import { runningOnMobile } from '@/utils/env';
import { AppContext } from './AppContext';

export type THandler = () => void;

type TProps = {
  pushHandler: (handler: THandler) => void;
  popHandler: () => void;
}

export const BackButtonContext = createContext<TProps>({
  pushHandler: () => {
    console.error('pushHandler called out of context');
    return true;
  },
  popHandler: () => {
    console.error('popHandler called out of context');
    return true;
  },
});

let queue: {
  action: 'pushState' | 'back';
}[] = [];
let isProcessing = false;
let ignoreNext = 0;

function processQueue() {
  if (isProcessing || queue.length === 0) {
    return;
  }
  isProcessing = true;

  const item = queue.shift();
  if (!item) {
    return;
  }
  if (item.action === 'pushState') {
    console.log('QUEUE PUSH');
    window.history.pushState('BLOCKED', '', window.location.href);
  } else {
    console.log('QUEUE POP');
    window.history.back();
    setTimeout(() => console.log('lol', window.history.state), 500);
  }

  // Ensure the browser has time to process each navigation
  setTimeout(() => {
    isProcessing = false;
    processQueue();
  }, 1000); // Adjust delay based on testing
}

type TProviderProps = {
  children: ReactNode;
}

export const BackButtonProvider = ({ children }: TProviderProps) => {
  const [handlers, sethandlers] = useState<THandler[]>([]);
  const { guideShown, setGuideShown } = useContext(AppContext);
  const previousGuideShown = usePrevious(guideShown);

  const callTopHandler = useCallback(() => {
    console.log('CALL TOP', handlers.length);

    if (handlers.length > 0) {
      const topHandler = handlers[handlers.length - 1];
      topHandler();
      return true;
    }
    return false;
  }, [handlers]);

  useEffect(() => {
    const handler = () => {
      if (ignoreNext > 0) {
        console.log('IGNORED', ignoreNext);
        ignoreNext--;
        return;
      }

      if (callTopHandler()) {
        queue.push({ action: 'pushState' });
        processQueue();
      }
    };
    window.addEventListener('popstate', handler);
    return () => {
      window.removeEventListener('popstate', handler);
    };
  }, [callTopHandler]);

  const pushHandler = useCallback((handler: THandler) => {
    console.log('pushHandler');
    sethandlers((prevStack) => [...prevStack, handler]);
    queue.push({ action: 'pushState' });
    processQueue();
  }, []);

  const popHandler = useCallback(() => {
    console.log('popHandler');
    sethandlers((prevStack) => prevStack.slice(0, -1));
    queue.push({ action: 'back' });
    ignoreNext++;
    processQueue();
  }, []);

  // On mobile, the guide covers the whole screen.
  // Make the back button remove the guide first.
  // On desktop the guide does not cover everything and one can keep navigating while it is visible.
  useEffect(() => {
    if (!runningOnMobile) {
      return;
    }
    if (guideShown && !previousGuideShown) {
      pushHandler(() => setGuideShown(false));
    }
    if (!guideShown && previousGuideShown) {
      popHandler();
    }
  }, [pushHandler, popHandler, guideShown, previousGuideShown, setGuideShown]);

  return (
    <BackButtonContext.Provider value={{ pushHandler, popHandler }}>
      {children}
    </BackButtonContext.Provider>
  );
};
