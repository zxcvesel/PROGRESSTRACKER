import { useEffect, useMemo, useState, type CSSProperties, type FormEvent, type ReactNode } from 'react'
import './App.css'

type View = 'goals' | 'create' | 'stats'
type SessionState = 'idle' | 'running' | 'paused'
type ThemeMode = 'dark' | 'light'
type AccentColor = 'cyan' | 'purple' | 'orange' | 'green'
type FontSize = 'compact' | 'default' | 'large'
type AppLanguage = 'en' | 'ru'

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
  calendar: WeeklyStat[]
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
  completedDays: number
  missedDays: number
  completionRate: number
  weeklyCompletionRate: number
  previousWeekMinutes: number
  weekComparisonPct: number
  todayMinutes: number
  dailyTargetMinutes: number
  weekly: WeeklyStat[]
  calendar: WeeklyStat[]
  monthlyTotalMinutes: number
  goalDistribution: GoalDistribution[]
  goalId: number
  goalTitle: string
}

type GoalForm = {
  title: string
  description: string
  totalDays: string
  dailyTargetHours: string
  dailyTargetMinutes: string
}

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

const defaultStats: AppStats = {
  totalSessions: 0,
  totalPracticeMinutes: 0,
  currentStreak: 0,
  longestStreak: 0,
  completedDays: 0,
  missedDays: 0,
  completionRate: 0,
  weeklyCompletionRate: 0,
  previousWeekMinutes: 0,
  weekComparisonPct: 0,
  todayMinutes: 0,
  dailyTargetMinutes: 0,
  weekly: [],
  calendar: [],
  monthlyTotalMinutes: 0,
  goalDistribution: [],
  goalId: 0,
  goalTitle: '',
}

const markerColors = ['#19f7e8', '#ff7a3d', '#e6d37a', '#b45cff', '#58d8ff']
const timerSpeeds = [0.5, 1, 1.5, 2, 5]
const settingsStorageKey = 'progress-tracker-settings'
const defaultSettings: AppSettings = {
  theme: 'dark',
  accent: 'cyan',
  fontSize: 'default',
  reducedEffects: false,
  language: 'en',
  defaultGoalDays: '90',
  defaultTargetHours: '2',
  defaultTargetMinutes: '0',
  confirmGoalDelete: true,
}

