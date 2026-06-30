import { useEffect, useState } from 'react'
import './App.css'

type Entry = {
  id: number
  date: string
  category: string
  minutes: number
  note: string
}

type Stats = {
  totalEntries: number
  totalMinutes: number
}

function App() {
  const [backendStatus, setBackendStatus] = useState('checking')
  const [entries, setEntries] = useState<Entry[]>([])
  const [stats, setStats] = useState<Stats>({
    totalEntries: 0,
    totalMinutes: 0,
  })
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [category, setCategory] = useState('')
  const [minutes, setMinutes] = useState('')
  const [note, setNote] = useState('')
  const [formError, setFormError] = useState('')
  const [isSaving, setIsSaving] = useState(false)

  useEffect(() => {
    fetch('/api/health')
      .then((response) => response.json())
      .then((data: { status: string }) => {
        setBackendStatus(data.status === 'ok' ? 'connected' : 'error')
      })
      .catch(() => {
        setBackendStatus('error')
      })

    fetch('/api/entries')
      .then((response) => response.json())
      .then((data: Entry[]) => {
        setEntries(data)
      })
      .catch(() => {
        setEntries([])
      })

    fetch('/api/stats')
      .then((response) => response.json())
      .then((data: Stats) => {
        setStats(data)
      })
      .catch(() => {
        setStats({ totalEntries: 0, totalMinutes: 0 })
      })
  }, [])

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setFormError('')
    setIsSaving(true)

    fetch('/api/entries', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        date,
        category,
        minutes: Number(minutes),
        note,
      }),
    })
      .then((response) => {
        if (!response.ok) {
          throw new Error('Failed to save entry')
        }

        return response.json()
      })
      .then((newEntry: Entry) => {
        setEntries((currentEntries) => [newEntry, ...currentEntries])
        setStats((currentStats) => ({
          totalEntries: currentStats.totalEntries + 1,
          totalMinutes: currentStats.totalMinutes + newEntry.minutes,
        }))
        setCategory('')
        setMinutes('')
        setNote('')
      })
      .catch(() => {
        setFormError('Could not save entry')
      })
      .finally(() => {
        setIsSaving(false)
      })
  }

  return (
    <main className="app">
      <section className="hero">
        <p className="eyebrow">Learning project</p>
        <h1>Progress Tracker</h1>
        <p className="intro">
          A small mobile-first app for tracking daily learning progress.
        </p>
      </section>

      <section className="status-card">
        <span className={`status-dot status-dot--${backendStatus}`} />
        <div>
          <h2>Backend status</h2>
          <p>
            {backendStatus === 'checking' && 'Checking connection...'}
            {backendStatus === 'connected' && 'Backend connected'}
            {backendStatus === 'error' && 'Backend is not available'}
          </p>
        </div>
      </section>

      <section className="stats-grid">
        <article className="stat-card">
          <p>Total entries</p>
          <strong>{stats.totalEntries}</strong>
        </article>
        <article className="stat-card">
          <p>Total study time</p>
          <strong>{stats.totalMinutes} min</strong>
        </article>
      </section>

      <form className="entry-form" onSubmit={handleSubmit}>
        <h2>Add entry</h2>

        <label>
          Date
          <input
            type="date"
            value={date}
            onChange={(event) => setDate(event.target.value)}
            required
          />
        </label>

        <label>
          Category
          <input
            value={category}
            onChange={(event) => setCategory(event.target.value)}
            placeholder="Go, React, SQL"
            required
          />
        </label>

        <label>
          Minutes
          <input
            type="number"
            min="1"
            value={minutes}
            onChange={(event) => setMinutes(event.target.value)}
            required
          />
        </label>

        <label>
          Note
          <textarea
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder="What did you learn?"
            rows={3}
          />
        </label>

        {formError && <p className="form-error">{formError}</p>}

        <button type="submit" disabled={isSaving}>
          {isSaving ? 'Saving...' : 'Save entry'}
        </button>
      </form>

      <section className="entries-section">
        <h2>Latest entries</h2>
        {entries.length === 0 && (
          <p className="empty-message">No entries yet. Add your first one.</p>
        )}
        {entries.map((entry) => (
          <article className="entry-card" key={entry.id}>
            <div>
              <p className="entry-date">{entry.date}</p>
              <h3>{entry.category}</h3>
            </div>
            <p className="entry-minutes">{entry.minutes} min</p>
            <p className="entry-note">{entry.note}</p>
          </article>
        ))}
      </section>
    </main>
  )
}

export default App
