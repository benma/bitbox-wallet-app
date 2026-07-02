// SPDX-License-Identifier: Apache-2.0

import { alertUser } from '@/components/alert/Alert';
import { runSQLiteDemo } from '@/api/backend';
import { SettingsItem } from '@/routes/settings/components/settingsItem/settingsItem';

export const SQLiteDemoSetting = () => {
  const handleClick = async () => {
    try {
      const result = await runSQLiteDemo();
      if (!result.success) {
        alertUser(`SQLCipher demo failed: ${result.errorMessage || 'unknown error'}`);
        return;
      }
      alertUser(
        `SQLCipher demo worked.\n\nCipher: ${result.cipherVersion}\nRows: ${
          result.rows.map(row => `${row.id}: ${row.title}`).join(', ')
        }`
      );
    } catch (err) {
      console.error(err);
      alertUser(`SQLCipher demo failed: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  return (
    <SettingsItem
      settingName="Run SQLCipher demo"
      secondaryText="Creates, writes, closes, reopens and reads an encrypted SQLite database."
      onClick={handleClick}
    />
  );
};
