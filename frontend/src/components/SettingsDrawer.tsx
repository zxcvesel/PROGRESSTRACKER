import { useState, type FormEvent, type ReactNode } from 'react'

type ThemeMode = 'dark' | 'light'
type AccentColor = 'cyan' | 'purple' | 'orange' | 'green'
type FontSize = 'compact' | 'default' | 'large'
type AppLanguage = 'en' | 'ru'

type AppSettings = {
  theme: ThemeMode
  accent: AccentColor
  fontSize: FontSize
  reducedEffects: boolean
  language: AppLanguage
  defaultGoalDays: string
  defaultTargetHours: string
  defaultTargetMinutes: string
  confirmGoalDelete: boolean
  notificationsEnabled: boolean
}

type AuthUser = {
  email: string
  name: string
}

type PasswordForm = {
  currentPassword: string
  newPassword: string
  confirmPassword: string
}

type SettingsCopy = {
  settings: string
  settingsSubtitle: string
  closeSettings: string
  account: string
  signedInAs: string
  displayName: string
  saveProfile: string
  currentPassword: string
  newPassword: string
  confirmPassword: string
  passwordHint: string
  changePassword: string
  logout: string
  deleteAccount: string
  deleteAccountHint: string
  deleteAccountConfirm: string
  deleteAccountAction: string
  exportData: string
  exportJSON: string
  exportCSV: string
  appearance: string
  theme: string
  themeDark: string
  themeLight: string
  accentColor: string
  fontSize: string
  fontCompact: string
  fontDefault: string
  fontLarge: string
  reducedGlow: string
  language: string
  appLanguage: string
  languageNote: string
  goals: string
  defaultDuration: string
  defaultHours: string
  minutes: string
  confirmGoalDeletion: string
  about: string
  productTagline: string
  version: string
  releaseChannel: string
  beta: string
  dataPrivacy: string
  privateAccountData: string
  copyright: string
  manageAccount: string
  notifications: string
  notificationDescription: string
  enableNotifications: string
  disableNotifications: string
  notificationsBlocked: string
  notificationsUnsupported: string
  notificationNote: string
  legal: string
  privacyPolicy: string
  privacyText: string
  termsOfUse: string
  termsText: string
}

type SettingsDrawerProps = {
  isOpen: boolean
  settings: AppSettings
  currentUser: AuthUser | null
  accountName: string
  passwordForm: PasswordForm
  accountMessage: string
  accountError: string
  appVersion: string
  notificationPermission: NotificationPermission | 'unsupported'
  copy: SettingsCopy
  onClose: () => void
  onChange: (settings: Partial<AppSettings>) => void
  onAccountNameChange: (name: string) => void
  onPasswordFormChange: (form: PasswordForm) => void
  onProfileSubmit: (event: FormEvent<HTMLFormElement>) => void
  onPasswordSubmit: (event: FormEvent<HTMLFormElement>) => void
  onLogout: () => void
  onDeleteAccount: (password: string) => Promise<boolean>
  onExport: (format: 'json' | 'csv') => void
  onToggleNotifications: () => void
}

