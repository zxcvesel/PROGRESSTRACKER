export type AdminUserSummary = {
  id: number
  email: string
  name: string
  createdAt: string
  emailVerified: boolean
  role: 'user' | 'admin'
  activeSessions: number
  goalCount: number
}

type AdminCopy = {
  adminSubtitle: string
  registeredUsers: string
  verifiedUsers: string
  userAccess: string
  verified: string
  unverified: string
  roleAdmin: string
  roleUser: string
  activeSessions: string
  goalsCount: string
  registeredOn: string
  revokeSessions: string
  revokingSessions: string
  currentAccount: string
  noActiveSessions: string
  adminEmpty: string
  refresh: string
}

type AdminScreenProps = {
  users: AdminUserSummary[]
  currentUserId: number
  isLoading: boolean
  revokingUserId: number | null
  error: string
  copy: AdminCopy
  language: 'en' | 'ru'
  onRefresh: () => void
  onRevokeSessions: (user: AdminUserSummary) => void
}

export function AdminScreen({
  users,
  currentUserId,
  isLoading,
  revokingUserId,
  error,
  copy,
  language,
  onRefresh,
  onRevokeSessions,
}: AdminScreenProps) {
  const verifiedUsers = users.filter((user) => user.emailVerified).length

  return (
    <section className="admin-screen">
      <div className="admin-intro">
        <div>
          <span>SPARX ADMIN</span>
          <h1>{copy.userAccess}</h1>
          <p>{copy.adminSubtitle}</p>
        </div>
        <button className="icon-button admin-refresh" type="button" onClick={onRefresh} aria-label={copy.refresh}>
          <RefreshIcon />
        </button>
      </div>

      <div className="admin-summary" aria-label={copy.userAccess}>
        <div>
          <span>{copy.registeredUsers}</span>
          <strong>{users.length}</strong>
        </div>
        <div>
          <span>{copy.verifiedUsers}</span>
          <strong>{verifiedUsers}</strong>
        </div>
      </div>

      {error && <p className="form-error">{error}</p>}
      {isLoading && users.length === 0 && <p className="empty-message">...</p>}
      {!isLoading && users.length === 0 && <p className="empty-message">{copy.adminEmpty}</p>}

      <div className="admin-user-list">
        {users.map((user) => {
          const isCurrentUser = user.id === currentUserId
          const isRevoking = revokingUserId === user.id
          return (
            <article className="admin-user-card" key={user.id}>
              <div className="admin-user-card__header">
                <span className="admin-user-avatar" aria-hidden="true">
                  {(user.name || user.email).slice(0, 1).toUpperCase()}
                </span>
                <div className="admin-user-identity">
                  <strong>{user.name || user.email}</strong>
                  <span>{user.email}</span>
                </div>
                <span className={`admin-role admin-role--${user.role}`}>
                  {user.role === 'admin' ? copy.roleAdmin : copy.roleUser}
                </span>
              </div>

              <div className="admin-user-meta">
                <p><span>{copy.registeredOn}</span><strong>{formatAdminDate(user.createdAt, language)}</strong></p>
                <p><span>{copy.goalsCount}</span><strong>{user.goalCount}</strong></p>
                <p><span>{copy.activeSessions}</span><strong>{user.activeSessions}</strong></p>
                <p>
                  <span>Email</span>
                  <strong className={user.emailVerified ? 'is-verified' : 'is-unverified'}>
                    {user.emailVerified ? copy.verified : copy.unverified}
                  </strong>
                </p>
              </div>

              <button
                className="ghost-button admin-revoke-button"
                type="button"
                disabled={isCurrentUser || user.activeSessions === 0 || isRevoking}
                onClick={() => onRevokeSessions(user)}
              >
                {isCurrentUser
                  ? copy.currentAccount
                  : isRevoking
                    ? copy.revokingSessions
                    : user.activeSessions === 0
                      ? copy.noActiveSessions
                      : copy.revokeSessions}
              </button>
            </article>
          )
        })}
      </div>
    </section>
  )
}

function formatAdminDate(value: string, language: 'en' | 'ru') {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-RU' : 'en-US', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  }).format(date)
}

function RefreshIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M20 11a8 8 0 1 0-2.3 5.7" />
      <path d="M20 4v7h-7" />
    </svg>
  )
}