const translations = {
  en: {
    screenNewGoal: 'New goal',
    screenStats: 'Stats',
    screenGoal: 'Goal',
    screenGoals: 'Goals',
    backToGoals: 'Back to goals',
    openSettings: 'Open settings',
    mainNavigation: 'Main navigation',
    navGoals: 'Goals',
    navCreateGoal: 'Create goal',
    navStats: 'Stats',
    loadingGoals: 'Loading goals...',
    emptyGoalTitle: 'Create your first goal',
    emptyGoalText: 'Build long-term momentum: choose a focus, set a daily target, and track real practice sessions with the timer.',
    createGoal: 'Create goal',
    myGoals: 'My goals',
    activeGoals: 'active',
    goalFallbackDescription: 'Long-term learning goal',
    streak: 'Streak',
    today: 'Today',
    day: 'Day',
    of: 'of',
    total: 'total',
    createGoalTitle: 'Create goal',
    longTermFocus: 'long-term focus',
    title: 'Title',
    description: 'Description',
    days: 'Days',
    dailyTargetHours: 'Daily target hours',
    minutes: 'Minutes',
    createHint: 'The goal starts today automatically. Streaks are counted daily.',
    editGoal: 'Edit goal',
    adjustTarget: 'adjust target',
    cancel: 'Cancel',
    saveChanges: 'Save changes',
    dayStreak: 'day streak',
    goalCompleted: 'Goal completed',
    leftToday: 'left today',
    todayTarget: 'Today target',
    practiced: 'practiced',
    devTimer: 'Dev timer',
    startSession: 'Start session',
    sessionRunning: 'Session running',
    paused: 'Paused',
    pause: 'Pause',
    resume: 'Resume',
    finishSession: 'Finish session',
    deleteGoal: 'Delete goal',
    sessionCompleted: 'Session completed',
    notes: 'Notes',
    sessionNotesPlaceholder: 'What did you do, learn, or complete today?',
    tags: 'Tags',
    tagsPlaceholder: 'SQLite, HTTP, handlers',
    back: 'Back',
    saveSession: 'Save session',
    sessions: 'Sessions',
    practice: 'Practice',
    currentStreak: 'Current streak',
    longestStreak: 'Longest streak',
    targetPercent: 'of target',
    week: 'Week',
    actualVsTarget: 'actual vs target',
    month: 'Month',
    emptyDistribution: 'Distribution will appear after your first session.',
    allGoals: 'All goals',
    selectedGoal: 'Selected goal',
    calendar: 'Calendar',
    completedDays: 'Completed days',
    missedDays: 'Missed days',
    completionRate: 'Completion rate',
    weeklyCompletionRate: 'Weekly rate',
    previousWeek: 'Previous week',
    weekComparison: 'Week comparison',
    completedDay: 'Completed',
    partialDay: 'Partial',
    missedDay: 'Missed',
    history: 'History',
    recent: 'recent',
    emptyHistory: 'No sessions yet. Start the timer and save your result.',
    edit: 'Edit',
    delete: 'Delete',
    save: 'Save',
    settings: 'Settings',
    settingsSubtitle: 'Personalize the tracker',
    closeSettings: 'Close settings',
    appearance: 'Appearance',
    theme: 'Theme',
    themeDark: 'Dark',
    themeLight: 'Light',
    accentColor: 'Accent color',
    fontSize: 'Font size',
    fontCompact: 'Compact',
    fontDefault: 'Default',
    fontLarge: 'Large',
    reducedGlow: 'Reduced glow',
    language: 'Language',
    appLanguage: 'App language',
    languageNote: 'The selected language is saved and applied to the interface.',
    goals: 'Goals',
    defaultDuration: 'Default duration, days',
    defaultHours: 'Default hours',
    confirmGoalDeletion: 'Confirm goal deletion',
    about: 'About',
    product: 'Product',
    focus: 'Focus',
    goalBasedLearning: 'Goal-based learning',
    stack: 'Stack',
    dailyTargetError: 'Daily target must be greater than 0',
    createGoalError: 'Could not create goal',
    updateGoalError: 'Could not update goal',
    saveSessionError: 'Could not save session',
    updateSessionError: 'Could not update session',
    deleteGoalError: 'Could not delete goal',
    deleteSessionConfirm: 'Delete session?',
    todayCompleteAlert: 'Today target is already complete',
    deleteGoalConfirmSuffix: 'All saved sessions for this goal will also be deleted.',
    statusActive: 'active',
    statusCompleted: 'completed',
  },
  ru: {
    screenNewGoal: 'Новая цель',
    screenStats: 'Статистика',
    screenGoal: 'Цель',
    screenGoals: 'Цели',
    backToGoals: 'Назад к целям',
    openSettings: 'Открыть настройки',
    mainNavigation: 'Основная навигация',
    navGoals: 'Цели',
    navCreateGoal: 'Создать цель',
    navStats: 'Статистика',
    loadingGoals: 'Загрузка целей...',
    emptyGoalTitle: 'Создайте первую цель',
    emptyGoalText: 'Выберите фокус, задайте дневную норму и отслеживайте реальные занятия через таймер.',
    createGoal: 'Создать цель',
    myGoals: 'Мои цели',
    activeGoals: 'активных',
    goalFallbackDescription: 'Долгосрочная учебная цель',
    streak: 'Стрик',
    today: 'Сегодня',
    day: 'День',
    of: 'из',
    total: 'всего',
    createGoalTitle: 'Создание цели',
    longTermFocus: 'долгосрочный фокус',
    title: 'Название',
    description: 'Описание',
    days: 'Дни',
    dailyTargetHours: 'Дневная норма, часы',
    minutes: 'Минуты',
    createHint: 'Цель начинается сегодня автоматически. Стрик считается по дням.',
    editGoal: 'Редактировать цель',
    adjustTarget: 'изменить цель',
    cancel: 'Отмена',
    saveChanges: 'Сохранить',
    dayStreak: 'дней серии',
    goalCompleted: 'Цель выполнена',
    leftToday: 'осталось сегодня',
    todayTarget: 'Дневная норма',
    practiced: 'занятий',
    devTimer: 'Dev timer',
    startSession: 'Начать сессию',
    sessionRunning: 'Сессия идет',
    paused: 'Пауза',
    pause: 'Пауза',
    resume: 'Продолжить',
    finishSession: 'Завершить',
    deleteGoal: 'Удалить цель',
    sessionCompleted: 'Сессия завершена',
    notes: 'Заметки',
    sessionNotesPlaceholder: 'Что вы сделали, изучили или завершили сегодня?',
    tags: 'Теги',
    tagsPlaceholder: 'SQLite, HTTP, handlers',
    back: 'Назад',
    saveSession: 'Сохранить сессию',
    sessions: 'Сессии',
    practice: 'Практика',
    currentStreak: 'Текущая серия',
    longestStreak: 'Лучшая серия',
    targetPercent: 'от нормы',
    week: 'Неделя',
    actualVsTarget: 'факт против нормы',
    month: 'Месяц',
    emptyDistribution: 'Распределение появится после первой сессии.',
    allGoals: 'Все цели',
    selectedGoal: 'Выбранная цель',
    calendar: 'Календарь',
    completedDays: 'Выполненные дни',
    missedDays: 'Пропущенные дни',
    completionRate: 'Процент выполнения',
    weeklyCompletionRate: 'Процент за неделю',
    previousWeek: 'Прошлая неделя',
    weekComparison: 'Сравнение недели',
    completedDay: 'Выполнено',
    partialDay: 'Частично',
    missedDay: 'Пропущено',
    history: 'История',
    recent: 'последних',
    emptyHistory: 'Сессий пока нет. Запустите таймер и сохраните результат.',
    edit: 'Изменить',
    delete: 'Удалить',
    save: 'Сохранить',
    settings: 'Настройки',
    settingsSubtitle: 'Настройте трекер под себя',
    closeSettings: 'Закрыть настройки',
    appearance: 'Внешний вид',
    theme: 'Тема',
    themeDark: 'Темная',
    themeLight: 'Светлая',
    accentColor: 'Цвет акцента',
    fontSize: 'Размер шрифта',
    fontCompact: 'Компактный',
    fontDefault: 'Обычный',
    fontLarge: 'Крупный',
    reducedGlow: 'Меньше свечения',
    language: 'Язык',
    appLanguage: 'Язык приложения',
    languageNote: 'Выбранный язык сохраняется и применяется к интерфейсу.',
    goals: 'Цели',
    defaultDuration: 'Длительность по умолчанию, дни',
    defaultHours: 'Часы по умолчанию',
    confirmGoalDeletion: 'Подтверждать удаление цели',
    about: 'О приложении',
    product: 'Продукт',
    focus: 'Фокус',
    goalBasedLearning: 'Целевое обучение',
    stack: 'Стек',
    dailyTargetError: 'Дневная норма должна быть больше 0',
    createGoalError: 'Не удалось создать цель',
    updateGoalError: 'Не удалось обновить цель',
    saveSessionError: 'Не удалось сохранить сессию',
    updateSessionError: 'Не удалось обновить сессию',
    deleteGoalError: 'Не удалось удалить цель',
    deleteSessionConfirm: 'Удалить сессию?',
    todayCompleteAlert: 'Дневная норма уже выполнена',
    deleteGoalConfirmSuffix: 'Все сохраненные сессии этой цели также будут удалены.',
    statusActive: 'активна',
    statusCompleted: 'завершена',
  },
} as const

type Copy = typeof translations[AppLanguage]