export function SettingsDrawer({
  isOpen,
  settings,
  currentUser,
  accountName,
  passwordForm,
  accountMessage,
  accountError,
  appVersion,
  notificationPermission,
  copy,
  onClose,
  onChange,
  onAccountNameChange,
  onPasswordFormChange,
  onProfileSubmit,
  onPasswordSubmit,
  onLogout,
  onDeleteAccount,
  onExport,
  onToggleNotifications,
}: SettingsDrawerProps) {
  const [deleteAccountOpen, setDeleteAccountOpen] = useState(false)
  const [deletePassword, setDeletePassword] = useState('')

  function closeDrawer() {
    setDeleteAccountOpen(false)
    setDeletePassword('')
    onClose()
  }

  if (!isOpen) {
    return null
  }

  return (
    <div className="drawer-backdrop" onClick={closeDrawer}>
      <aside
        className="settings-drawer"
        aria-label={copy.settings}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="settings-drawer__header">
          <div>
            <p>{copy.settings}</p>
            <span>{copy.settingsSubtitle}</span>
          </div>
          <button className="icon-button icon-button--close" type="button" aria-label={copy.closeSettings} onClick={closeDrawer}>
            <span />
            <span />
          </button>
        </div>

        {currentUser && (
          <SettingsGroup title={copy.account}>
            <div
              className="account-summary"
            >
              <span className="account-summary__avatar" aria-hidden="true">
                {(currentUser.name || currentUser.email).slice(0, 1).toUpperCase()}
              </span>
              <span className="account-summary__identity">
                <strong>{currentUser.name || copy.account}</strong>
                <small>{currentUser.email}</small>
              </span>
            </div>

            <div className="account-management">
              <details className="settings-details">
                <summary>{copy.displayName}</summary>
                <form className="settings-form" onSubmit={onProfileSubmit}>
                  <label>
                    {copy.displayName}
                    <input
                      type="text"
                      value={accountName}
                      onChange={(event) => onAccountNameChange(event.target.value)}
                    />
                  </label>
                  <button className="ghost-button" type="submit">{copy.saveProfile}</button>
                </form>
              </details>

              <details className="settings-details">
                <summary>{copy.changePassword}</summary>
                <form className="settings-form" onSubmit={onPasswordSubmit}>
                  <label>
                    {copy.currentPassword}
                    <input
                      type="password"
                      autoComplete="current-password"
                      required
                      value={passwordForm.currentPassword}
                      onChange={(event) => onPasswordFormChange({ ...passwordForm, currentPassword: event.target.value })}
                    />
                  </label>
                  <label>
                    {copy.newPassword}
                    <input
                      type="password"
                      autoComplete="new-password"
                      minLength={8}
                      required
                      value={passwordForm.newPassword}
                      onChange={(event) => onPasswordFormChange({ ...passwordForm, newPassword: event.target.value })}
                    />
                  </label>
                  <label>
                    {copy.confirmPassword}
                    <input
                      type="password"
                      autoComplete="new-password"
                      minLength={8}
                      required
                      value={passwordForm.confirmPassword}
                      onChange={(event) => onPasswordFormChange({ ...passwordForm, confirmPassword: event.target.value })}
                    />
                  </label>
                  <p className="settings-note">{copy.passwordHint}</p>
                  <button className="ghost-button" type="submit">{copy.changePassword}</button>
                </form>
              </details>

              <details className="settings-details">
                <summary>{copy.exportData}</summary>
                <div className="settings-inline-actions">
                  <button className="ghost-button" type="button" onClick={() => onExport('json')}>{copy.exportJSON}</button>
                  <button className="ghost-button" type="button" onClick={() => onExport('csv')}>{copy.exportCSV}</button>
                </div>
              </details>

              <details
                className="settings-details settings-details--danger"
                open={deleteAccountOpen}
                onToggle={(event) => setDeleteAccountOpen(event.currentTarget.open)}
              >
                <summary>{copy.deleteAccount}</summary>
                <form
                  className="settings-form"
                  onSubmit={async (event) => {
                    event.preventDefault()
                    if (await onDeleteAccount(deletePassword)) {
                      setDeletePassword('')
                      setDeleteAccountOpen(false)
                    }
                  }}
                >
                  <p className="settings-note settings-note--plain">{copy.deleteAccountHint}</p>
                  <label>
                    {copy.deleteAccountConfirm}
                    <input
                      type="password"
                      autoComplete="current-password"
                      required
                      value={deletePassword}
                      onChange={(event) => setDeletePassword(event.target.value)}
                    />
                  </label>
                  <button className="ghost-button danger-button" type="submit">{copy.deleteAccountAction}</button>
                </form>
              </details>

              {accountMessage && <p className="settings-success">{accountMessage}</p>}
              {accountError && <p className="form-error">{accountError}</p>}

              <button className="ghost-button account-logout" type="button" onClick={onLogout}>{copy.logout}</button>
            </div>
          </SettingsGroup>
        )}

        <SettingsGroup title={copy.appearance}>
          <label>
            {copy.theme}
            <select
              value={settings.theme}
              onChange={(event) => onChange({ theme: event.target.value as ThemeMode })}
            >
              <option value="dark">{copy.themeDark}</option>
              <option value="light">{copy.themeLight}</option>
            </select>
          </label>

          <div className="settings-field">
            <span>{copy.accentColor}</span>
            <div className="swatch-grid" role="list" aria-label={copy.accentColor}>
              {(['cyan', 'purple', 'orange', 'green'] as AccentColor[]).map((accent) => (
                <button
                  className={`swatch-button swatch-button--${accent} ${settings.accent === accent ? 'is-selected' : ''}`}
                  type="button"
                  key={accent}
                  aria-label={accent}
                  onClick={() => onChange({ accent })}
                />
              ))}
            </div>
          </div>

          <label>
            {copy.fontSize}
            <select
              value={settings.fontSize}
              onChange={(event) => onChange({ fontSize: event.target.value as FontSize })}
            >
              <option value="compact">{copy.fontCompact}</option>
              <option value="default">{copy.fontDefault}</option>
              <option value="large">{copy.fontLarge}</option>
            </select>
          </label>

          <ToggleRow
            label={copy.reducedGlow}
            checked={settings.reducedEffects}
            onChange={(checked) => onChange({ reducedEffects: checked })}
          />
        </SettingsGroup>

        <SettingsGroup title={copy.language}>
          <label>
            {copy.appLanguage}
            <select
              value={settings.language}
              onChange={(event) => onChange({ language: event.target.value as AppLanguage })}
            >
              <option value="en">English</option>
              <option value="ru">Русский</option>
            </select>
          </label>
          <p className="settings-note">{copy.languageNote}</p>
        </SettingsGroup>

        <SettingsGroup title={copy.goals}>
          <label>
            {copy.defaultDuration}
            <input
              type="number"
              min="1"
              value={settings.defaultGoalDays}
              onChange={(event) => onChange({ defaultGoalDays: event.target.value })}
            />
          </label>

          <div className="form-row form-row--target">
            <label>
              {copy.defaultHours}
              <input
                type="number"
                min="0"
                step="1"
                value={settings.defaultTargetHours}
                onChange={(event) => onChange({ defaultTargetHours: event.target.value })}
              />
            </label>
            <label>
              {copy.minutes}
              <input
                type="number"
                min="0"
                max="59"
                step="5"
                value={settings.defaultTargetMinutes}
                onChange={(event) => onChange({ defaultTargetMinutes: event.target.value })}
              />
            </label>
          </div>

          <ToggleRow
            label={copy.confirmGoalDeletion}
            checked={settings.confirmGoalDelete}
            onChange={(checked) => onChange({ confirmGoalDelete: checked })}
          />
        </SettingsGroup>

        <SettingsGroup title={copy.notifications}>
          <p className="settings-note settings-note--plain">{copy.notificationDescription}</p>
          <button
            className="ghost-button"
            type="button"
            disabled={notificationPermission === 'unsupported' || notificationPermission === 'denied'}
            onClick={onToggleNotifications}
          >
            {notificationPermission === 'unsupported'
              ? copy.notificationsUnsupported
              : notificationPermission === 'denied'
                ? copy.notificationsBlocked
                : settings.notificationsEnabled
                  ? copy.disableNotifications
                  : copy.enableNotifications}
          </button>
          <p className="settings-note">{copy.notificationNote}</p>
        </SettingsGroup>

        <SettingsGroup title={copy.legal}>
          <details className="settings-details legal-details">
            <summary>{copy.privacyPolicy}</summary>
            <p>{copy.privacyText}</p>
          </details>
          <details className="settings-details legal-details">
            <summary>{copy.termsOfUse}</summary>
            <p>{copy.termsText}</p>
          </details>
        </SettingsGroup>

        <SettingsGroup title={copy.about}>
          <div className="about-product">
            <span className="about-product__mark" aria-hidden="true">PT</span>
            <div>
              <strong>Progress Tracker</strong>
              <p>{copy.productTagline}</p>
            </div>
          </div>
          <div className="about-list">
            <p><span>{copy.version}</span><strong>{appVersion}</strong></p>
            <p><span>{copy.releaseChannel}</span><strong>{copy.beta}</strong></p>
            <p><span>{copy.dataPrivacy}</span><strong>{copy.privateAccountData}</strong></p>
          </div>
          <p className="about-copyright">{copy.copyright}</p>
        </SettingsGroup>
      </aside>
    </div>
  )
}

function SettingsGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <details className="settings-group">
      <summary><h2>{title}</h2></summary>
      <div className="settings-group__content">{children}</div>
    </details>
  )
}

function ToggleRow({
  label,
  checked,
  onChange,
}: {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <label className="toggle-row">
      <span>{label}</span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
      />
    </label>
  )
}
