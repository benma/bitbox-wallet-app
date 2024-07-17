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

import { useContext, useEffect, useRef } from 'react';
import { BackButtonContext, THandler } from '@/contexts/BackButtonContext';

export const useBackButton = (handler: THandler) => {
  const { pushHandler, popHandler } = useContext(BackButtonContext);

  // We don't want to re-trigger the handler effect below when the handler changes, no need to
  // repeat the push/pop pair unnecessarily.
  const handlerRef = useRef<(() => void)>(handler);
  useEffect(() => {
    handlerRef.current = handler;
  }, [handler]);


  useEffect(() => {
    pushHandler(handlerRef.current);
    return popHandler;
  }, [handlerRef, pushHandler, popHandler]);
};

// A convenience component that makes sure useBackButton is only used when the component is rendered.
// This avoids complicated useEffect() uses to make sure useBackButton is only active depending on
// rendering conditions.
export const UseBackButton = ({ handler }: { handler: THandler }) => {
  useBackButton(handler);
  return null;
};

// import { useCallback, useEffect, useRef } from 'react';
//
// let inUse = false;
//
// let queue: { action: 'pushState' | 'back'; invoke?: () => void; }[] = [];
// let isProcessing = false;
//
// function processQueue() {
//   if (isProcessing || queue.length === 0) {
//     return;
//   }
//   isProcessing = true;
//
//   const item = queue.shift();
//   if (!item) {
//     return;
//   }
//   if (item.action === 'pushState') {
//     console.log('QUEUE PUSH');
//     window.history.pushState('BLOCKED', '', window.location.href);
//   } else {
//     console.log('QUEUE POP');
//     window.history.back();
//   }
//   if (item.invoke) {
//     item.invoke();
//   }
//
//   // Ensure the browser has time to process each navigation
//   setTimeout(() => {
//     isProcessing = false;
//     processQueue();
//   }, 100); // Adjust delay based on testing
// }
//
//
// export const useDisableBackButton = (disable: boolean, customAction?: () => void, foo?: () => void): void => {
//   const customActionRef = useRef<undefined | (() => void)>(customAction);
//
//   useEffect(() => {
//     customActionRef.current = customAction;
//   }, [customAction]);
//
//   const fooRef = useRef<undefined | (() => void)>(foo);
//
//   useEffect(() => {
//     fooRef.current = foo;
//   }, [foo]);
//
//   const cleanupRef = useRef<null | (() => void)>(null);
//
//   const blockedState = 'BLOCKED';
//
//   const handler = useCallback((event: PopStateEvent) => {
//     event.preventDefault();
//     if (event.state === blockedState) {
//       // Nested event handlers present.
//       console.error('useDisableBackButton 1: nested hooks, but there should only be one at a time.');
//     } else {
//       console.info('useDisableBackButton: blocked back button.');
//       queue.push({ action: 'pushState' });
//       processQueue();
//       console.info('useDisableBackButton: PUSHED.');
//       if (customActionRef.current) {
//         customActionRef.current();
//       }
//     }
//   }, [customActionRef, fooRef]);
//
//   useEffect(() => {
//     const cleanup = () => {
//       if (cleanupRef.current) {
//         inUse = false;
//         cleanupRef.current();
//         cleanupRef.current = null;
//       }
//     };
//     if (disable) {
//
//       if (inUse) {
//         // Figuring out how to have multiple of these hooks work in a stack seems very hard to
//         // figure out, especially in combination with the custom action. For simplicity, we simply
//         // allow only one active back button hook at one time.
//
//         console.error('ERROR: multiple useDisableBackButton effects in use. This one will have no effect.');
//         return;
//       }
//       inUse = true;
//
//       cleanupRef.current = () => {
//         cleanupRef.current = null;
//         console.info('useDisableBackButton: STOP');
//         window.removeEventListener('popstate', handler);
//         // In theory this should always be true b/c we always pushed the state before.
//         // However, history push/pop are not synchronous, so if a push happens immediatelly followed
//         // by this cleanup, the previous push might have not been completed and aborted by
//         // if (window.history.state === blockedState) {
//         // window.history.replaceState(null, '', window.location.href);
//         //setTimeout(() => window.history.back(), 0);
//         queue.push({ action: 'back', invoke: fooRef.current });
//         processQueue();
//
//         //}
//       };
//
//       console.info('useDisableBackButton: START');
//       //if (window.history.state !== blockedState) {
//         //window.history.pushState(blockedState, '', window.location.href);
//       queue.push({ action: 'pushState' });
//       // } else {
//       //   // We are already blocking, something went wrong.
//       //   console.error('useDisableBackButton 2: nested hooks, but there should only be one at a time.');
//       // }
//       window.addEventListener('popstate', handler);
//     } else {
//       cleanup();
//     }
//     return cleanup;
//   }, [handler, disable]);
// };