function App() {
  const [backendStatus, setBackendStatus] = useState('checking')
  const [view, setView] = useState<View>('goals')
  const [settings, setSettings] = useState<AppSettings>(() => loadSettings())
  const copy = translations[settings.language]
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoalId, setSelectedGoalId] = useState<number | null>(null)
  const [goalDetail, setGoalDetail] = useState<GoalDetail | null>(null)
  const [stats, setStats] = useState<AppStats>(defaultStats)
  const [selectedStatsGoalId, setSelectedStatsGoalId] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [formError, setFormError] = useState('')

  const [goalForm, setGoalForm] = useState<GoalForm>(() => createDefaultGoalForm(settings))

  const [sessionState, setSessionState] = useState<SessionState>('idle')
  const [sessionStartedAt, setSessionStartedAt] = useState('')
  const [elapsedSeconds, setElapsedSeconds] = useState(0)
  const [timerSpeed, setTimerSpeed] = useState(1)
  const [finishModalOpen, setFinishModalOpen] = useState(false)
  const [sessionNotes, setSessionNotes] = useState('')
  const [sessionTags, setSessionTags] = useState('')
  const [isEditingGoal, setIsEditingGoal] = useState(false)
  const [editGoalForm, setEditGoalForm] = useState<GoalForm>({
    title: '',
    description: '',
    totalDays: '90',
    dailyTargetHours: '2',
    dailyTargetMinutes: '0',
  })
  const [editingSessionId, setEditingSessionId] = useState<number | null>(null)
  const [sessionEditNotes, setSessionEditNotes] = useState('')
  const [sessionEditTags, setSessionEditTags] = useState('')

  useEffect(() => {
    loadInitialData()
  }, [])

  useEffect(() => {
    window.localStorage.setItem(settingsStorageKey, JSON.stringify(settings))
  }, [settings])

  useEffect(() => {
    if (sessionState !== 'running' || !goalDetail) {
      return
    }

    const timer = window.setInterval(() => {
      setElapsedSeconds((current) => {
        const targetSeconds = Math.max((goalDetail.dailyTargetMinutes - goalDetail.todayMinutes) * 60, 0)
        const next = current + timerSpeed

        if (targetSeconds > 0 && next >= targetSeconds) {
          window.clearInterval(timer)
          setSessionState('paused')
          setFinishModalOpen(true)
          return targetSeconds
        }

        return next
      })
    }, 1000)

    return () => window.clearInterval(timer)
  }, [goalDetail, sessionState, timerSpeed])

  const screenTitle = useMemo(() => {
    if (view === 'create') {
      return copy.screenNewGoal
    }

    if (view === 'stats') {
      return copy.screenStats
    }

    if (selectedGoalId) {
      return copy.screenGoal
    }

    return copy.screenGoals
  }, [copy, selectedGoalId, view])

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

  async function loadStats(goalId = selectedStatsGoalId) {
    try {
      const query = goalId ? `?goalId=${goalId}` : ''
      const response = await fetch(`/api/stats${query}`)
      const data = (await response.json()) as AppStats
      setStats(data)
    } catch {
      setStats(defaultStats)
    }
  }

  async function openGoal(goalId: number) {
    setSelectedGoalId(goalId)
    setView('goals')
    setIsEditingGoal(false)
    setEditingSessionId(null)
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

    const dailyTargetMinutes =
      Number(goalForm.dailyTargetHours) * 60 + Number(goalForm.dailyTargetMinutes)

    if (dailyTargetMinutes <= 0) {
      setFormError(copy.dailyTargetError)
      return
    }

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
          dailyTargetMinutes,
        }),
      })

      if (!response.ok) {
        throw new Error('Failed to create goal')
      }

      const createdGoal = (await response.json()) as GoalSummary
      setGoals((currentGoals) => [createdGoal, ...currentGoals])
      setGoalForm(createDefaultGoalForm(settings))
      setView('goals')
      await openGoal(createdGoal.id)
      await loadStats()
    } catch {
      setFormError(copy.createGoalError)
    }
  }

  function startSession() {
    if (!goalDetail) {
      return
    }

    if (goalDetail.todayMinutes >= goalDetail.dailyTargetMinutes) {
      window.alert(copy.todayCompleteAlert)
      return
    }

    setSessionStartedAt(toLocalISOString(new Date()))
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
        startedAt: sessionStartedAt || toLocalISOString(new Date()),
        endedAt: toLocalISOString(new Date()),
        durationMinutes: Math.max(1, Math.ceil(elapsedSeconds / 60)),
        notes: sessionNotes,
        tags: sessionTags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      }),
    })

    if (!response.ok) {
      setFormError(copy.saveSessionError)
      return
    }

    resetSession()
    await Promise.all([loadGoals(), loadStats(), loadGoalDetail(goalDetail.id)])
  }

  function startEditGoal() {
    if (!goalDetail) {
      return
    }

    setEditGoalForm(goalToForm(goalDetail))
    setIsEditingGoal(true)
    setFormError('')
  }

  async function updateGoal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    if (!goalDetail) {
      return
    }

    const dailyTargetMinutes =
      Number(editGoalForm.dailyTargetHours) * 60 + Number(editGoalForm.dailyTargetMinutes)

    if (dailyTargetMinutes <= 0) {
      setFormError(copy.dailyTargetError)
      return
    }

    const response = await fetch(`/api/goals/${goalDetail.id}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        title: editGoalForm.title,
        description: editGoalForm.description,
        totalDays: Number(editGoalForm.totalDays),
        dailyTargetMinutes,
        status: goalDetail.status,
      }),
    })

    if (!response.ok) {
      setFormError(copy.updateGoalError)
      return
    }

    setIsEditingGoal(false)
    setFormError('')
    await Promise.all([loadGoals(), loadStats(), loadGoalDetail(goalDetail.id)])
  }

  async function deleteGoal() {
    if (!goalDetail) {
      return
    }

    if (settings.confirmGoalDelete) {
      const confirmed = window.confirm(
        `${copy.deleteGoal} "${goalDetail.title}"? ${copy.deleteGoalConfirmSuffix}`,
      )
      if (!confirmed) {
        return
      }
    }

    const response = await fetch(`/api/goals/${goalDetail.id}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      window.alert(copy.deleteGoalError)
      return
    }

    resetSession()
    setSelectedGoalId(null)
    setGoalDetail(null)
    setView('goals')
    setIsEditingGoal(false)
    await Promise.all([loadGoals(), loadStats()])
  }

  function startEditSession(session: Session) {
    setEditingSessionId(session.id)
    setSessionEditNotes(session.notes)
    setSessionEditTags(session.tags.join(', '))
  }

  async function updateSession(sessionId: number) {
    if (!goalDetail) {
      return
    }

    const response = await fetch(`/api/goals/${goalDetail.id}/sessions/${sessionId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        notes: sessionEditNotes,
        tags: sessionEditTags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      }),
    })

    if (!response.ok) {
      window.alert(copy.updateSessionError)
      return
    }

    setEditingSessionId(null)
    await Promise.all([loadGoals(), loadStats(), loadGoalDetail(goalDetail.id)])
  }

  async function deleteSession(session: Session) {
    if (!goalDetail) {
      return
    }

    const confirmed = window.confirm(`${copy.deleteSessionConfirm} ${formatMinutes(session.durationMinutes)}`)
    if (!confirmed) {
      return
    }

    const response = await fetch(`/api/goals/${goalDetail.id}/sessions/${session.id}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      window.alert('Could not delete session')
      return
    }

    if (editingSessionId === session.id) {
      setEditingSessionId(null)
    }
    await Promise.all([loadGoals(), loadStats(), loadGoalDetail(goalDetail.id)])
  }

  function closeFinishModal() {
    setFinishModalOpen(false)
    setFormError('')
  }

  function resetSession() {
    setSessionState('idle')
    setElapsedSeconds(0)
    setSessionStartedAt('')
    setFinishModalOpen(false)
    setSessionNotes('')
    setSessionTags('')
    setFormError('')
    setTimerSpeed(1)
  }

  return (
    <main
      className={[
        'page-shell',
        `theme-${settings.theme}`,
        `accent-${settings.accent}`,
        `font-${settings.fontSize}`,
        settings.reducedEffects ? 'effects-reduced' : '',
      ].filter(Boolean).join(' ')}
    >
      <section className="phone-shell" aria-label="Progress Tracker">
        <div className="screen-content">
          <header className="top-bar">
            <button
              className={`icon-button ${selectedGoalId ? 'icon-button--back' : 'icon-button--menu'}`}
              type="button"
              aria-label={selectedGoalId ? copy.backToGoals : copy.openSettings}
              onClick={() => {
                if (selectedGoalId) {
                  setSelectedGoalId(null)
                  setGoalDetail(null)
                  setIsEditingGoal(false)
                  setEditingSessionId(null)
                  resetSession()
                  return
                }

                setSettingsOpen(true)
              }}
            >
              {selectedGoalId ? (
                <BackIcon />
              ) : (
                <>
                  <span />
                  <span />
                  <span />
                </>
              )}
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
              copy={copy}
              onCreate={() => {
                setGoalForm(createDefaultGoalForm(settings))
                setView('create')
              }}
              onOpenGoal={openGoal}
            />
          )}

          {view === 'goals' && selectedGoalId && goalDetail && (
            <GoalDetailsScreen
              goal={goalDetail}
              copy={copy}
              language={settings.language}
              elapsedSeconds={elapsedSeconds}
              timerSpeed={timerSpeed}
              isEditingGoal={isEditingGoal}
              editGoalForm={editGoalForm}
              formError={formError}
              sessionState={sessionState}
              editingSessionId={editingSessionId}
              sessionEditNotes={sessionEditNotes}
              sessionEditTags={sessionEditTags}
              onTimerSpeedChange={setTimerSpeed}
              onEditGoalChange={setEditGoalForm}
              onEditGoalSubmit={updateGoal}
              onEditGoalStart={startEditGoal}
              onEditGoalCancel={() => {
                setIsEditingGoal(false)
                setFormError('')
              }}
              onStart={startSession}
              onPause={pauseSession}
              onResume={resumeSession}
              onFinish={finishSession}
              onDelete={deleteGoal}
              onEditSessionStart={startEditSession}
              onEditSessionCancel={() => setEditingSessionId(null)}
              onEditSessionSave={updateSession}
              onDeleteSession={deleteSession}
              onSessionEditNotesChange={setSessionEditNotes}
              onSessionEditTagsChange={setSessionEditTags}
            />
          )}

          {view === 'create' && (
            <CreateGoalScreen
              form={goalForm}
              formError={formError}
              copy={copy}
              onChange={setGoalForm}
              onSubmit={handleCreateGoal}
            />
          )}

          {view === 'stats' && (
            <StatsScreen
              stats={stats}
              goals={goals}
              selectedGoalId={selectedStatsGoalId}
              copy={copy}
              language={settings.language}
              onGoalChange={(goalId) => {
                setSelectedStatsGoalId(goalId)
                void loadStats(goalId)
              }}
            />
          )}
        </div>

        <nav className="bottom-nav" aria-label={copy.mainNavigation}>
          <button
            className={view === 'goals' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setView('goals')
              setSelectedGoalId(null)
              setGoalDetail(null)
              setIsEditingGoal(false)
              setEditingSessionId(null)
            }}
            aria-label={copy.navGoals}
          >
            <FlameIcon />
          </button>
          <button
            className={view === 'create' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setGoalForm(createDefaultGoalForm(settings))
              setView('create')
              setSelectedGoalId(null)
              setGoalDetail(null)
              setIsEditingGoal(false)
              setEditingSessionId(null)
            }}
            aria-label={copy.navCreateGoal}
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
              setIsEditingGoal(false)
              setEditingSessionId(null)
              void loadStats(selectedStatsGoalId)
            }}
            aria-label={copy.navStats}
          >
            <ChartIcon />
          </button>
        </nav>

        {finishModalOpen && goalDetail && (
          <FinishSessionModal
            goal={goalDetail}
            copy={copy}
            elapsedSeconds={elapsedSeconds}
            notes={sessionNotes}
            tags={sessionTags}
            formError={formError}
            onNotesChange={setSessionNotes}
            onTagsChange={setSessionTags}
            onSave={saveSession}
            onClose={closeFinishModal}
          />
        )}

        <SettingsDrawer
          isOpen={settingsOpen}
          settings={settings}
          copy={copy}
          onClose={() => setSettingsOpen(false)}
          onChange={(nextSettings) => setSettings((current) => ({ ...current, ...nextSettings }))}
        />
      </section>
    </main>
  )
}

