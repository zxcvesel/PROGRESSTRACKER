type CalendarDay = {
  date: string
  minutes: number
  targetMinutes: number
  isCompleted: boolean
}

type CalendarCopy = {
  calendar: string
  days: string
  completedDay: string
  partialDay: string
  missedDay: string
}

type ActivityCalendarProps = {
  days: CalendarDay[]
  copy: CalendarCopy
  language: 'en' | 'ru'
}

export function ActivityCalendar({ days, copy, language }: ActivityCalendarProps) {
  return (
    <section className="chart-panel">
      <div className="section-heading">
        <h2>{copy.calendar}</h2>
        <span>{days.length} {copy.days}</span>
      </div>
      <div className="activity-calendar" aria-label={copy.calendar}>
        {days.map((day) => {
          const state = calendarDayState(day)
          const description = `${formatFullDate(day.date, language)} · ${formatMinutes(day.minutes)} / ${formatMinutes(day.targetMinutes)}`

          return (
            <span
              className={`calendar-day calendar-day--${state}`}
              key={day.date}
              title={description}
              aria-label={description}
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

function calendarDayState(day: CalendarDay) {
  if (day.targetMinutes <= 0) return 'empty'
  if (day.isCompleted) return 'completed'
  if (day.minutes > 0) return 'partial'
  return 'missed'
}

function formatMinutes(totalMinutes: number) {
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60

  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function formatFullDate(value: string, language: 'en' | 'ru') {
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-RU' : 'en-US', {
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}
