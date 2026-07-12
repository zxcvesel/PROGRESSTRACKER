type GoalSummary = {
  id: number
  title: string
  description: string
  totalDays: number
  dailyTargetMinutes: number
  status: 'active' | 'completed'
  currentStreak: number
  todayMinutes: number
  todayProgressPct: number
  totalProgressPct: number
}

type GoalsCopy = {
  loadingGoals: string
  emptyGoalTitle: string
  emptyGoalText: string
  createGoal: string
  myGoals: string
  activeGoals: string
  goalFallbackDescription: string
  streak: string
  today: string
  day: string
  of: string
  total: string
  statusActive: string
  statusCompleted: string
}

type GoalsScreenProps = {
  goals: GoalSummary[]
  isLoading: boolean
  copy: GoalsCopy
  onCreate: () => void
  onOpenGoal: (goalId: number) => void
}

const markerColors = ['#19f7e8', '#ff7a3d', '#e6d37a', '#b45cff', '#58d8ff']

export function GoalsScreen({ goals, isLoading, copy, onCreate, onOpenGoal }: GoalsScreenProps) {
  if (isLoading) {
    return <p className="empty-message">{copy.loadingGoals}</p>
  }

  if (goals.length === 0) {
    return (
      <section className="empty-state">
        <div className="flame-orb" aria-hidden="true"><FlameIcon /></div>
        <h1>{copy.emptyGoalTitle}</h1>
        <p>{copy.emptyGoalText}</p>
        <button className="primary-button" type="button" onClick={onCreate}>{copy.createGoal}</button>
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
            <span className="entry-marker" style={{ backgroundColor: markerColors[index % markerColors.length] }} />
            <div>
              <h3>{goal.title}</h3>
              <p>{goal.description || copy.goalFallbackDescription}</p>
            </div>
            <span className={`status-pill status-pill--${goal.status}`}>
              {goal.status === 'completed' ? copy.statusCompleted : copy.statusActive}
            </span>
          </div>

          <div className="goal-card__metrics">
            <span>{copy.streak}: {goal.currentStreak}</span>
            <span>{copy.today}: {formatMinutes(goal.todayMinutes)} / {formatMinutes(goal.dailyTargetMinutes)}</span>
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

function ProgressBar({ value }: { value: number }) {
  return <span className="bar-track"><span style={{ width: `${Math.min(value, 100)}%` }} /></span>
}

function formatMinutes(totalMinutes: number) {
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function FlameIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M12.2 3.5c.5 2.7-.6 4.4-2 5.9-1.3 1.4-2.5 2.7-2.5 4.9a4.4 4.4 0 0 0 8.8.1c0-1.8-.9-3.4-2.4-4.8.1 1.4-.4 2.4-1.5 3.1-.4-2.6.8-4.7-.4-9.2Z" />
      <path d="M12 20.8c-4 0-7.1-2.8-7.1-6.8 0-2.7 1.5-4.6 3.1-6.2 1.5-1.5 3.1-3.1 3.2-5.8 4 2.8 6.4 6.4 6.4 10.6 1.1-.9 1.6-2.1 1.6-3.5 1.4 1.5 2 3.1 2 4.8 0 4.1-3.2 6.9-9.2 6.9Z" />
    </svg>
  )
}