function SettingsDrawer({
  isOpen,
  settings,
  copy,
  onClose,
  onChange,
}: {
  isOpen: boolean
  settings: AppSettings
  copy: Copy
  onClose: () => void
  onChange: (settings: Partial<AppSettings>) => void
}) {
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

function GoalsScreen({
  goals,
  isLoading,
  copy,
  onCreate,
  onOpenGoal,
}: {
  goals: GoalSummary[]
  isLoading: boolean
  copy: Copy
  onCreate: () => void
  onOpenGoal: (goalId: number) => void
}) {
  if (isLoading) {
    return <p className="empty-message">{copy.loadingGoals}</p>
  }

  if (goals.length === 0) {
    return (
      <section className="empty-state">
        <div className="flame-orb" aria-hidden="true">
          <FlameIcon />
        </div>
        <h1>{copy.emptyGoalTitle}</h1>
        <p>{copy.emptyGoalText}</p>
        <button className="primary-button" type="button" onClick={onCreate}>
          {copy.createGoal}
        </button>
      </section>
    )
  }

  return (
    <section className="goals-list">
      <div className="section-heading">
        <h2>{copy.myGoals}</h2>
        <span>{goals.length} {copy.activeGoals}</span>
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
              <p>{goal.description || copy.goalFallbackDescription}</p>
            </div>
            <span className={`status-pill status-pill--${goal.status}`}>{statusLabel(goal.status, copy)}</span>
          </div>

          <div className="goal-card__metrics">
            <span>{copy.streak}: {goal.currentStreak}</span>
            <span>
              {copy.today}: {formatMinutes(goal.todayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}
            </span>
          </div>

          <ProgressBar value={goal.todayProgressPct} />

          <div className="goal-card__footer">
            <span>{copy.day} {goal.currentStreak} {copy.of} {goal.totalDays}</span>
            <span>{goal.totalProgressPct}% {copy.total}</span>
          </div>
        </button>
      ))}
    </section>
  )
}

