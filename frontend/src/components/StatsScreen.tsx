import type { CSSProperties } from 'react'
import { ActivityCalendar } from './ActivityCalendar'

type GoalSummary = {
  id: number
  title: string
}

type DailyStat = {
  date: string
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

type Stats = {
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
  weekly: DailyStat[]
  calendar: DailyStat[]
  monthlyTotalMinutes: number
  goalDistribution: GoalDistribution[]
}

type StatsCopy = {
  screenStats: string
  selectedGoal: string
  allGoals: string
  completionRate: string
  completedDays: string
  missedDays: string
  currentStreak: string
  longestStreak: string
  sessions: string
  practice: string
  today: string
  todayTarget: string
  targetReached: string
  remainingToday: string
  noDailyTarget: string
  week: string
  weeklyCompletionRate: string
  previousWeek: string
  moreThanPrevious: string
  lessThanPrevious: string
  sameAsPrevious: string
  actualVsTarget: string
  month: string
  emptyDistribution: string
  calendar: string
  days: string
  completedDay: string
  partialDay: string
  missedDay: string
}

type StatsScreenProps = {
  stats: Stats
  goals: GoalSummary[]
  selectedGoalId: number
  copy: StatsCopy
  language: 'en' | 'ru'
  onGoalChange: (goalId: number) => void
}

const markerColors = ['#19f7e8', '#ff7a3d', '#e6d37a', '#b45cff', '#58d8ff']

export function StatsScreen({
  stats,
  goals,
  selectedGoalId,
  copy,
  language,
  onGoalChange,
}: StatsScreenProps) {
  const todayPercent = percent(stats.todayMinutes, stats.dailyTargetMinutes)
  const remainingToday = Math.max(stats.dailyTargetMinutes - stats.todayMinutes, 0)
  const currentWeekMinutes = stats.weekly.reduce((total, day) => total + day.minutes, 0)
  const comparisonCopy = stats.weekComparisonPct > 0
    ? copy.moreThanPrevious
    : stats.weekComparisonPct < 0
      ? copy.lessThanPrevious
      : copy.sameAsPrevious
  const completionStyle = {
    '--stats-ring-progress': `${stats.completionRate}%`,
  } as CSSProperties

  return (
    <>
      <section className="stats-filter-panel">
        <label className="stats-goal-filter">
          {copy.selectedGoal}
          <select value={selectedGoalId} onChange={(event) => onGoalChange(Number(event.target.value))}>
            <option value={0}>{copy.allGoals}</option>
            {goals.map((goal) => (
              <option value={goal.id} key={goal.id}>{goal.title}</option>
            ))}
          </select>
        </label>
      </section>

      <section className="stats-overview" aria-label={copy.screenStats}>
        <div className="stats-completion-ring" style={completionStyle}>
          <div>
            <strong>{stats.completionRate}%</strong>
            <span>{copy.completionRate}</span>
          </div>
        </div>
        <div className="stats-overview__summary">
          <article>
            <span>{copy.completedDays}</span>
            <strong>{stats.completedDays}</strong>
          </article>
          <article>
            <span>{copy.missedDays}</span>
            <strong>{stats.missedDays}</strong>
          </article>
          <article>
            <span>{copy.currentStreak}</span>
            <strong>{stats.currentStreak}</strong>
          </article>
          <article>
            <span>{copy.longestStreak}</span>
            <strong>{stats.longestStreak}</strong>
          </article>
        </div>
      </section>

      <section className="stats-grid" aria-label={copy.screenStats}>
        <article className="stat-card">
          <p>{copy.sessions}</p>
          <strong>{stats.totalSessions}</strong>
        </article>
        <article className="stat-card">
          <p>{copy.practice}</p>
          <strong>{formatMinutes(stats.totalPracticeMinutes)}</strong>
        </article>
      </section>

      <section className={`chart-panel today-progress ${todayPercent >= 100 ? 'is-complete' : ''}`}>
        <div className="section-heading">
          <h2>{copy.today}</h2>
          <span>{todayPercent}%</span>
        </div>
        <div className="today-progress__numbers">
          <strong>{formatMinutes(stats.todayMinutes)}</strong>
          <span>/ {formatMinutes(stats.dailyTargetMinutes)} {copy.todayTarget.toLowerCase()}</span>
        </div>
        <ProgressBar value={todayPercent} />
        <p className="chart-caption">
          {stats.dailyTargetMinutes <= 0
            ? copy.noDailyTarget
            : remainingToday === 0
              ? copy.targetReached
              : `${formatMinutes(remainingToday)} ${copy.remainingToday}`}
        </p>
      </section>

      <section className="chart-panel weekly-panel">
        <div className="section-heading">
          <h2>{copy.week}</h2>
          <span>{stats.weeklyCompletionRate}% {copy.weeklyCompletionRate.toLowerCase()}</span>
        </div>
        <div className="week-comparison">
          <div>
            <strong>{formatMinutes(currentWeekMinutes)}</strong>
            <span>{copy.week}</span>
          </div>
          <div className={stats.weekComparisonPct < 0 ? 'is-negative' : 'is-positive'}>
            <strong>{stats.weekComparisonPct > 0 ? '+' : ''}{stats.weekComparisonPct}%</strong>
            <span>{comparisonCopy}</span>
          </div>
        </div>
        <p className="chart-caption">{copy.previousWeek}: {formatMinutes(stats.previousWeekMinutes)}</p>
        <div className="weekly-chart" aria-label={copy.actualVsTarget}>
          {stats.weekly.map((day) => {
            const value = percent(day.minutes, day.targetMinutes)

            return (
              <div className="week-day" key={day.date} title={`${formatMinutes(day.minutes)} / ${formatMinutes(day.targetMinutes)}`}>
                <small className="week-day__value">{shortMinutes(day.minutes)}</small>
                <span className="week-bar">
                  <span
                    className={day.isCompleted ? 'is-complete' : ''}
                    style={{ height: `${Math.max(Math.min(value, 100), day.minutes > 0 ? 8 : 0)}%` }}
                  />
                </span>
                <small>{formatWeekday(day.date, language)}</small>
              </div>
            )
          })}
        </div>
        <div className="weekly-legend">
          <span><i className="weekly-legend__target" />100% {copy.todayTarget.toLowerCase()}</span>
          <span><i className="weekly-legend__actual" />{copy.actualVsTarget}</span>
        </div>
      </section>

      <ActivityCalendar days={stats.calendar} copy={copy} language={language} />

      <section className="chart-panel">
        <div className="section-heading">
          <h2>{copy.month}</h2>
          <span>{formatMinutes(stats.monthlyTotalMinutes)}</span>
        </div>
        {stats.goalDistribution.length === 0 && <p className="empty-message">{copy.emptyDistribution}</p>}
        {stats.goalDistribution.map((item, index) => (
          <article className="category-row" key={item.goalId}>
            <span className="entry-marker" style={{ backgroundColor: markerColors[index % markerColors.length] }} />
            <div>
              <div className="category-line">
                <p>{item.title}</p>
                <strong>{formatMinutes(item.minutes)} · {item.percent}%</strong>
              </div>
              <ProgressBar value={item.percent} />
            </div>
          </article>
        ))}
      </section>
    </>
  )
}

function ProgressBar({ value }: { value: number }) {
  return <span className="bar-track"><span style={{ width: `${Math.min(value, 100)}%` }} /></span>
}

function percent(value: number, total: number) {
  if (total <= 0) return 0
  return Math.round((value / total) * 100)
}

function formatMinutes(totalMinutes: number) {
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function shortMinutes(minutes: number) {
  if (minutes === 0) return '0'
  if (minutes >= 60) return `${Math.round((minutes / 60) * 10) / 10}h`
  return `${minutes}m`
}

function formatWeekday(value: string, language: 'en' | 'ru') {
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-RU' : 'en-US', { weekday: 'short' })
    .format(new Date(value))
}
