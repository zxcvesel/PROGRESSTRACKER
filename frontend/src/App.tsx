import { useEffect, useMemo, useState, type CSSProperties, type FormEvent } from 'react'
import './App.css'

type View = 'goals' | 'create' | 'stats'
type SessionState = 'idle' | 'running' | 'paused'

type GoalSummary = {
  id: number
  title: string
  description: string
  totalDays: number
  dailyTargetMinutes: number
  activeWeekdays: number[]
  startDate: string
  createdAt: string
  status: 'active' | 'completed'
  currentStreak: number
  todayMinutes: number
  todayProgressPct: number
  currentDay: number
  totalProgressPct: number
  totalPracticeMinutes: number
}

type Session = {
  id: number
  goalId: number
  startedAt: string
  endedAt: string
  durationMinutes: number
  notes: string
  tags: string[]
  createdAt: string
}

type GoalDetail = GoalSummary & {
  todayRemainingMinutes: number
  recentSessions: Session[]
}

type WeeklyStat = {
  date: string
  label: string
  minutes: number
  targetMinutes: number
  isCompleted: boolean
}

type GoalDistribution = {
  goalId: number
  title: string
  minutes: number
  percent: number
}

type AppStats = {
  totalSessions: number
  totalPracticeMinutes: number
  currentStreak: number
  longestStreak: number
  todayMinutes: number
  dailyTargetMinutes: number
  weekly: WeeklyStat[]
  monthlyTotalMinutes: number
  goalDistribution: GoalDistribution[]
}

type GoalForm = {
  title: string
  description: string
  totalDays: string
  dailyTargetHours: string
}

const defaultStats: AppStats = {
  totalSessions: 0,
  totalPracticeMinutes: 0,
  currentStreak: 0,
  longestStreak: 0,
  todayMinutes: 0,
  dailyTargetMinutes: 0,
  weekly: [],
  monthlyTotalMinutes: 0,
  goalDistribution: [],
}

const markerColors = ['#19f7e8', '#ff7a3d', '#e6d37a', '#b45cff', '#58d8ff']

