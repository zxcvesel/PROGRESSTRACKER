import type { FormEvent, ReactNode } from 'react'

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
  product: string
  focus: string
  goalBasedLearning: string
  stack: string
}

type SettingsDrawerProps = {
  isOpen: boolean
  settings: AppSettings
  currentUser: AuthUser | null
  accountName: string
  passwordForm: PasswordForm
  accountMessage: string
  accountError: string
  copy: SettingsCopy
  onClose: () => void
  onChange: (settings: Partial<AppSettings>) => void
  onAccountNameChange: (name: string) => void
  onPasswordFormChange: (form: PasswordForm) => void
  onProfileSubmit: (event: FormEvent<HTMLFormElement>) => void
  onPasswordSubmit: (event: FormEvent<HTMLFormElement>) => void
  onLogout: () => void
}

export function SettingsDrawer({
  isOpen,
  settings,
  currentUser,
  accountName,
  passwordForm,
  accountMessage,
  accountError,
  copy,
  onClose,
  onChange,
  onAccountNameChange,
  onPasswordFormChange,
  onProfileSubmit,
  onPasswordSubmit,
  onLogout,
}: SettingsDrawerProps) {
  if (!isOpen) {
    return null
  }

  return (
    <div className="drawer-backdrop" onClick={onClose}>
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
          <button className="icon-button icon-button--close" type="button" aria-label={copy.closeSettings} onClick={onClose}>
            <span />
            <span />
          </button>
        </div>

        {currentUser && (
          <SettingsGroup title={copy.account}>
            <div className="about-list">
              <p><span>{copy.signedInAs}</span><strong>{currentUser.email}</strong></p>
            </div>

            <form className="settings-form" onSubmit={onProfileSubmit}>
              <label>
                {copy.displayName}
                <input
                  type="text"
                  value={accountName}
                  onChange={(event) => onAccountNameChange(event.target.value)}
                />
              </label>
              <button className="ghost-button" type="submit">
                {copy.saveProfile}
              </button>
            </form>

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
              <button className="ghost-button" type="submit">
                {copy.changePassword}
              </button>
            </form>

            {accountMessage && <p className="settings-success">{accountMessage}</p>}
            {accountError && <p className="form-error">{accountError}</p>}

            <button className="ghost-button" type="button" onClick={onLogout}>
              {copy.logout}
            </button>
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

        <SettingsGroup title={copy.about}>
          <div className="about-list">
            <p><span>{copy.product}</span><strong>Progress Tracker</strong></p>
            <p><span>{copy.focus}</span><strong>{copy.goalBasedLearning}</strong></p>
            <p><span>{copy.stack}</span><strong>Go, SQLite, React, TypeScript</strong></p>
          </div>
        </SettingsGroup>
      </aside>
    </div>
  )
}

function SettingsGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="settings-group">
      <h2>{title}</h2>
      {children}
    </section>
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