function CreateGoalScreen({
  form,
  formError,
  copy,
  onChange,
  onSubmit,
}: {
  form: GoalForm
  formError: string
  copy: Copy
  onChange: (form: GoalForm) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <form className="entry-form" onSubmit={onSubmit}>
      <div className="section-heading">
        <h2>{copy.createGoalTitle}</h2>
        <span>{copy.longTermFocus}</span>
      </div>

      <label>
        {copy.title}
        <input
          value={form.title}
          onChange={(event) => onChange({ ...form, title: event.target.value })}
          placeholder="Learn Go"
          required
        />
      </label>

      <label>
        {copy.description}
        <textarea
          value={form.description}
          onChange={(event) => onChange({ ...form, description: event.target.value })}
          placeholder="Study the language, build APIs, and reinforce with practice"
          rows={3}
        />
      </label>

      <div className="form-row form-row--single">
        <label>
          {copy.days}
          <input
            type="number"
            min="1"
            value={form.totalDays}
            onChange={(event) => onChange({ ...form, totalDays: event.target.value })}
            required
          />
        </label>
      </div>

      <div className="form-row form-row--target">
        <label>
          {copy.dailyTargetHours}
          <input
            type="number"
            min="0"
            step="1"
            value={form.dailyTargetHours}
            onChange={(event) => onChange({ ...form, dailyTargetHours: event.target.value })}
            required
          />
        </label>

        <label>
          {copy.minutes}
          <input
            type="number"
            min="0"
            max="59"
            step="5"
            value={form.dailyTargetMinutes}
            onChange={(event) => onChange({ ...form, dailyTargetMinutes: event.target.value })}
            required
          />
        </label>
      </div>

      <p className="form-hint">{copy.createHint}</p>

      {formError && <p className="form-error">{formError}</p>}

      <button className="primary-button" type="submit">
        {copy.createGoal}
      </button>
    </form>
  )
}

function GoalEditForm({
  form,
  formError,
  copy,
  onChange,
  onSubmit,
  onCancel,
}: {
  form: GoalForm
  formError: string
  copy: Copy
  onChange: (form: GoalForm) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onCancel: () => void
}) {
  return (
    <form className="entry-form" onSubmit={onSubmit}>
      <div className="section-heading">
        <h2>{copy.editGoal}</h2>
        <span>{copy.adjustTarget}</span>
      </div>

      <label>
        {copy.title}
        <input
          value={form.title}
          onChange={(event) => onChange({ ...form, title: event.target.value })}
          required
        />
      </label>

      <label>
        {copy.description}
        <textarea
          value={form.description}
          onChange={(event) => onChange({ ...form, description: event.target.value })}
          rows={3}
        />
      </label>

      <div className="form-row form-row--single">
        <label>
          {copy.days}
          <input
            type="number"
            min="1"
            value={form.totalDays}
            onChange={(event) => onChange({ ...form, totalDays: event.target.value })}
            required
          />
        </label>
      </div>

      <div className="form-row form-row--target">
        <label>
          {copy.dailyTargetHours}
          <input
            type="number"
            min="0"
            step="1"
            value={form.dailyTargetHours}
            onChange={(event) => onChange({ ...form, dailyTargetHours: event.target.value })}
            required
          />
        </label>

        <label>
          {copy.minutes}
          <input
            type="number"
            min="0"
            max="59"
            step="5"
            value={form.dailyTargetMinutes}
            onChange={(event) => onChange({ ...form, dailyTargetMinutes: event.target.value })}
            required
          />
        </label>
      </div>

      {formError && <p className="form-error">{formError}</p>}

      <div className="sheet-actions">
        <button className="ghost-button" type="button" onClick={onCancel}>{copy.cancel}</button>
        <button className="primary-button" type="submit">{copy.saveChanges}</button>
      </div>
    </form>
  )
}

