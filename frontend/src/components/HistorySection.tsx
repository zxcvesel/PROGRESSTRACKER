import { useState } from 'react'

export type HistorySession = {
  id: number
  goalId: number
  startedAt: string
  endedAt: string
  durationMinutes: number
  notes: string
  tags: string[]
  createdAt: string
}

type HistoryCopy = {
  history: string
  recent: string
  historySearch: string
  noHistoryResults: string
  emptyHistory: string
  notes: string
  tags: string
  tagsPlaceholder: string
  cancel: string
  save: string
  edit: string
  delete: string
}

type HistorySectionProps = {
  sessions: HistorySession[]
  copy: HistoryCopy
  language: 'en' | 'ru'
  editingSessionId: number | null
  editNotes: string
  editTags: string
  onEditStart: (session: HistorySession) => void
  onEditCancel: () => void
  onEditSave: (sessionId: number) => void
  onDelete: (session: HistorySession) => void
  onNotesChange: (value: string) => void
  onTagsChange: (value: string) => void
}

const markerColors = ['#19f7e8', '#ff7a3d', '#e6d37a', '#b45cff', '#58d8ff']

export function HistorySection({
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
}: HistorySectionProps) {
  const [query, setQuery] = useState('')
  const normalizedQuery = query.trim().toLocaleLowerCase(language === 'ru' ? 'ru-RU' : 'en-US')
  const visibleSessions = normalizedQuery
    ? sessions.filter((session) => (
      session.notes.toLocaleLowerCase().includes(normalizedQuery)
      || session.tags.some((tag) => tag.toLocaleLowerCase().includes(normalizedQuery))
      || formatSessionDate(session.endedAt, language).toLocaleLowerCase().includes(normalizedQuery)
    ))
    : sessions

  return (
    <section className="entries-section">
      <div className="section-heading">
        <h2>{copy.history}</h2>
        <span>{sessions.length} {copy.recent}</span>
      </div>
      {sessions.length > 0 && (
        <input
          className="history-search"
          type="search"
          value={query}
          placeholder={copy.historySearch}
          aria-label={copy.historySearch}
          onChange={(event) => setQuery(event.target.value)}
        />
      )}
      {sessions.length === 0 && <p className="empty-message">{copy.emptyHistory}</p>}
      {sessions.length > 0 && visibleSessions.length === 0 && (
        <p className="empty-message">{copy.noHistoryResults}</p>
      )}
      {visibleSessions.map((session, index) => (
        <article className="history-card" key={session.id}>
          <span className="entry-marker" style={{ backgroundColor: markerColors[index % markerColors.length] }} />
          <div>
            <p>{formatSessionDate(session.endedAt, language)}</p>
            <strong>{formatMinutes(session.durationMinutes)}</strong>
            {editingSessionId === session.id ? (
              <div className="history-edit-form">
                <label>
                  {copy.notes}
                  <textarea value={editNotes} onChange={(event) => onNotesChange(event.target.value)} rows={3} />
                </label>
                <label>
                  {copy.tags}
                  <input value={editTags} onChange={(event) => onTagsChange(event.target.value)} placeholder={copy.tagsPlaceholder} />
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
                    {session.tags.map((tag) => <small key={tag}>{tag}</small>)}
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

function formatMinutes(totalMinutes: number) {
  const hours = Math.floor(totalMinutes / 60)
  const minutes = totalMinutes % 60
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function formatSessionDate(value: string, language: 'en' | 'ru') {
  return new Intl.DateTimeFormat(language === 'ru' ? 'ru-RU' : 'en-US', {
    month: 'long',
    day: 'numeric',
  }).format(new Date(value))
}
