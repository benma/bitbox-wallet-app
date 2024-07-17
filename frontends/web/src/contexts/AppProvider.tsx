/**
 * Copyright 2023-2024 Shift Crypto AG
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

import { ReactNode, useCallback, useEffect, useState } from 'react';
import { getConfig, setConfig } from '@/utils/config';
import { AppContext, TBackHandler } from './AppContext';
import { useLoad } from '@/hooks/api';
import { usePrevious } from '@/hooks/previous';
import { useDefault } from '@/hooks/default';
import { getNativeLocale } from '@/api/nativelocale';
import { i18nextFormat } from '@/i18n/utils';
import type { TChartDisplay, TSidebarStatus } from './AppContext';

type TProps = {
    children: ReactNode;
}

let queue: {
  action: 'pushState' | 'back';
  invoke?: () => void;
}[] = [];
let isProcessing = false;
let ignoreNext = false;

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
  }
  if (item.invoke) {
    item.invoke();
  }

  // Ensure the browser has time to process each navigation
  setTimeout(() => {
    isProcessing = false;
    processQueue();
  }, 100); // Adjust delay based on testing
}

export const AppProvider = ({ children }: TProps) => {
  const nativeLocale = i18nextFormat(useDefault(useLoad(getNativeLocale), 'de-CH'));
  const [guideShown, setGuideShown] = useState(false);
  const [guideExists, setGuideExists] = useState(false);
  const [hideAmounts, setHideAmounts] = useState(false);
  const [activeSidebar, setActiveSidebar] = useState(false);
  const [sidebarStatus, setSidebarStatus] = useState<TSidebarStatus>('');
  const [chartDisplay, setChartDisplay] = useState<TChartDisplay>('all');
  const [backHandlers, setBackHandlers] = useState<TBackHandler[]>([]);

  const toggleGuide = () => {
    setConfig({ frontend: { guideShown: !guideShown } });
    setGuideShown(prev => !prev);
  };

  const toggleHideAmounts = () => {
    setConfig({ frontend: { hideAmounts: !hideAmounts } });
    setHideAmounts(prev => !prev);
  };

  const toggleSidebar = () => {
    setActiveSidebar(prev => !prev);
  };

  useEffect(() => {
    getConfig().then(({ frontend }) => {
      if (frontend) {
        if (frontend.guideShown !== undefined) {
          setGuideShown(frontend.guideShown);
        }
        if (frontend.hideAmounts !== undefined) {
          setHideAmounts(frontend.hideAmounts);
        }
      } else {
        setGuideShown(true);
      }
    });
  }, []);

  const callTopHandler = useCallback(() => {
    console.log('CALL TOP', backHandlers.length);

    if (backHandlers.length > 0) {
      const topHandler = backHandlers[backHandlers.length - 1];
      topHandler();
      return true;
    }
    return false;
  }, [backHandlers]);

  useEffect(() => {
    const handler = () => {
      if (ignoreNext) {
        console.log('IGNORED');
        ignoreNext = false;
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


  const pushBackHandler = useCallback((handler: TBackHandler) => {
    console.log('pushHandler');
    setBackHandlers((prevStack) => [...prevStack, handler]);
    queue.push({ action: 'pushState' });
    processQueue();
  }, []);

  const popBackHandler = useCallback(() => {
    console.log('popHandler');
    setBackHandlers((prevStack) => prevStack.slice(0, -1));
    queue.push({ action: 'back' });
    ignoreNext = true;
    processQueue();
  }, []);

  const previousGuideShown = usePrevious(guideShown);
  useEffect(() => {
    if (guideShown && !previousGuideShown) {
      pushBackHandler(() => setGuideShown(false));
    }
    if (!guideShown && previousGuideShown) {
      popBackHandler();
    }
  }, [pushBackHandler, popBackHandler, guideShown, previousGuideShown, setGuideShown]);

  return (
    <AppContext.Provider
      value={{
        activeSidebar,
        toggleGuide,
        guideShown,
        guideExists,
        hideAmounts,
        nativeLocale,
        sidebarStatus,
        chartDisplay,
        setActiveSidebar,
        setGuideShown,
        setGuideExists,
        setSidebarStatus,
        setHideAmounts,
        setChartDisplay,
        toggleHideAmounts,
        toggleSidebar,
        pushBackHandler,
        popBackHandler,
      }}>
      {children}
    </AppContext.Provider>
  );
};
