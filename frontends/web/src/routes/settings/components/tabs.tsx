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

import { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { NavLink, useNavigate } from 'react-router-dom';
import { SettingsItem } from './settingsItem/settingsItem';
import { ChevronRightDark } from '../../../components/icon';
import styles from './tabs.module.css';

type TWithSettingsTabsProps = {
  children: ReactNode;
  deviceIDs: string[];
  hasAccounts: boolean;
  hideMobileMenu?: boolean;
}

type TTab = {
  name: string;
  url: string;
  hideMobileMenu?: boolean;
}

type TTabs = {
  deviceIDs: string[];
  hasAccounts: boolean;
  hideMobileMenu?: boolean;
}

export const WithSettingsTabs = ({
  children,
  deviceIDs,
  hideMobileMenu,
  hasAccounts,
}: TWithSettingsTabsProps) => {
  return (
    <>
      <div className="hide-on-small">
        <Tabs
          hideMobileMenu={hideMobileMenu}
          deviceIDs={deviceIDs}
          hasAccounts={hasAccounts}
        />
      </div>
      {children}
    </>
  );
};

export const Tab = ({ name, url, hideMobileMenu }: TTab) => {
  const navigate = useNavigate();

  if (!hideMobileMenu) {
    // Will only be shown on mobile (index/general settings page)
    return (
      <div key={url} className="show-on-small">
        <SettingsItem
          settingName={name}
          onClick={() => navigate(url)}
          extraComponent={<ChevronRightDark/>} />
      </div>
    );
  }

  return (
    <NavLink
      key={url}
      className={({ isActive }) => isActive ? `${styles.active} hide-on-small` : 'hide-on-small'}
      to={url}
    >
      {name}
    </NavLink>
  );
};

export const Tabs = ({ deviceIDs, hideMobileMenu, hasAccounts }: TTabs) => {
  const { t } = useTranslation();
  return (
    <div className={styles.container}>
      <Tab
        key="appearance"
        hideMobileMenu={hideMobileMenu}
        name={t('settings.appearance')}
        url="/settings/appearance"
      />
      {hasAccounts ? (
        <Tab
          key="manage-accounts"
          hideMobileMenu={hideMobileMenu}
          name={t('manageAccounts.title')}
          url="/settings/manage-accounts"
        />
      ) : null}
      {deviceIDs.map(id => (
        <Tab
          key={`device-${id}`}
          hideMobileMenu={hideMobileMenu}
          name={t('sidebar.device')}
          url={`/settings/device-settings/${id}`}
        />
      )) }
      <Tab
        key="advanced-settings"
        hideMobileMenu={hideMobileMenu}
        name={t('settings.advancedSettings')}
        url="/settings/advanced-settings"
      />
      <Tab
        key="about"
        hideMobileMenu={hideMobileMenu}
        name={t('settings.about')}
        url="/settings/about"
      />
    </div>
  );
};
