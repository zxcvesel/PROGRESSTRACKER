import { useEffect, useMemo, useState, type CSSProperties, type FormEvent } from 'react'
import './App.css'
import { AuthScreen } from './components/AuthScreen'
import { SettingsDrawer } from './components/SettingsDrawer'
import { ActivityCalendar } from './components/ActivityCalendar'
import { StatsScreen } from './components/StatsScreen'
import { GoalForm } from './components/GoalForm'
import { GoalsScreen } from './components/GoalsScreen'
import { HistorySection } from './components/HistorySection'
import { readAPIError, requestAPI } from './api/client'

type View = 'goals' | 'create' | 'stats'
type SessionState = 'idle' | 'running' | 'paused'
type ServerTimerState = 'running' | 'paused' | 'finished'
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

type AuthUser = {
  id: number
  email: string
  name: string
  createdAt: string
  emailVerified: boolean
  timezone: string
}

type AuthResponse = {
  user: AuthUser
  developmentToken?: string
}

type AuthMode = 'login' | 'register' | 'forgot' | 'reset'

type AuthForm = {
  email: string
  name: string
  password: string
  confirmPassword: string
}

type PasswordForm = {
  currentPassword: string
  newPassword: string
  confirmPassword: string
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
  notificationsEnabled: boolean
}

type ActiveTimer = {
  goalId: number
  state: ServerTimerState
  startedAt: string
  elapsedSeconds: number
  targetSeconds: number
  speedMultiplier: number
}