function App() {
  const [backendStatus, setBackendStatus] = useState('checking')
  const [view, setView] = useState<View>('goals')
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoalId, setSelectedGoalId] = useState<number | null>(null)
  const [goalDetail, setGoalDetail] = useState<GoalDetail | null>(null)
  const [stats, setStats] = useState<AppStats>(defaultStats)
  const [isLoading, setIsLoading] = useState(true)
  const [formError, setFormError] = useState('')

  const [goalForm, setGoalForm] = useState<GoalForm>({
    title: '',
    description: '',
    totalDays: '90',
    dailyTargetHours: '2',
  })

  const [sessionState, setSessionState] = useState<SessionState>('idle')
  const [sessionStartedAt, setSessionStartedAt] = useState('')
  const [elapsedSeconds, setElapsedSeconds] = useState(0)
  const [finishModalOpen, setFinishModalOpen] = useState(false)
  const [sessionNotes, setSessionNotes] = useState('')
  const [sessionTags, setSessionTags] = useState('')

  useEffect(() => {
    loadInitialData()
  }, [])

  useEffect(() => {
    if (sessionState !== 'running') {
      return
    }

    const timer = window.setInterval(() => {
      setElapsedSeconds((current) => current + 1)
    }, 1000)

    return () => window.clearInterval(timer)
  }, [sessionState])

  const screenTitle = useMemo(() => {
    if (view === 'create') {
      return 'Новая цель'
    }

    if (view === 'stats') {
      return 'Статистика'
    }

    if (selectedGoalId) {
      return 'Цель'
    }

    return 'Цели'
  }, [selectedGoalId, view])

  async function loadInitialData() {
    setIsLoading(true)
    await Promise.all([checkBackend(), loadGoals(), loadStats()])
    setIsLoading(false)
  }

  async function checkBackend() {
    try {
      const response = await fetch('/api/health')
      const data = (await response.json()) as { status: string }
      setBackendStatus(data.status === 'ok' ? 'connected' : 'error')
    } catch {
      setBackendStatus('error')
    }
  }

  async function loadGoals() {
    try {
      const response = await fetch('/api/goals')
      const data = (await response.json()) as GoalSummary[]
      setGoals(data)
    } catch {
      setGoals([])
    }
  }

  async function loadStats() {
    try {
      const response = await fetch('/api/stats')
      const data = (await response.json()) as AppStats
      setStats(data)
    } catch {
      setStats(defaultStats)
    }
  }

  async function openGoal(goalId: number) {
    setSelectedGoalId(goalId)
    setView('goals')
    await loadGoalDetail(goalId)
  }

  async function loadGoalDetail(goalId: number) {
    const response = await fetch(`/api/goals/${goalId}`)
    const data = (await response.json()) as GoalDetail
    setGoalDetail(data)
  }

  async function handleCreateGoal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError('')

    try {
      const response = await fetch('/api/goals', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: goalForm.title,
          description: goalForm.description,
          totalDays: Number(goalForm.totalDays),
          dailyTargetMinutes: Math.round(Number(goalForm.dailyTargetHours) * 60),
        }),
      })

      if (!response.ok) {
        throw new Error('Failed to create goal')
      }

      const createdGoal = (await response.json()) as GoalSummary
      setGoals((currentGoals) => [createdGoal, ...currentGoals])
      setGoalForm({
        title: '',
        description: '',
        totalDays: '90',
        dailyTargetHours: '2',
      })
      setView('goals')
      await openGoal(createdGoal.id)
      await loadStats()
    } catch {
      setFormError('Не удалось создать цель')
    }
  }

  function startSession() {
    setSessionStartedAt(new Date().toISOString())
    setElapsedSeconds(0)
    setSessionState('running')
  }

  function pauseSession() {
    setSessionState('paused')
  }

  function resumeSession() {
    setSessionState('running')
  }

  function finishSession() {
    setSessionState('paused')
    setFinishModalOpen(true)
  }

  async function saveSession() {
    if (!goalDetail) {
      return
    }

    const response = await fetch(`/api/goals/${goalDetail.id}/sessions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        startedAt: sessionStartedAt || new Date().toISOString(),
        endedAt: new Date().toISOString(),
        durationMinutes: Math.max(1, Math.ceil(elapsedSeconds / 60)),
        notes: sessionNotes,
        tags: sessionTags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      }),
    })

    if (!response.ok) {
      setFormError('Не удалось сохранить сессию')
      return
    }

    resetSession()
    await Promise.all([loadGoals(), loadStats(), loadGoalDetail(goalDetail.id)])
  }

  function discardSession() {
    resetSession()
  }

  function resetSession() {
    setSessionState('idle')
    setElapsedSeconds(0)
    setSessionStartedAt('')
    setFinishModalOpen(false)
    setSessionNotes('')
    setSessionTags('')
    setFormError('')
  }

  return (
    <main className="page-shell">
      <section className="phone-shell" aria-label="Progress Tracker">
        <div className="screen-content">
          <header className="top-bar">
            <button
              className="icon-button"
              type="button"
              aria-label={selectedGoalId ? 'Назад к целям' : 'Открыть меню'}
              onClick={() => {
                if (selectedGoalId) {
                  setSelectedGoalId(null)
                  setGoalDetail(null)
                  resetSession()
                }
              }}
            >
              <span />
              <span />
            </button>
            <p>{screenTitle}</p>
            <span
              className={`connection-dot connection-dot--${backendStatus}`}
              title={backendStatus}
            />
          </header>

          {view === 'goals' && !selectedGoalId && (
            <GoalsScreen
              goals={goals}
              isLoading={isLoading}
              onCreate={() => setView('create')}
              onOpenGoal={openGoal}
            />
          )}

          {view === 'goals' && selectedGoalId && goalDetail && (
            <GoalDetailsScreen
              goal={goalDetail}
              elapsedSeconds={elapsedSeconds}
              sessionState={sessionState}
              onStart={startSession}
              onPause={pauseSession}
              onResume={resumeSession}
              onFinish={finishSession}
            />
          )}

          {view === 'create' && (
            <CreateGoalScreen
              form={goalForm}
              formError={formError}
              onChange={setGoalForm}
              onSubmit={handleCreateGoal}
            />
          )}

          {view === 'stats' && <StatsScreen stats={stats} />}
        </div>

        <nav className="bottom-nav" aria-label="Основная навигация">
          <button
            className={view === 'goals' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setView('goals')
              setSelectedGoalId(null)
              setGoalDetail(null)
            }}
            aria-label="Цели"
          >
            <HomeIcon />
          </button>
          <button
            className={view === 'create' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setView('create')
              setSelectedGoalId(null)
              setGoalDetail(null)
            }}
            aria-label="Создать цель"
          >
            <PlusIcon />
          </button>
          <button
            className={view === 'stats' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setView('stats')
              setSelectedGoalId(null)
              setGoalDetail(null)
            }}
            aria-label="Статистика"
          >
            <ChartIcon />
          </button>
        </nav>

        {finishModalOpen && goalDetail && (
          <FinishSessionModal
            goal={goalDetail}
            elapsedSeconds={elapsedSeconds}
            notes={sessionNotes}
            tags={sessionTags}
            formError={formError}
            onNotesChange={setSessionNotes}
            onTagsChange={setSessionTags}
            onSave={saveSession}
            onDiscard={discardSession}
          />
        )}
      </section>
    </main>
  )
}

function GoalsScreen({
  goals,
  isLoading,
  onCreate,
  onOpenGoal,
}: {
  goals: GoalSummary[]
  isLoading: boolean
  onCreate: () => void
  onOpenGoal: (goalId: number) => void
}) {
  if (isLoading) {
    return <p className="empty-message">Загружаем цели...</p>
  }

  if (goals.length === 0) {
    return (
      <section className="empty-state">
        <div className="flame-orb" aria-hidden="true">
          <HomeIcon />
        </div>
        <h1>Создай первую цель</h1>
        <p>
          Трекер работает вокруг долгосрочных целей: выбери направление, дневную норму и
          отмечай реальные учебные сессии таймером.
        </p>
        <button className="primary-button" type="button" onClick={onCreate}>
          Create goal
        </button>
      </section>
    )
  }

  return (
    <section className="goals-list">
      <div className="section-heading">
        <h2>Мои цели</h2>
        <span>{goals.length} активных</span>
      </div>

      {goals.map((goal, index) => (
        <button className="goal-card" type="button" key={goal.id} onClick={() => onOpenGoal(goal.id)}>
          <div className="goal-card__top">
            <span
              className="entry-marker"
              style={{ backgroundColor: markerColors[index % markerColors.length] }}
            />
            <div>
              <h3>{goal.title}</h3>
              <p>{goal.description || 'Долгосрочная учебная цель'}</p>
            </div>
            <span className={`status-pill status-pill--${goal.status}`}>{goal.status}</span>
          </div>

          <div className="goal-card__metrics">
            <span>Streak: {goal.currentStreak} дн.</span>
            <span>
              Today: {formatMinutes(goal.todayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}
            </span>
          </div>

          <ProgressBar value={goal.todayProgressPct} />

          <div className="goal-card__footer">
            <span>Day {goal.currentDay} of {goal.totalDays}</span>
            <span>{goal.totalProgressPct}% всего</span>
          </div>
        </button>
      ))}
    </section>
  )
}

function CreateGoalScreen({
  form,
  formError,
  onChange,
  onSubmit,
}: {
  form: GoalForm
  formError: string
  onChange: (form: GoalForm) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <form className="entry-form" onSubmit={onSubmit}>
      <div className="section-heading">
        <h2>Создать цель</h2>
        <span>долгий фокус</span>
      </div>

      <label>
        Название
        <input
          value={form.title}
          onChange={(event) => onChange({ ...form, title: event.target.value })}
          placeholder="Learn Go"
          required
        />
      </label>

      <label>
        Описание
        <textarea
          value={form.description}
          onChange={(event) => onChange({ ...form, description: event.target.value })}
          placeholder="Изучать язык, писать API и закреплять практикой"
          rows={3}
        />
      </label>

      <div className="form-row">
        <label>
          Дней
          <input
            type="number"
            min="1"
            value={form.totalDays}
            onChange={(event) => onChange({ ...form, totalDays: event.target.value })}
            required
          />
        </label>

        <label>
          Часов в день
          <input
            type="number"
            min="0.25"
            step="0.25"
            value={form.dailyTargetHours}
            onChange={(event) => onChange({ ...form, dailyTargetHours: event.target.value })}
            required
          />
        </label>
      </div>

      <p className="form-hint">
        Старт цели будет установлен автоматически на сегодняшний день. Стрик считается ежедневно.
      </p>

      {formError && <p className="form-error">{formError}</p>}

      <button className="primary-button" type="submit">
        Create goal
      </button>
    </form>
  )
}

function GoalDetailsScreen({
  goal,
  elapsedSeconds,
  sessionState,
  onStart,
  onPause,
  onResume,
  onFinish,
}: {
  goal: GoalDetail
  elapsedSeconds: number
  sessionState: SessionState
  onStart: () => void
  onPause: () => void
  onResume: () => void
  onFinish: () => void
}) {
  const ringStyle = {
    '--ring-progress': `${goal.todayProgressPct}%`,
  } as CSSProperties

  return (
    <>
      <section className="hero-panel">
        <div className="hero-copy">
          <span className="flame-orb" aria-hidden="true">
            <HomeIcon />
          </span>
          <div>
            <strong>{goal.currentStreak}</strong>
            <p>дней streak</p>
          </div>
        </div>
        <div
          className="hero-ring"
          style={ringStyle}
          aria-label={`Сегодня выполнено ${goal.todayProgressPct}%`}
        >
          <span>{goal.todayProgressPct}%</span>
        </div>
      </section>

      <section className="goal-panel">
        <div className="section-heading">
          <h2>{goal.title}</h2>
          <span>{goal.status}</span>
        </div>
        <p>{formatMinutes(goal.todayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}</p>
        <small>{formatMinutes(goal.todayRemainingMinutes)} осталось сегодня</small>
        <ProgressBar value={goal.todayProgressPct} />
        <div className="goal-meta">
          <span>Day {goal.currentDay} of {goal.totalDays}</span>
          <span>{goal.totalProgressPct}% цели</span>
        </div>
      </section>

      <section className="timer-panel">
        {sessionState === 'idle' && (
          <button className="primary-button primary-button--large" type="button" onClick={onStart}>
            Start session
          </button>
        )}

        {sessionState !== 'idle' && (
          <>
            <p>{sessionState === 'running' ? 'Сессия идет' : 'Пауза'}</p>
            <strong>{formatTimer(elapsedSeconds)}</strong>
            <div className="timer-actions">
              {sessionState === 'running' ? (
                <button type="button" onClick={onPause}>Pause</button>
              ) : (
                <button type="button" onClick={onResume}>Resume</button>
              )}
              <button type="button" onClick={onFinish}>Finish session</button>
            </div>
          </>
        )}
      </section>

      <HistorySection sessions={goal.recentSessions} />
    </>
  )
}

function FinishSessionModal({
  goal,
  elapsedSeconds,
  notes,
  tags,
  formError,
  onNotesChange,
  onTagsChange,
  onSave,
  onDiscard,
}: {
  goal: GoalDetail
  elapsedSeconds: number
  notes: string
  tags: string
  formError: string
  onNotesChange: (value: string) => void
  onTagsChange: (value: string) => void
  onSave: () => void
  onDiscard: () => void
}) {
  return (
    <div className="modal-backdrop">
      <section className="bottom-sheet">
        <div className="section-heading">
          <h2>Session completed</h2>
          <span>{formatTimer(elapsedSeconds)}</span>
        </div>
        <p className="sheet-subtitle">{goal.title}</p>

        <label>
          Notes
          <textarea
            value={notes}
            onChange={(event) => onNotesChange(event.target.value)}
            placeholder="Что сделал, изучил или завершил сегодня?"
            rows={4}
          />
        </label>

        <label>
          Tags
          <input
            value={tags}
            onChange={(event) => onTagsChange(event.target.value)}
            placeholder="SQLite, HTTP, handlers"
          />
        </label>

        {formError && <p className="form-error">{formError}</p>}

        <div className="sheet-actions">
          <button className="ghost-button" type="button" onClick={onDiscard}>Discard</button>
          <button className="primary-button" type="button" onClick={onSave}>Save session</button>
        </div>
      </section>
    </div>
  )
}

function StatsScreen({ stats }: { stats: AppStats }) {
  const todayPercent = percent(stats.todayMinutes, stats.dailyTargetMinutes)

  return (
    <>
      <section className="stats-grid stats-grid--large" aria-label="Статистика">
        <article className="stat-card">
          <p>Sessions</p>
          <strong>{stats.totalSessions}</strong>
        </article>
        <article className="stat-card">
          <p>Practice</p>
          <strong>{formatMinutes(stats.totalPracticeMinutes)}</strong>
        </article>
        <article className="stat-card">
          <p>Current streak</p>
          <strong>{stats.currentStreak}</strong>
        </article>
        <article className="stat-card">
          <p>Longest streak</p>
          <strong>{stats.longestStreak}</strong>
        </article>
      </section>

      <section className="chart-panel">
        <div className="section-heading">
          <h2>Сегодня</h2>
          <span>{todayPercent}% нормы</span>
        </div>
        <p className="chart-caption">
          {formatMinutes(stats.todayMinutes)} / {formatMinutes(stats.dailyTargetMinutes)}
        </p>
        <ProgressBar value={todayPercent} />
      </section>

      <section className="chart-panel">
        <div className="section-heading">
          <h2>Неделя</h2>
          <span>факт против нормы</span>
        </div>
        <div className="weekly-chart">
          {stats.weekly.map((day) => {
            const value = percent(day.minutes, day.targetMinutes)
            return (
              <div className="week-day" key={day.date}>
                <span className="week-bar">
                  <span
                    className={day.isCompleted ? 'is-complete' : ''}
                    style={{ height: `${Math.max(value, day.minutes > 0 ? 12 : 0)}%` }}
                  />
                </span>
                <small>{day.label}</small>
              </div>
            )
          })}
        </div>
      </section>

      <section className="chart-panel">
        <div className="section-heading">
          <h2>Месяц</h2>
          <span>{formatMinutes(stats.monthlyTotalMinutes)}</span>
        </div>
        {stats.goalDistribution.length === 0 && (
          <p className="empty-message">Распределение появится после первой сессии.</p>
        )}
        {stats.goalDistribution.map((item, index) => (
          <article className="category-row" key={item.goalId}>
            <span
              className="entry-marker"
              style={{ backgroundColor: markerColors[index % markerColors.length] }}
            />
            <div>
              <div className="category-line">
                <p>{item.title}</p>
                <strong>{formatMinutes(item.minutes)}</strong>
              </div>
              <span className="bar-track">
                <span style={{ width: `${item.percent}%` }} />
              </span>
            </div>
          </article>
        ))}
      </section>
    </>
  )
}

function HistorySection({ sessions }: { sessions: Session[] }) {
  return (
    <section className="entries-section">
      <div className="section-heading">
        <h2>History</h2>
        <span>{sessions.length} recent</span>
      </div>
      {sessions.length === 0 && (
        <p className="empty-message">Сессий пока нет. Запусти таймер и сохрани результат.</p>
      )}
      {sessions.map((session, index) => (
        <article className="history-card" key={session.id}>
          <span
            className="entry-marker"
            style={{ backgroundColor: markerColors[index % markerColors.length] }}
          />
          <div>
            <p>{formatSessionDate(session.endedAt)}</p>
            <strong>{formatMinutes(session.durationMinutes)}</strong>
            {session.notes && <span>{session.notes}</span>}
            {session.tags.length > 0 && (
              <div className="tag-row">
                {session.tags.map((tag) => (
                  <small key={tag}>{tag}</small>
                ))}
              </div>
            )}
          </div>
        </article>
      ))}
    </section>
  )
}

function ProgressBar({ value }: { value: number }) {
  return (
    <span className="bar-track">
      <span style={{ width: `${Math.min(value, 100)}%` }} />
    </span>
  )
}

function formatMinutes(totalMinutes: number) {
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (hours === 0) {
    return `${minutes}m`
  }

  if (minutes === 0) {
    return `${hours}h`
  }

  return `${hours}h ${minutes}m`
}

function formatTimer(seconds: number) {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const restSeconds = seconds % 60

  return [hours, minutes, restSeconds]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}

function formatSessionDate(value: string) {
  return new Intl.DateTimeFormat('en-US', {
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}

function percent(value: number, total: number) {
  if (total <= 0) {
    return 0
  }

  return Math.min(Math.round((value / total) * 100), 100)
}

function HomeIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M3 11.5 12 4l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1v-8.5Z" />
    </svg>
  )
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12 5v14M5 12h14" />
    </svg>
  )
}

function ChartIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M5 19V9M12 19V5M19 19v-7" />
    </svg>
  )
}

export default App
