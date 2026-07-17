import { useState, type FormEvent } from 'react'

type AuthMode = 'login' | 'register' | 'forgot' | 'reset'

type AuthForm = {
  email: string
  name: string
  password: string
  confirmPassword: string
}

type AuthCopy = {
  registerTitle: string
  signInTitle: string
  authSubtitle: string
  email: string
  name: string
  password: string
  confirmPassword: string
  hidePassword: string
  showPassword: string
  passwordHint: string
  createAccount: string
  signIn: string
  haveAccount: string
  noAccount: string
  forgotPassword: string
  forgotPasswordTitle: string
  resetPasswordTitle: string
  sendResetLink: string
  resetPassword: string
  backToSignIn: string
}

type AuthScreenProps = {
  mode: AuthMode
  form: AuthForm
  error: string
  message: string
  copy: AuthCopy
  onModeChange: (mode: AuthMode) => void
  onChange: (form: AuthForm) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function AuthScreen({
  mode,
  form,
  error,
  message,
  copy,
  onModeChange,
  onChange,
  onSubmit,
}: AuthScreenProps) {
  const isRegister = mode === 'register'
  const isForgot = mode === 'forgot'
  const isReset = mode === 'reset'
  const [showPassword, setShowPassword] = useState(false)

  return (
    <section className="auth-screen">
      <div className="flame-orb" aria-hidden="true">
        <FlameIcon />
      </div>
      <div>
        <h1>{isRegister
          ? copy.registerTitle
          : isForgot
            ? copy.forgotPasswordTitle
            : isReset
              ? copy.resetPasswordTitle
              : copy.signInTitle}</h1>
        <p>{copy.authSubtitle}</p>
      </div>

      <form className="auth-form" onSubmit={onSubmit}>
        {!isReset && <label>
          {copy.email}
          <input
            type="email"
            autoComplete="email"
            required
            value={form.email}
            onChange={(event) => onChange({ ...form, email: event.target.value })}
          />
        </label>}

        {isRegister && (
          <label>
            {copy.name}
            <input
              type="text"
              autoComplete="name"
              value={form.name}
              onChange={(event) => onChange({ ...form, name: event.target.value })}
            />
          </label>
        )}

        {!isForgot && <div className="auth-field">
          <label htmlFor="auth-password">{copy.password}</label>
          <span className="password-field">
            <input
              id="auth-password"
              type={showPassword ? 'text' : 'password'}
              autoComplete={isRegister || isReset ? 'new-password' : 'current-password'}
              minLength={8}
              required
              value={form.password}
              onChange={(event) => onChange({ ...form, password: event.target.value })}
            />
            <button
              type="button"
              aria-label={showPassword ? copy.hidePassword : copy.showPassword}
              onClick={() => setShowPassword((current) => !current)}
            >
              {showPassword ? <EyeOffIcon /> : <EyeIcon />}
            </button>
          </span>
        </div>}

        {(isRegister || isReset) && (
          <label>
            {copy.confirmPassword}
            <span className="password-field">
              <input
                type={showPassword ? 'text' : 'password'}
                autoComplete="new-password"
                minLength={8}
                required
                value={form.confirmPassword}
                onChange={(event) => onChange({ ...form, confirmPassword: event.target.value })}
              />
              <button
                type="button"
                aria-label={showPassword ? copy.hidePassword : copy.showPassword}
                onClick={() => setShowPassword((current) => !current)}
              >
                {showPassword ? <EyeOffIcon /> : <EyeIcon />}
              </button>
            </span>
          </label>
        )}

        {isRegister && <span className="form-hint">{copy.passwordHint}</span>}
        {message && <p className="settings-success">{message}</p>}
        {error && <p className="form-error">{error}</p>}

        {mode === 'login' ? (
          <div className="auth-login-actions">
            <button className="primary-button primary-button--large" type="submit">{copy.signIn}</button>
            <button className="auth-forgot-button" type="button" onClick={() => onModeChange('forgot')}>
              {copy.forgotPassword}
            </button>
          </div>
        ) : (
          <button className="primary-button primary-button--large" type="submit">
            {isRegister
              ? copy.createAccount
              : isForgot
                ? copy.sendResetLink
                : copy.resetPassword}
          </button>
        )}
      </form>

      <button
        className="ghost-button auth-switch"
        type="button"
        onClick={() => onModeChange(isRegister || isForgot || isReset ? 'login' : 'register')}
      >
        {isForgot || isReset
          ? copy.backToSignIn
          : <>{isRegister ? copy.haveAccount : copy.noAccount} {isRegister ? copy.signIn : copy.createAccount}</>}
      </button>
    </section>
  )
}

function FlameIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12.2 3.5c.5 2.7-.6 4.4-2 5.9-1.3 1.4-2.5 2.7-2.5 4.9a4.4 4.4 0 0 0 8.8.1c0-1.8-.9-3.4-2.4-4.8.1 1.4-.4 2.4-1.5 3.1-.4-2.6.8-4.7-.4-9.2Z" />
      <path d="M12 20.8c-4 0-7.1-2.8-7.1-6.8 0-2.7 1.5-4.6 3.1-6.2 1.5-1.5 3.1-3.1 3.2-5.8 4 2.8 6.4 6.4 6.4 10.6 1.1-.9 1.6-2.1 1.6-3.5 1.4 1.5 2 3.1 2 4.8 0 4.1-3.2 6.9-9.2 6.9Z" />
    </svg>
  )
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z" />
      <path d="M12 9.5a2.5 2.5 0 1 1 0 5 2.5 2.5 0 0 1 0-5Z" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 3l18 18" />
      <path d="M10.6 10.6a2.5 2.5 0 0 0 2.8 2.8" />
      <path d="M9.4 5.3A9.7 9.7 0 0 1 12 5c6 0 9.5 7 9.5 7a16 16 0 0 1-3 3.7" />
      <path d="M6.4 6.8C3.9 8.5 2.5 12 2.5 12s3.5 7 9.5 7c1.5 0 2.8-.4 4-1" />
    </svg>
  )
}