type TimerStatusResponse = {
  active: boolean
  timer?: ActiveTimer
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

const settingsStorageKey = 'progress-tracker-settings'
const reminderStorageKey = 'progress-tracker-last-reminder'
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
  notificationsEnabled: false,
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
    signInTitle: 'Welcome',
    registerTitle: 'Create account',
    authSubtitle: 'Access your goals, sessions, streaks, and private progress history.',
    signIn: 'Sign in',
    createAccount: 'Create account',
    email: 'Email',
    name: 'Name',
    password: 'Password',
    confirmPassword: 'Confirm password',
    currentPassword: 'Current password',
    newPassword: 'New password',
    passwordHint: 'At least 8 characters with an uppercase letter, a number, and a special character.',
    passwordMismatch: 'Passwords do not match',
    passwordPolicyError: 'Password must include an uppercase letter, a number, and a special character.',
    showPassword: 'Show password',
    hidePassword: 'Hide password',
    noAccount: 'No account yet?',
    forgotPassword: 'Forgot password?',
    forgotPasswordTitle: 'Restore access',
    resetPasswordTitle: 'Set a new password',
    sendResetLink: 'Send reset link',
    resetPassword: 'Save new password',
    backToSignIn: 'Back to sign in',
    resetEmailSent: 'If this account exists, a reset link has been sent.',
    passwordResetDone: 'Password updated. Sign in with your new password.',
    verifyEmailTitle: 'Verify your email',
    verifyEmailText: 'Open the verification link sent to your email before using your goals.',
    resendVerification: 'Send verification again',
    verificationSent: 'Verification email sent.',
    haveAccount: 'Already have an account?',
    authError: 'Could not complete authentication',
    account: 'Account',
    signedInAs: 'Signed in as',
    displayName: 'Display name',
    saveProfile: 'Save',
    profileSaved: 'Profile updated',
    profileError: 'Could not update profile',
    changePassword: 'Change password',
    passwordChanged: 'Password changed',
    passwordChangeError: 'Could not change password',
    logout: 'Log out',
    deleteAccount: 'Delete account',
    deleteAccountHint: 'This permanently removes your goals, sessions, and account data.',
    deleteAccountConfirm: 'Enter your current password to permanently delete the account.',
    deleteAccountAction: 'Delete permanently',
    accountDeleted: 'Account deleted',
    exportData: 'Export data',
    exportJSON: 'Download JSON',
    exportCSV: 'Download CSV',
    exportError: 'Could not export data',
    logoutError: 'Could not log out',
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
    templates: 'Quick templates',
    templateCoding: 'Coding',
    templateLanguage: 'Language',
    templateFitness: 'Fitness',
    templateCodingTitle: 'Learn programming',
    templateCodingDescription: 'Practice concepts and build a small project every day.',
    templateLanguageTitle: 'Learn a language',
    templateLanguageDescription: 'Study vocabulary, listening, and speaking every day.',
    templateFitnessTitle: 'Daily training',
    templateFitnessDescription: 'Follow a consistent training routine and record each session.',
    editGoal: 'Edit goal',
    adjustTarget: 'adjust target',
    cancel: 'Cancel',
    saveChanges: 'Save changes',
    dayStreak: 'day streak',
    goalCompleted: 'Goal completed',
    leftToday: 'left today',
    todayTarget: 'Today target',
    practiced: 'practiced',
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
    emptyStatsTitle: 'No progress data yet',
    emptyStatsText: 'Create a goal and complete your first timed session to unlock statistics.',
    allGoals: 'All goals',
    selectedGoal: 'Selected goal',
    calendar: 'Calendar',
    completedDays: 'Completed days',
    missedDays: 'Missed days',
    completionRate: 'Completion rate',
    weeklyCompletionRate: 'Weekly rate',
    previousWeek: 'Previous week',
    weekComparison: 'Week comparison',
    targetReached: 'Daily target reached',
    remainingToday: 'remaining today',
    noDailyTarget: 'No active daily target',
    moreThanPrevious: 'more than previous week',
    lessThanPrevious: 'less than previous week',
    sameAsPrevious: 'same as previous week',
    completedDay: 'Completed',
    partialDay: 'Partial',
    missedDay: 'Missed',
    history: 'History',
    recent: 'recent',
    emptyHistory: 'No sessions yet. Start the timer and save your result.',
    historySearch: 'Search notes or tags',
    noHistoryResults: 'No sessions match this search.',
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
    productTagline: 'Daily focus for long-term learning goals.',
    version: 'Version',
    releaseChannel: 'Release channel',
    beta: 'Beta',
    dataPrivacy: 'Your data',
    privateAccountData: 'Private to your account',
    copyright: '© 2026 Progress Tracker',
    manageAccount: 'Manage account',
    notifications: 'Notifications',
    notificationDescription: 'Reminders while the app is open and session completion alerts.',
    enableNotifications: 'Enable',
    disableNotifications: 'Disable',
    notificationsBlocked: 'Blocked in browser',
    notificationsUnsupported: 'Not supported',
    notificationNote: 'Background alerts on iPhone require the app to be installed to the Home Screen and Web Push support.',
    targetReachedNotification: 'Daily target reached',
    reminderNotification: 'Your daily target is still waiting',
    reminderNotificationBody: 'Open Progress Tracker and continue today’s practice.',
    activeSessionOtherGoal: 'Finish or discard the active session before opening another goal.',
    legal: 'Legal',
    privacyPolicy: 'Privacy Policy',
    privacyText: 'Progress Tracker stores your account details, goals, sessions, notes, tags, and progress to provide the service. Your learning data is separated by account and is not sold or used for advertising. You can export your data or permanently delete the account.',
    termsOfUse: 'Terms of Use',
    termsText: 'Progress Tracker is a personal productivity tool. You are responsible for the information you save and for keeping access to your account secure. The service is provided without guarantees of uninterrupted availability while it remains in beta.',
    overview: 'Overview',
    overallProgress: 'Overall progress',
    totalPractice: 'Total practice',
    dailyTargetError: 'Daily target must be greater than 0',
    createGoalError: 'Could not create goal',
    loadGoalError: 'Could not load goal',
    updateGoalError: 'Could not update goal',
    saveSessionError: 'Could not save session',
    updateSessionError: 'Could not update session',
    deleteGoalError: 'Could not delete goal',
    deleteSessionError: 'Could not delete session',
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
    signInTitle: 'Добро пожаловать',
    registerTitle: 'Создание аккаунта',
    authSubtitle: 'Получите доступ к своим целям, сессиям, сериям и личной истории прогресса.',
    signIn: 'Войти',
    createAccount: 'Создать аккаунт',
    email: 'Email',
    name: 'Имя',
    password: 'Пароль',
    confirmPassword: 'Повторите пароль',
    currentPassword: 'Текущий пароль',
    newPassword: 'Новый пароль',
    passwordHint: 'Минимум 8 символов, заглавная буква, цифра и спецсимвол.',
    passwordMismatch: 'Пароли не совпадают',
    passwordPolicyError: 'Пароль должен содержать заглавную букву, цифру и спецсимвол.',
    showPassword: 'Показать пароль',
    hidePassword: 'Скрыть пароль',
    noAccount: 'Еще нет аккаунта?',
    forgotPassword: 'Забыли пароль?',
    forgotPasswordTitle: 'Восстановление доступа',
    resetPasswordTitle: 'Новый пароль',
    sendResetLink: 'Отправить ссылку',
    resetPassword: 'Сохранить новый пароль',
    backToSignIn: 'Вернуться ко входу',
    resetEmailSent: 'Если аккаунт существует, ссылка для восстановления отправлена.',
    passwordResetDone: 'Пароль обновлен. Войдите с новым паролем.',
    verifyEmailTitle: 'Подтвердите email',
    verifyEmailText: 'Откройте ссылку из письма, прежде чем пользоваться целями.',
    resendVerification: 'Отправить письмо повторно',
    verificationSent: 'Письмо с подтверждением отправлено.',
    haveAccount: 'Уже есть аккаунт?',
    authError: 'Не удалось выполнить вход',
    account: 'Аккаунт',
    signedInAs: 'Вы вошли как',
    displayName: 'Имя',
    saveProfile: 'Сохранить',
    profileSaved: 'Профиль обновлен',
    profileError: 'Не удалось обновить профиль',
    changePassword: 'Сменить пароль',
    passwordChanged: 'Пароль изменен',
    passwordChangeError: 'Не удалось сменить пароль',
    logout: 'Выйти',
    deleteAccount: 'Удалить аккаунт',
    deleteAccountHint: 'Цели, сессии и данные аккаунта будут удалены безвозвратно.',
    deleteAccountConfirm: 'Введите текущий пароль, чтобы навсегда удалить аккаунт.',
    deleteAccountAction: 'Удалить навсегда',
    accountDeleted: 'Аккаунт удален',
    exportData: 'Экспорт данных',
    exportJSON: 'Скачать JSON',
    exportCSV: 'Скачать CSV',
    exportError: 'Не удалось экспортировать данные',
    logoutError: 'Не удалось выйти',
    loadingGoals: 'Загрузка целей...',
    emptyGoalTitle: 'Создайте первую цель',
    emptyGoalText: 'Выберите фокус, задайте дневную норму и отслеживайте реальные занятия через таймер.',
    createGoal: 'Создать цель',
    myGoals: 'Мои цели',
    activeGoals: 'активных',
    goalFallbackDescription: 'Долгосрочная учебная цель',
    streak: 'Серия',
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
    createHint: 'Цель начинается сегодня автоматически. Серия считается по дням.',
    templates: 'Быстрые шаблоны',
    templateCoding: 'Кодинг',
    templateLanguage: 'Язык',
    templateFitness: 'Тренировки',
    templateCodingTitle: 'Изучить программирование',
    templateCodingDescription: 'Ежедневно изучать концепции и развивать небольшой проект.',
    templateLanguageTitle: 'Изучить язык',
    templateLanguageDescription: 'Ежедневно заниматься словарём, аудированием и разговорной практикой.',
    templateFitnessTitle: 'Ежедневные тренировки',
    templateFitnessDescription: 'Соблюдать регулярный план и отмечать каждую тренировку.',
    editGoal: 'Редактировать цель',
    adjustTarget: 'изменить цель',
    cancel: 'Отмена',
    saveChanges: 'Сохранить',
    dayStreak: 'дней серии',
    goalCompleted: 'Цель выполнена',
    leftToday: 'осталось сегодня',
    todayTarget: 'Дневная норма',
    practiced: 'занятий',
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
    emptyStatsTitle: 'Данных о прогрессе пока нет',
    emptyStatsText: 'Создайте цель и завершите первую сессию по таймеру, чтобы появилась статистика.',
    allGoals: 'Все цели',
    selectedGoal: 'Выбранная цель',
    calendar: 'Календарь',
    completedDays: 'Выполненные дни',
    missedDays: 'Пропущенные дни',
    completionRate: 'Процент выполнения',
    weeklyCompletionRate: 'Процент за неделю',
    previousWeek: 'Прошлая неделя',
    weekComparison: 'Сравнение недели',
    targetReached: 'Дневная норма выполнена',
    remainingToday: 'осталось сегодня',
    noDailyTarget: 'Нет активной дневной нормы',
    moreThanPrevious: 'больше прошлой недели',
    lessThanPrevious: 'меньше прошлой недели',
    sameAsPrevious: 'как на прошлой неделе',
    completedDay: 'Выполнено',
    partialDay: 'Частично',
    missedDay: 'Пропущено',
    history: 'История',
    recent: 'последних',
    emptyHistory: 'Сессий пока нет. Запустите таймер и сохраните результат.',
    historySearch: 'Поиск по заметкам или тегам',
    noHistoryResults: 'Подходящие сессии не найдены.',
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
    productTagline: 'Ежедневный фокус для долгосрочных учебных целей.',
    version: 'Версия',
    releaseChannel: 'Канал выпуска',
    beta: 'Бета',
    dataPrivacy: 'Ваши данные',
    privateAccountData: 'Доступны только вашему аккаунту',
    copyright: '© 2026 Progress Tracker',
    manageAccount: 'Управление аккаунтом',
    notifications: 'Уведомления',
    notificationDescription: 'Напоминания при открытом приложении и уведомления о завершении сессии.',
    enableNotifications: 'Включить',
    disableNotifications: 'Выключить',
    notificationsBlocked: 'Заблокированы в браузере',
    notificationsUnsupported: 'Не поддерживаются',
    notificationNote: 'Для фоновых уведомлений на iPhone приложение потребуется установить на экран «Домой» и подключить Web Push.',
    targetReachedNotification: 'Дневная норма выполнена',
    reminderNotification: 'Дневная норма ещё не выполнена',
    reminderNotificationBody: 'Откройте Progress Tracker и продолжите сегодняшнее занятие.',
    activeSessionOtherGoal: 'Завершите или отмените активную сессию перед открытием другой цели.',
    legal: 'Правовая информация',
    privacyPolicy: 'Политика конфиденциальности',
    privacyText: 'Progress Tracker хранит данные аккаунта, цели, сессии, заметки, теги и прогресс для работы сервиса. Учебные данные разделены по аккаунтам, не продаются и не используются для рекламы. Данные можно экспортировать, а аккаунт удалить безвозвратно.',
    termsOfUse: 'Условия использования',
    termsText: 'Progress Tracker — персональный инструмент продуктивности. Вы отвечаете за сохранённую информацию и безопасность доступа к аккаунту. Пока приложение находится в бета-версии, сервис предоставляется без гарантии непрерывной доступности.',
    overview: 'Обзор',
    overallProgress: 'Общий прогресс',
    totalPractice: 'Всего практики',
    dailyTargetError: 'Дневная норма должна быть больше 0',
    createGoalError: 'Не удалось создать цель',
    loadGoalError: 'Не удалось загрузить цель',
    updateGoalError: 'Не удалось обновить цель',
    saveSessionError: 'Не удалось сохранить сессию',
    updateSessionError: 'Не удалось обновить сессию',
    deleteGoalError: 'Не удалось удалить цель',
    deleteSessionError: 'Не удалось удалить сессию',
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
  const [currentUser, setCurrentUser] = useState<AuthUser | null>(null)
  const [authMode, setAuthMode] = useState<AuthMode>(() => (
    new URLSearchParams(window.location.search).has('resetToken') ? 'reset' : 'login'
  ))
  const [authForm, setAuthForm] = useState<AuthForm>({ email: '', name: '', password: '', confirmPassword: '' })
  const [authError, setAuthError] = useState('')
  const [authMessage, setAuthMessage] = useState('')
  const [passwordResetToken, setPasswordResetToken] = useState(
    () => new URLSearchParams(window.location.search).get('resetToken') ?? '',
  )
  const [verificationMessage, setVerificationMessage] = useState('')
  const [accountName, setAccountName] = useState('')
  const [passwordForm, setPasswordForm] = useState<PasswordForm>({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  const [accountMessage, setAccountMessage] = useState('')
  const [accountError, setAccountError] = useState('')
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoalId, setSelectedGoalId] = useState<number | null>(null)
  const [goalDetail, setGoalDetail] = useState<GoalDetail | null>(null)
  const [stats, setStats] = useState<AppStats>(defaultStats)
  const [selectedStatsGoalId, setSelectedStatsGoalId] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [formError, setFormError] = useState('')

  const [goalForm, setGoalForm] = useState<GoalForm>(() => createDefaultGoalForm(settings))

  const [sessionState, setSessionState] = useState<SessionState>('idle')
  const [sessionGoalId, setSessionGoalId] = useState<number | null>(null)
  const [elapsedSeconds, setElapsedSeconds] = useState(0)
  const [timerTargetSeconds, setTimerTargetSeconds] = useState(0)
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
  const [notificationPermission, setNotificationPermission] = useState<NotificationPermission | 'unsupported'>(() => (
    'Notification' in window ? Notification.permission : 'unsupported'
  ))

  useEffect(() => {
    let isMounted = true

    async function bootApp() {
      setIsLoading(true)

      try {
        const response = await fetch('/api/health')
        const data = (await response.json()) as { status: string }
        if (isMounted) {
          setBackendStatus(data.status === 'ok' ? 'connected' : 'error')
        }
      } catch {
        if (isMounted) {
          setBackendStatus('error')
        }
      }

      try {
        const verificationToken = new URLSearchParams(window.location.search).get('verifyToken')
        if (verificationToken) {
          await requestAPI('/api/auth/verify-email', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ token: verificationToken }),
          })
          window.history.replaceState({}, '', window.location.pathname)
        }

        const accountResponse = await fetch('/api/me', { credentials: 'same-origin' })
        if (!accountResponse.ok) {
          throw new Error('No active session')
        }

        let user = (await accountResponse.json()) as AuthUser
        const timezone = browserTimezone()
        if (user.timezone !== timezone) {
          const timezoneResponse = await requestAPI('/api/me/timezone', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ timezone }),
          })
          if (timezoneResponse.ok) {
            user = (await timezoneResponse.json()) as AuthUser
          }
        }

        const [goalsResponse, statsResponse, timerResponse] = await Promise.all([
          fetch('/api/goals', { credentials: 'same-origin' }),
          fetch('/api/stats', { credentials: 'same-origin' }),
          fetch('/api/timer', { credentials: 'same-origin' }),
        ])

        if (!goalsResponse.ok || !statsResponse.ok || !timerResponse.ok) {
          throw new Error('Failed to load account data')
        }

        const [loadedGoals, loadedStats, timerStatus] = await Promise.all([
          goalsResponse.json() as Promise<GoalSummary[]>,
          statsResponse.json() as Promise<AppStats>,
          timerResponse.json() as Promise<TimerStatusResponse>,
        ])

        let restoredDetail: GoalDetail | null = null
        if (timerStatus.active && timerStatus.timer && loadedGoals.some((goal) => goal.id === timerStatus.timer?.goalId)) {
          const detailResponse = await fetch(`/api/goals/${timerStatus.timer.goalId}`, { credentials: 'same-origin' })
          if (detailResponse.ok) {
            restoredDetail = (await detailResponse.json()) as GoalDetail
          }
        }

        if (isMounted) {
          setCurrentUser(user)
          setGoals(loadedGoals)
          setStats(loadedStats)
          if (timerStatus.active && timerStatus.timer && restoredDetail) {
            setSelectedGoalId(timerStatus.timer.goalId)
            setGoalDetail(restoredDetail)
            setSessionGoalId(timerStatus.timer.goalId)
            setElapsedSeconds(timerStatus.timer.elapsedSeconds)
            setTimerTargetSeconds(timerStatus.timer.targetSeconds)
            setSessionState(timerStatus.timer.state === 'running' ? 'running' : 'paused')
            setFinishModalOpen(timerStatus.timer.state === 'finished')
          }
        }
      } catch {
        if (isMounted) {
          setCurrentUser(null)
          setGoals([])
          setGoalDetail(null)
          setSelectedGoalId(null)
          setSelectedStatsGoalId(0)
          setStats(defaultStats)
        }
      } finally {
        if (isMounted) {
          setIsLoading(false)
        }
      }
    }

    void bootApp()

    return () => {
      isMounted = false
    }
  }, [])

  useEffect(() => {
    window.localStorage.setItem(settingsStorageKey, JSON.stringify(settings))
  }, [settings])

  useEffect(() => {
    setAccountName(currentUser?.name || '')
  }, [currentUser])

  useEffect(() => {
    if (!currentUser || !settings.notificationsEnabled || notificationPermission !== 'granted') {
      return
    }

    function remindAboutDailyGoals() {
      const now = new Date()
      if (now.getHours() < 20) {
        return
      }

      const reminderId = `${currentUser?.id}:${localDateString(now)}`
      if (window.localStorage.getItem(reminderStorageKey) === reminderId) {
        return
      }

      const hasIncompleteGoal = goals.some((goal) => (
        goal.status === 'active' && goal.todayMinutes < goal.dailyTargetMinutes
      ))
      if (!hasIncompleteGoal) {
        return
      }

      showBrowserNotification(copy.reminderNotification, copy.reminderNotificationBody)
      window.localStorage.setItem(reminderStorageKey, reminderId)
    }

    remindAboutDailyGoals()
    const reminderTimer = window.setInterval(remindAboutDailyGoals, 60_000)
    return () => window.clearInterval(reminderTimer)
  }, [copy, currentUser, goals, notificationPermission, settings.notificationsEnabled])

  useEffect(() => {
    if (sessionState !== 'running' || !goalDetail) {
      return
    }

    const timer = window.setInterval(() => {
      setElapsedSeconds((current) => {
        const next = current + 1

        if (timerTargetSeconds > 0 && next >= timerTargetSeconds) {
          window.clearInterval(timer)
          setSessionState('paused')
          setFinishModalOpen(true)
          if (settings.notificationsEnabled && notificationPermission === 'granted') {
            showBrowserNotification(copy.targetReachedNotification, goalDetail.title)
          }
          return timerTargetSeconds
        }

        return next
      })
    }, 1000)

    return () => window.clearInterval(timer)
  }, [copy.targetReachedNotification, goalDetail, notificationPermission, sessionState, settings.notificationsEnabled, timerTargetSeconds])

  useEffect(() => {
    if (!currentUser || sessionState === 'idle') {
      return
    }

    const sync = () => void syncActiveTimer()
    const syncTimer = window.setInterval(sync, 10_000)
    window.addEventListener('focus', sync)
    return () => {
      window.clearInterval(syncTimer)
      window.removeEventListener('focus', sync)
    }
  }, [currentUser, sessionState])

  const screenTitle = useMemo(() => {
    if (!currentUser) {
      return ''
    }

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
  }, [copy, currentUser, selectedGoalId, view])

  async function apiFetch(path: string, options: RequestInit = {}) {
    const response = await requestAPI(path, options)
    if (response.status === 401) {
      handleAuthReset()
    }
    return response
  }

  async function syncUserTimezone(user: AuthUser) {
    const timezone = browserTimezone()
    if (user.timezone === timezone) {
      return user
    }
    const response = await requestAPI('/api/me/timezone', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ timezone }),
    })
    if (!response.ok) {
      return user
    }
    return (await response.json()) as AuthUser
  }

  async function readApiError(response: Response, fallback: string) {
    return readAPIError(response, fallback)
  }

  async function syncActiveTimer() {
    try {
      const response = await apiFetch('/api/timer')
      if (!response.ok) {
        return
      }
      const status = (await response.json()) as TimerStatusResponse
      if (!status.active || !status.timer) {
        resetSession()
        return
      }

      const timer = status.timer
      setSessionGoalId(timer.goalId)
      setElapsedSeconds(timer.elapsedSeconds)
      setTimerTargetSeconds(timer.targetSeconds)
      setSessionState(timer.state === 'running' ? 'running' : 'paused')
      if (timer.state === 'finished') {
        setFinishModalOpen(true)
      }
    } catch {
      // Keep the local display running and retry on the next synchronization.
    }
  }

  function handleAuthReset() {
    setCurrentUser(null)
    setGoals([])
    setGoalDetail(null)
    setSelectedGoalId(null)
    setSelectedStatsGoalId(0)
    setStats(defaultStats)
    setView('goals')
    setAccountMessage('')
    setAccountError('')
    setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
    resetSession()
  }

  async function handleAuthSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAuthError('')
    setAuthMessage('')

    if (authMode === 'register' || authMode === 'reset') {
      if (authForm.password !== authForm.confirmPassword) {
        setAuthError(copy.passwordMismatch)
        return
      }
      if (!isStrongPassword(authForm.password)) {
        setAuthError(copy.passwordPolicyError)
        return
      }
    }

    try {
      if (authMode === 'forgot') {
        const response = await requestAPI('/api/auth/forgot-password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: authForm.email }),
        })
        if (!response.ok) {
          setAuthError(await readApiError(response, copy.authError))
          return
        }
        const data = (await response.json()) as { developmentToken?: string }
        setAuthMessage(copy.resetEmailSent)
        if (data.developmentToken) {
          setPasswordResetToken(data.developmentToken)
          setAuthMode('reset')
        }
        return
      }

      if (authMode === 'reset') {
        if (!passwordResetToken) {
          setAuthError(copy.authError)
          return
        }
        const response = await requestAPI('/api/auth/reset-password', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token: passwordResetToken, newPassword: authForm.password }),
        })
        if (!response.ok) {
          setAuthError(await readApiError(response, copy.authError))
          return
        }
        setPasswordResetToken('')
        setAuthForm({ email: authForm.email, name: '', password: '', confirmPassword: '' })
        setAuthMode('login')
        setAuthMessage(copy.passwordResetDone)
        window.history.replaceState({}, '', window.location.pathname)
        return
      }

      const response = await fetch(`/api/auth/${authMode === 'login' ? 'login' : 'register'}`, {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json',
        },
          body: JSON.stringify({
            email: authForm.email,
            name: authForm.name,
            password: authForm.password,
            timezone: browserTimezone(),
          }),
      })

      if (!response.ok) {
        setAuthError(await readApiError(response, copy.authError))
        return
      }

      let data = (await response.json()) as AuthResponse
      if (authMode === 'register' && data.developmentToken) {
        const verifyResponse = await requestAPI('/api/auth/verify-email', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ token: data.developmentToken }),
        })
        if (verifyResponse.ok) {
          data = (await verifyResponse.json()) as AuthResponse
        }
      }
      const syncedUser = await syncUserTimezone(data.user)
      setCurrentUser(syncedUser)
      setAuthForm({ email: '', name: '', password: '', confirmPassword: '' })
      setSelectedGoalId(null)
      setSelectedStatsGoalId(0)
      setGoalDetail(null)
      setView('goals')
      await Promise.all([loadGoals(), loadStats(0)])
    } catch {
      setAuthError(copy.authError)
    }
  }

  async function handleLogout() {
    try {
      await apiFetch('/api/auth/logout', { method: 'POST' })
    } catch (error) {
      window.alert(error instanceof Error ? `${copy.logoutError}: ${error.message}` : copy.logoutError)
    } finally {
      handleAuthReset()
    }
  }

  async function handleDeleteAccount(password: string) {
    setAccountMessage('')
    setAccountError('')
    const response = await apiFetch('/api/me', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    })
    if (!response.ok) {
      setAccountError(await readApiError(response, copy.deleteAccount))
      return false
    }
    handleAuthReset()
    setAuthMessage(copy.accountDeleted)
    return true
  }

  async function exportAccount(format: 'json' | 'csv') {
    setAccountError('')
    try {
      const response = await apiFetch(`/api/me/export?format=${format}`)
      if (!response.ok) {
        setAccountError(await readApiError(response, copy.exportError))
        return
      }
      const blob = await response.blob()
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = format === 'csv' ? 'progress-tracker-sessions.csv' : 'progress-tracker-export.json'
      link.click()
      URL.revokeObjectURL(link.href)
    } catch {
      setAccountError(copy.exportError)
    }
  }

  async function resendVerification() {
    setVerificationMessage('')
    const response = await apiFetch('/api/auth/resend-verification', { method: 'POST' })
    if (!response.ok) {
      setVerificationMessage(await readApiError(response, copy.authError))
      return
    }
    setVerificationMessage(copy.verificationSent)
  }

  async function updateProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAccountMessage('')
    setAccountError('')

    const response = await apiFetch('/api/me', {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: accountName }),
    })

    if (!response.ok) {
      setAccountError(await readApiError(response, copy.profileError))
      return
    }

    const user = (await response.json()) as AuthUser
    setCurrentUser(user)
    setAccountMessage(copy.profileSaved)
  }

  async function changePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAccountMessage('')
    setAccountError('')

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      setAccountError(copy.passwordMismatch)
      return
    }
    if (!isStrongPassword(passwordForm.newPassword)) {
      setAccountError(copy.passwordPolicyError)
      return
    }

    const response = await apiFetch('/api/me/password', {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        currentPassword: passwordForm.currentPassword,
        newPassword: passwordForm.newPassword,
      }),
    })

    if (!response.ok) {
      setAccountError(await readApiError(response, copy.passwordChangeError))
      return
    }

    setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' })
    setAccountMessage(copy.passwordChanged)
  }

  async function loadGoals() {
    try {
      const response = await apiFetch('/api/goals')
      if (!response.ok) {
        throw new Error(await readApiError(response, copy.loadingGoals))
      }
      const data = (await response.json()) as GoalSummary[]
      setGoals(data)
    } catch {
      setGoals([])
    }
  }

  async function loadStats(goalId = selectedStatsGoalId) {
    try {
      const query = goalId ? `?goalId=${goalId}` : ''
      const response = await apiFetch(`/api/stats${query}`)
      if (!response.ok) {
        throw new Error(await readApiError(response, copy.screenStats))
      }
      const data = (await response.json()) as AppStats
      setStats(data)
    } catch {
      setStats(defaultStats)
    }
  }

  async function openGoal(goalId: number) {
    if (sessionState !== 'idle' && sessionGoalId && sessionGoalId !== goalId) {
      window.alert(copy.activeSessionOtherGoal)
      goalId = sessionGoalId
    }

    setSelectedGoalId(goalId)
    setView('goals')
    setIsEditingGoal(false)
    setEditingSessionId(null)
    await loadGoalDetail(goalId)
  }

  async function loadGoalDetail(goalId: number) {
    const response = await apiFetch(`/api/goals/${goalId}`)
    if (!response.ok) {
      window.alert(await readApiError(response, copy.loadGoalError))
      setGoalDetail(null)
      setSelectedGoalId(null)
      return
    }
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
      const response = await apiFetch('/api/goals', {
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
        setFormError(await readApiError(response, copy.createGoalError))
        return
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

  async function startSession() {
    if (!goalDetail) {
      return
    }

    if (goalDetail.todayMinutes >= goalDetail.dailyTargetMinutes) {
      window.alert(copy.todayCompleteAlert)
      return
    }

    try {
      const response = await apiFetch(`/api/goals/${goalDetail.id}/timer/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ speedMultiplier: 1 }),
      })
      if (!response.ok) {
        setFormError(await readApiError(response, copy.saveSessionError))
        return
      }
      const timer = (await response.json()) as ActiveTimer
      setSessionGoalId(timer.goalId)
      setElapsedSeconds(timer.elapsedSeconds)
      setTimerTargetSeconds(timer.targetSeconds)
      setSessionState('running')
      setFormError('')
    } catch {
      setFormError(copy.saveSessionError)
    }
  }

  async function pauseSession() {
    if (!sessionGoalId) {
      return
    }
    const response = await apiFetch(`/api/goals/${sessionGoalId}/timer/pause`, { method: 'POST' })
    if (!response.ok) {
      setFormError(await readApiError(response, copy.saveSessionError))
      return
    }
    const timer = (await response.json()) as ActiveTimer
    setElapsedSeconds(timer.elapsedSeconds)
    setSessionState('paused')
  }

  async function resumeSession() {
    if (!sessionGoalId) {
      return
    }
    const response = await apiFetch(`/api/goals/${sessionGoalId}/timer/resume`, { method: 'POST' })
    if (!response.ok) {
      setFormError(await readApiError(response, copy.saveSessionError))
      return
    }
    const timer = (await response.json()) as ActiveTimer
    setElapsedSeconds(timer.elapsedSeconds)
    setSessionState(timer.state === 'running' ? 'running' : 'paused')
  }

  async function finishSession() {
    if (sessionState === 'running') {
      await pauseSession()
    }
    setFinishModalOpen(true)
  }

  async function saveSession() {
    if (!goalDetail) {
      return
    }

    const response = await apiFetch(`/api/goals/${goalDetail.id}/timer/finish`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        notes: sessionNotes,
        tags: sessionTags
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      }),
    })

    if (!response.ok) {
      setFormError(await readApiError(response, copy.saveSessionError))
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

    const response = await apiFetch(`/api/goals/${goalDetail.id}`, {
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
      setFormError(await readApiError(response, copy.updateGoalError))
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

    const response = await apiFetch(`/api/goals/${goalDetail.id}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      window.alert(await readApiError(response, copy.deleteGoalError))
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

    const response = await apiFetch(`/api/goals/${goalDetail.id}/sessions/${sessionId}`, {
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
      window.alert(await readApiError(response, copy.updateSessionError))
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

    const response = await apiFetch(`/api/goals/${goalDetail.id}/sessions/${session.id}`, {
      method: 'DELETE',
    })

    if (!response.ok) {
      window.alert(await readApiError(response, copy.deleteSessionError))
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
    setSessionGoalId(null)
    setTimerTargetSeconds(0)
    setFinishModalOpen(false)
    setSessionNotes('')
    setSessionTags('')
    setFormError('')
  }

  function leaveGoalView() {
    if (sessionState === 'running') {
      void pauseSession()
    }
    setSelectedGoalId(null)
    setGoalDetail(null)
    setIsEditingGoal(false)
    setEditingSessionId(null)
  }

  async function toggleNotifications() {
    if (settings.notificationsEnabled) {
      setSettings((current) => ({ ...current, notificationsEnabled: false }))
      return
    }

    if (!('Notification' in window)) {
      setNotificationPermission('unsupported')
      return
    }

    try {
      const permission = await Notification.requestPermission()
      setNotificationPermission(permission)
      if (permission === 'granted') {
        setSettings((current) => ({ ...current, notificationsEnabled: true }))
        showBrowserNotification('Progress Tracker', copy.notificationDescription)
      }
    } catch {
      setNotificationPermission('unsupported')
    }
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
            {currentUser ? (
              <button
                className={`icon-button ${selectedGoalId ? 'icon-button--back' : 'icon-button--menu'}`}
                type="button"
                aria-label={selectedGoalId ? copy.backToGoals : copy.openSettings}
                onClick={() => {
                  if (selectedGoalId) {
                    leaveGoalView()
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
            ) : (
              <span />
            )}
            <p>{screenTitle}</p>
            <span
              className={`connection-dot connection-dot--${backendStatus}`}
              title={backendStatus}
            />
          </header>

          {!currentUser && (
            <AuthScreen
              mode={authMode}
              form={authForm}
              error={authError}
              message={authMessage}
              copy={copy}
              onModeChange={(mode) => {
                setAuthMode(mode)
                setAuthError('')
                setAuthMessage('')
              }}
              onChange={setAuthForm}
              onSubmit={handleAuthSubmit}
            />
          )}

          {currentUser && !currentUser.emailVerified && (
            <section className="empty-state">
              <div className="flame-orb" aria-hidden="true"><FlameIcon /></div>
              <h1>{copy.verifyEmailTitle}</h1>
              <p>{copy.verifyEmailText}</p>
              {verificationMessage && <p className="settings-success">{verificationMessage}</p>}
              <button className="primary-button" type="button" onClick={() => void resendVerification()}>
                {copy.resendVerification}
              </button>
              <button className="ghost-button" type="button" onClick={() => void handleLogout()}>{copy.logout}</button>
            </section>
          )}

          {currentUser && currentUser.emailVerified && (
            <>
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
                  elapsedSeconds={sessionGoalId === goalDetail.id ? elapsedSeconds : 0}
                  isEditingGoal={isEditingGoal}
                  editGoalForm={editGoalForm}
                  formError={formError}
                  sessionState={sessionGoalId === goalDetail.id ? sessionState : 'idle'}
                  editingSessionId={editingSessionId}
                  sessionEditNotes={sessionEditNotes}
                  sessionEditTags={sessionEditTags}
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
                <GoalForm
                  mode="create"
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
            </>
          )}
        </div>

        {currentUser?.emailVerified && <nav className="bottom-nav" aria-label={copy.mainNavigation}>
          <button
            className={view === 'goals' ? 'is-active' : ''}
            type="button"
            onClick={() => {
              setView('goals')
              leaveGoalView()
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
              leaveGoalView()
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
              leaveGoalView()
              void loadStats(selectedStatsGoalId)
            }}
            aria-label={copy.navStats}
          >
            <ChartIcon />
          </button>
        </nav>}

        {currentUser?.emailVerified && finishModalOpen && goalDetail && sessionGoalId === goalDetail.id && (
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
          isOpen={Boolean(currentUser && settingsOpen)}
          settings={settings}
          currentUser={currentUser}
          accountName={accountName}
          passwordForm={passwordForm}
          accountMessage={accountMessage}
          accountError={accountError}
          appVersion={__APP_VERSION__}
          notificationPermission={notificationPermission}
          copy={copy}
          onClose={() => setSettingsOpen(false)}
          onChange={(nextSettings) => setSettings((current) => ({ ...current, ...nextSettings }))}
          onAccountNameChange={setAccountName}
          onPasswordFormChange={setPasswordForm}
          onProfileSubmit={updateProfile}
          onPasswordSubmit={changePassword}
          onLogout={handleLogout}
          onDeleteAccount={handleDeleteAccount}
          onExport={(format) => void exportAccount(format)}
          onToggleNotifications={toggleNotifications}
        />
      </section>
    </main>
  )
}

function GoalDetailsScreen({
  goal,
  copy,
  language,
  elapsedSeconds,
  isEditingGoal,
  editGoalForm,
  formError,
  sessionState,
  editingSessionId,
  sessionEditNotes,
  sessionEditTags,
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
  isEditingGoal: boolean
  editGoalForm: GoalForm
  formError: string
  sessionState: SessionState
  editingSessionId: number | null
  sessionEditNotes: string
  sessionEditTags: string
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
  const [activeTab, setActiveTab] = useState<'overview' | 'history'>('overview')
  const ringStyle = {
    '--ring-progress': `${goal.totalProgressPct}%`,
  } as CSSProperties

  return (
    <>
      <section className="goal-overview">
        <div className="goal-overview__header">
          <div>
            <div className="goal-overview__title">
              <h1>{goal.title}</h1>
              <span className={`status-pill status-pill--${goal.status}`}>
                {statusLabel(goal.status, copy)}
              </span>
            </div>
            {goal.description && <p>{goal.description}</p>}
          </div>
          <button
            className="icon-button goal-overview__edit"
            type="button"
            aria-label={copy.editGoal}
            title={copy.editGoal}
            onClick={onEditGoalStart}
          >
            <EditIcon />
          </button>
        </div>

        <div className="goal-overview__summary">
          <div className="goal-overview__ring">
            <div
              className="hero-ring"
              style={ringStyle}
              aria-label={`${copy.goalCompleted} ${goal.totalProgressPct}%`}
            >
              <span>{goal.totalProgressPct}%</span>
            </div>
            <span>{copy.overallProgress}</span>
          </div>
          <div className="goal-overview__metrics">
            <article>
              <span>{copy.streak}</span>
              <strong>{goal.currentStreak} {copy.days.toLowerCase()}</strong>
            </article>
            <article>
              <span>{copy.totalPractice}</span>
              <strong>{formatMinutes(goal.totalPracticeMinutes)}</strong>
            </article>
          </div>
        </div>

        <div className="goal-overview__today">
          <div>
            <span>{copy.todayTarget}</span>
            <strong>{formatMinutes(liveTodayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}</strong>
          </div>
          <span>{liveTodayProgressPct}%</span>
        </div>
        <ProgressBar value={liveTodayProgressPct} />
        <small>{formatMinutes(liveRemainingMinutes)} {copy.leftToday}</small>
      </section>

      {isEditingGoal && (
        <GoalForm
          mode="edit"
          form={editGoalForm}
          formError={formError}
          copy={copy}
          onChange={onEditGoalChange}
          onSubmit={onEditGoalSubmit}
          onCancel={onEditGoalCancel}
        />
      )}

      {!isEditingGoal && (
        <nav className="goal-tabs" aria-label={copy.screenGoal}>
          <button
            className={activeTab === 'overview' ? 'is-active' : ''}
            type="button"
            onClick={() => setActiveTab('overview')}
          >
            {copy.overview}
          </button>
          <button
            className={activeTab === 'history' ? 'is-active' : ''}
            type="button"
            onClick={() => setActiveTab('history')}
          >
            {copy.history}
          </button>
        </nav>
      )}

      {!isEditingGoal && activeTab === 'overview' && <>
        <section className="timer-panel">
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

        <ActivityCalendar
          days={goal.calendar}
          copy={copy}
          language={language}
        />
      </>}

      {!isEditingGoal && activeTab === 'history' && <>
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
      </>}
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

function percent(value: number, total: number) {
  if (total <= 0) {
    return 0
  }

  return Math.min(Math.round((value / total) * 100), 100)
}

function localDateString(date: Date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function showBrowserNotification(title: string, body: string) {
  if (!('Notification' in window) || Notification.permission !== 'granted') {
    return
  }
  try {
    new Notification(title, { body, icon: '/favicon.svg' })
  } catch {
    // Mobile browsers may require notifications to be shown by a service worker.
  }
}

function isStrongPassword(password: string) {
  return password.length >= 8
    && /[A-Z]/.test(password)
    && /\d/.test(password)
    && /[^A-Za-z0-9]/.test(password)
}

function browserTimezone() {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
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

function EditIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M4 20h4l11-11-4-4L4 16v4Z" />
      <path d="m13.5 6.5 4 4" />
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