function GoalDetailsScreen({
  goal,
  copy,
  language,
  elapsedSeconds,
  timerSpeed,
  isEditingGoal,
  editGoalForm,
  formError,
  sessionState,
  editingSessionId,
  sessionEditNotes,
  sessionEditTags,
  onTimerSpeedChange,
  onEditGoalChange,
  onEditGoalSubmit,
  onEditGoalStart,
  onEditGoalCancel,
  onStart,
  onPause,
  onResume,
  onFinish,
  onDelete,
  onEditSessionStart,
  onEditSessionCancel,
  onEditSessionSave,
  onDeleteSession,
  onSessionEditNotesChange,
  onSessionEditTagsChange,
}: {
  goal: GoalDetail
  copy: Copy
  language: AppLanguage
  elapsedSeconds: number
  timerSpeed: number
  isEditingGoal: boolean
  editGoalForm: GoalForm
  formError: string
  sessionState: SessionState
  editingSessionId: number | null
  sessionEditNotes: string
  sessionEditTags: string
  onTimerSpeedChange: (speed: number) => void
  onEditGoalChange: (form: GoalForm) => void
  onEditGoalSubmit: (event: FormEvent<HTMLFormElement>) => void
  onEditGoalStart: () => void
  onEditGoalCancel: () => void
  onStart: () => void
  onPause: () => void
  onResume: () => void
  onFinish: () => void
  onDelete: () => void
  onEditSessionStart: (session: Session) => void
  onEditSessionCancel: () => void
  onEditSessionSave: (sessionId: number) => void
  onDeleteSession: (session: Session) => void
  onSessionEditNotesChange: (value: string) => void
  onSessionEditTagsChange: (value: string) => void
}) {
  const liveSessionMinutes = sessionState === 'idle' ? 0 : Math.ceil(elapsedSeconds / 60)
  const liveTodayMinutes = goal.todayMinutes + liveSessionMinutes
  const liveTodayProgressPct = percent(liveTodayMinutes, goal.dailyTargetMinutes)
  const liveRemainingMinutes = Math.max(goal.dailyTargetMinutes - liveTodayMinutes, 0)
  const ringStyle = {
    '--ring-progress': `${goal.totalProgressPct}%`,
  } as CSSProperties

  return (
    <>
      <section className="hero-panel">
        <div className="hero-copy">
          <span className="flame-orb" aria-hidden="true">
            <FlameIcon />
          </span>
          <div>
            <strong>{goal.currentStreak}</strong>
            <p>{copy.dayStreak}</p>
          </div>
        </div>
        <div
          className="hero-ring"
          style={ringStyle}
          aria-label={`${copy.goalCompleted} ${goal.totalProgressPct}%`}
        >
          <span>{goal.totalProgressPct}%</span>
        </div>
      </section>

      <section className="goal-panel">
        <div className="section-heading">
          <h2>{goal.title}</h2>
          <span>{statusLabel(goal.status, copy)}</span>
        </div>
        <p>{formatMinutes(liveTodayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}</p>
        <small>{formatMinutes(liveRemainingMinutes)} {copy.leftToday}</small>
        <div className="goal-progress-label">
          <span>{copy.todayTarget}</span>
          <span>{liveTodayProgressPct}%</span>
        </div>
        <ProgressBar value={liveTodayProgressPct} />
        <div className="goal-meta">
          <span>{copy.day} {goal.currentStreak} {copy.of} {goal.totalDays}</span>
          <span>{formatMinutes(goal.totalPracticeMinutes)} {copy.practiced}</span>
        </div>
        <button className="ghost-button compact-button" type="button" onClick={onEditGoalStart}>
          {copy.editGoal}
        </button>
      </section>

      {isEditingGoal && (
        <GoalEditForm
          form={editGoalForm}
          formError={formError}
          copy={copy}
          onChange={onEditGoalChange}
          onSubmit={onEditGoalSubmit}
          onCancel={onEditGoalCancel}
        />
      )}

      <ActivityCalendar
        days={goal.calendar}
        copy={copy}
        language={language}
      />

      <section className="timer-panel">
        <div className="dev-speed-panel" aria-label="Development timer speed">
          <span>{copy.devTimer}</span>
          <div>
            {timerSpeeds.map((speed) => (
              <button
                className={timerSpeed === speed ? 'is-selected' : ''}
                type="button"
                key={speed}
                onClick={() => onTimerSpeedChange(speed)}
              >
                {speed}x
              </button>
            ))}
          </div>
        </div>

        {sessionState === 'idle' && (
          <button className="primary-button primary-button--large" type="button" onClick={onStart}>
            {copy.startSession}
          </button>
        )}

        {sessionState !== 'idle' && (
          <>
            <p>{sessionState === 'running' ? copy.sessionRunning : copy.paused}</p>
            <strong>{formatTimer(elapsedSeconds)}</strong>
            <div className="timer-actions">
              {sessionState === 'running' ? (
                <button type="button" onClick={onPause}>{copy.pause}</button>
              ) : (
                <button type="button" onClick={onResume}>{copy.resume}</button>
              )}
              <button type="button" onClick={onFinish}>{copy.finishSession}</button>
            </div>
          </>
        )}
      </section>

      <HistorySection
        sessions={goal.recentSessions}
        copy={copy}
        language={language}
        editingSessionId={editingSessionId}
        editNotes={sessionEditNotes}
        editTags={sessionEditTags}
        onEditStart={onEditSessionStart}
        onEditCancel={onEditSessionCancel}
        onEditSave={onEditSessionSave}
        onDelete={onDeleteSession}
        onNotesChange={onSessionEditNotesChange}
        onTagsChange={onSessionEditTagsChange}
      />

      <section className="danger-panel">
        <button className="danger-button" type="button" onClick={onDelete}>
          {copy.deleteGoal}
        </button>
      </section>
    </>
  )
}

function FinishSessionModal({
  goal,
  copy,
  elapsedSeconds,
  notes,
  tags,
  formError,
  onNotesChange,
  onTagsChange,
  onSave,
  onClose,
}: {
  goal: GoalDetail
  copy: Copy
  elapsedSeconds: number
  notes: string
  tags: string
  formError: string
  onNotesChange: (value: string) => void
  onTagsChange: (value: string) => void
  onSave: () => void
  onClose: () => void
}) {
  return (
    <div className="modal-backdrop">
      <section className="bottom-sheet">
        <div className="section-heading">
          <h2>{copy.sessionCompleted}</h2>
          <span>{formatTimer(elapsedSeconds)}</span>
        </div>
        <p className="sheet-subtitle">{goal.title}</p>

        <label>
          {copy.notes}
          <textarea
            value={notes}
            onChange={(event) => onNotesChange(event.target.value)}
            placeholder={copy.sessionNotesPlaceholder}
            rows={4}
          />
        </label>

        <label>
          {copy.tags}
          <input
            value={tags}
            onChange={(event) => onTagsChange(event.target.value)}
            placeholder={copy.tagsPlaceholder}
          />
        </label>

        {formError && <p className="form-error">{formError}</p>}

        <div className="sheet-actions">
          <button className="ghost-button" type="button" onClick={onClose}>{copy.back}</button>
          <button className="primary-button" type="button" onClick={onSave}>{copy.saveSession}</button>
        </div>
      </section>
    </div>
  )
}

function StatsScreen({
  stats,
  goals,
  selectedGoalId,
  copy,
  language,
  onGoalChange,
}: {
  stats: AppStats
  goals: GoalSummary[]
  selectedGoalId: number
  copy: Copy
  language: AppLanguage
  onGoalChange: (goalId: number) => void
}) {
  const todayPercent = percent(stats.todayMinutes, stats.dailyTargetMinutes)

  return (
    <>
      <section className="chart-panel">
        <label className="stats-goal-filter">
          {copy.selectedGoal}
          <select
            value={selectedGoalId}
            onChange={(event) => onGoalChange(Number(event.target.value))}
          >
            <option value={0}>{copy.allGoals}</option>
            {goals.map((goal) => (
              <option value={goal.id} key={goal.id}>{goal.title}</option>
            ))}
          </select>
        </label>
      </section>

      <section className="stats-grid stats-grid--large" aria-label={copy.screenStats}>
        <article className="stat-card">
          <p>{copy.sessions}</p>
          <strong>{stats.totalSessions}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.practice}</p>
          <strong>{formatMinutes(stats.totalPracticeMinutes)}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.currentStreak}</p>
          <strong>{stats.currentStreak}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.longestStreak}</p>
          <strong>{stats.longestStreak}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.completedDays}</p>
          <strong>{stats.completedDays}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.missedDays}</p>
          <strong>{stats.missedDays}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.completionRate}</p>
          <strong>{stats.completionRate}%</strong>
        </article>
        <article className="stat-card">
          <p>{copy.weeklyCompletionRate}</p>
          <strong>{stats.weeklyCompletionRate}%</strong>
        </article>
      </section>

      <section className="chart-panel">
        <div className="section-heading">
          <h2>{copy.today}</h2>
          <span>{todayPercent}% {copy.targetPercent}</span>
        </div>
        <p className="chart-caption">
          {formatMinutes(stats.todayMinutes)} / {formatMinutes(stats.dailyTargetMinutes)}
        </p>
        <ProgressBar value={todayPercent} />
      </section>

      <section className="chart-panel">
        <div className="section-heading">
          <h2>{copy.week}</h2>
          <span>{stats.weekComparisonPct >= 0 ? '+' : ''}{stats.weekComparisonPct}%</span>
        </div>
        <p className="chart-caption">
          {copy.previousWeek}: {formatMinutes(stats.previousWeekMinutes)} · {copy.weekComparison}
        </p>
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
                <small>{formatWeekday(day.date, language)}</small>
              </div>
            )
          })}
        </div>
      </section>

      <ActivityCalendar
        days={stats.calendar}
        copy={copy}
        language={language}
      />

      <section className="chart-panel">
        <div className="section-heading">
          <h2>{copy.month}</h2>
          <span>{formatMinutes(stats.monthlyTotalMinutes)}</span>
        </div>
        {stats.goalDistribution.length === 0 && (
          <p className="empty-message">{copy.emptyDistribution}</p>
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

function ActivityCalendar({
  days,
  copy,
  language,
}: {
  days: WeeklyStat[]
  copy: Copy
  language: AppLanguage
}) {
  return (
    <section className="chart-panel">
      <div className="section-heading">
        <h2>{copy.calendar}</h2>
        <span>{days.length} {copy.days}</span>
      </div>
      <div className="activity-calendar" aria-label={copy.calendar}>
        {days.map((day) => {
          const state = calendarDayState(day)
          return (
            <span
              className={`calendar-day calendar-day--${state}`}
              key={day.date}
              title={`${formatFullDate(day.date, language)} · ${formatMinutes(day.minutes)} / ${formatMinutes(day.targetMinutes)}`}
              aria-label={`${formatFullDate(day.date, language)} ${formatMinutes(day.minutes)} / ${formatMinutes(day.targetMinutes)}`}
            />
          )
        })}
      </div>
      <div className="calendar-legend">
        <span><i className="calendar-day calendar-day--completed" />{copy.completedDay}</span>
        <span><i className="calendar-day calendar-day--partial" />{copy.partialDay}</span>
        <span><i className="calendar-day calendar-day--missed" />{copy.missedDay}</span>
      </div>
    </section>
  )
}

function HistorySection({
  sessions,
  copy,
  language,
  editingSessionId,
  editNotes,
  editTags,
  onEditStart,
  onEditCancel,
  onEditSave,
  onDelete,
  onNotesChange,
  onTagsChange,
}: {
  sessions: Session[]
  copy: Copy
  language: AppLanguage
  editingSessionId: number | null
  editNotes: string
  editTags: string
  onEditStart: (session: Session) => void
  onEditCancel: () => void
  onEditSave: (sessionId: number) => void
  onDelete: (session: Session) => void
  onNotesChange: (value: string) => void
  onTagsChange: (value: string) => void
}) {
  return (
    <section className="entries-section">
      <div className="section-heading">
        <h2>{copy.history}</h2>
        <span>{sessions.length} {copy.recent}</span>
      </div>
      {sessions.length === 0 && (
        <p className="empty-message">{copy.emptyHistory}</p>
      )}
      {sessions.map((session, index) => (
        <article className="history-card" key={session.id}>
          <span
            className="entry-marker"
            style={{ backgroundColor: markerColors[index % markerColors.length] }}
          />
          <div>
            <p>{formatSessionDate(session.endedAt, language)}</p>
            <strong>{formatMinutes(session.durationMinutes)}</strong>
            {editingSessionId === session.id ? (
              <div className="history-edit-form">
                <label>
                  {copy.notes}
                  <textarea
                    value={editNotes}
                    onChange={(event) => onNotesChange(event.target.value)}
                    rows={3}
                  />
                </label>
                <label>
                  {copy.tags}
                  <input
                    value={editTags}
                    onChange={(event) => onTagsChange(event.target.value)}
                    placeholder={copy.tagsPlaceholder}
                  />
                </label>
                <div className="history-actions">
                  <button type="button" onClick={onEditCancel}>{copy.cancel}</button>
                  <button className="history-action--primary" type="button" onClick={() => onEditSave(session.id)}>{copy.save}</button>
                </div>
              </div>
            ) : (
              <>
                {session.notes && <span>{session.notes}</span>}
                {session.tags.length > 0 && (
                  <div className="tag-row">
                    {session.tags.map((tag) => (
                      <small key={tag}>{tag}</small>
                    ))}
                  </div>
                )}
                <div className="history-actions">
                  <button type="button" onClick={() => onEditStart(session)}>{copy.edit}</button>
                  <button className="history-action--danger" type="button" onClick={() => onDelete(session)}>{copy.delete}</button>
                </div>
              </>
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

function goalToForm(goal: GoalSummary): GoalForm {
  return {
    title: goal.title,
    description: goal.description,
    totalDays: String(goal.totalDays),
    dailyTargetHours: String(Math.floor(goal.dailyTargetMinutes / 60)),
    dailyTargetMinutes: String(goal.dailyTargetMinutes % 60),
  }
}

function createDefaultGoalForm(settings: AppSettings): GoalForm {
  return {
    title: '',
    description: '',
    totalDays: settings.defaultGoalDays || defaultSettings.defaultGoalDays,
    dailyTargetHours: settings.defaultTargetHours || defaultSettings.defaultTargetHours,
    dailyTargetMinutes: settings.defaultTargetMinutes || defaultSettings.defaultTargetMinutes,
  }
}

function loadSettings(): AppSettings {
  try {
    const savedSettings = window.localStorage.getItem(settingsStorageKey)

    if (!savedSettings) {
      return defaultSettings
    }

    const parsedSettings = JSON.parse(savedSettings) as Partial<Omit<AppSettings, 'theme'>> & { theme?: string }

    return {
      ...defaultSettings,
      ...parsedSettings,
      theme: normalizeTheme(parsedSettings.theme),
    }
  } catch {
    return defaultSettings
  }
}

function normalizeTheme(theme: string | undefined): ThemeMode {
  if (theme === 'light') {
    return theme
  }

  return 'dark'
}

function formatTimer(seconds: number) {
  const normalizedSeconds = Math.floor(seconds)
  const hours = Math.floor(normalizedSeconds / 3600)
  const minutes = Math.floor((normalizedSeconds % 3600) / 60)
  const restSeconds = normalizedSeconds % 60

  return [hours, minutes, restSeconds]
    .map((value) => String(value).padStart(2, '0'))
    .join(':')
}

function statusLabel(status: GoalSummary['status'], copy: Copy) {
  if (status === 'completed') {
    return copy.statusCompleted
  }

  return copy.statusActive
}

function localeFor(language: AppLanguage) {
  return language === 'ru' ? 'ru-RU' : 'en-US'
}

function formatSessionDate(value: string, language: AppLanguage) {
  return new Intl.DateTimeFormat(localeFor(language), {
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}

function formatWeekday(value: string, language: AppLanguage) {
  return new Intl.DateTimeFormat(localeFor(language), {
    weekday: 'short',
  }).format(new Date(value))
}

function formatFullDate(value: string, language: AppLanguage) {
  return new Intl.DateTimeFormat(localeFor(language), {
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}

function calendarDayState(day: WeeklyStat) {
  if (day.targetMinutes <= 0) {
    return 'empty'
  }

  if (day.isCompleted) {
    return 'completed'
  }

  if (day.minutes > 0) {
    return 'partial'
  }

  return 'missed'
}

function toLocalISOString(date: Date) {
  const timezoneOffset = -date.getTimezoneOffset()
  const sign = timezoneOffset >= 0 ? '+' : '-'
  const offsetHours = Math.floor(Math.abs(timezoneOffset) / 60)
  const offsetMinutes = Math.abs(timezoneOffset) % 60
  const localDate = new Date(date.getTime() + timezoneOffset * 60 * 1000)

  return `${localDate.toISOString().slice(0, 19)}${sign}${String(offsetHours).padStart(2, '0')}:${String(offsetMinutes).padStart(2, '0')}`
}

function percent(value: number, total: number) {
  if (total <= 0) {
    return 0
  }

  return Math.min(Math.round((value / total) * 100), 100)
}

function FlameIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12.2 3.5c.5 2.7-.6 4.4-2 5.9-1.3 1.4-2.5 2.7-2.5 4.9a4.4 4.4 0 0 0 8.8.1c0-1.8-.9-3.4-2.4-4.8.1 1.4-.4 2.4-1.5 3.1-.4-2.6.8-4.7-.4-9.2Z" />
      <path d="M12 20.8c-4 0-7.1-2.8-7.1-6.8 0-2.7 1.5-4.6 3.1-6.2 1.5-1.5 3.1-3.1 3.2-5.8 4 2.8 6.4 6.4 6.4 10.6 1.1-.9 1.6-2.1 1.6-3.5 1.4 1.5 2 3.1 2 4.8 0 4.1-3.2 6.9-9.2 6.9Z" />
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

function BackIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M15 6l-6 6 6 6" />
    </svg>
  )
}

export default App
