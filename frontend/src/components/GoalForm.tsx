import type { FormEvent } from 'react'

export type GoalFormValue = {
  title: string
  description: string
  totalDays: string
  dailyTargetHours: string
  dailyTargetMinutes: string
}

type GoalFormCopy = {
  createGoalTitle: string
  longTermFocus: string
  editGoal: string
  adjustTarget: string
  title: string
  description: string
  days: string
  dailyTargetHours: string
  minutes: string
  createHint: string
  createGoal: string
  cancel: string
  saveChanges: string
}

type GoalFormProps = {
  mode: 'create' | 'edit'
  form: GoalFormValue
  formError: string
  copy: GoalFormCopy
  onChange: (form: GoalFormValue) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  onCancel?: () => void
}

export function GoalForm({ mode, form, formError, copy, onChange, onSubmit, onCancel }: GoalFormProps) {
  const isEdit = mode === 'edit'

  return (
    <form className="entry-form" onSubmit={onSubmit}>
      <div className="section-heading">
        <h2>{isEdit ? copy.editGoal : copy.createGoalTitle}</h2>
        <span>{isEdit ? copy.adjustTarget : copy.longTermFocus}</span>
      </div>

      <label>
        {copy.title}
        <input
          value={form.title}
          onChange={(event) => onChange({ ...form, title: event.target.value })}
          placeholder={isEdit ? undefined : 'Learn Go'}
          required
        />
      </label>

      <label>
        {copy.description}
        <textarea
          value={form.description}
          onChange={(event) => onChange({ ...form, description: event.target.value })}
          placeholder={isEdit ? undefined : 'Study the language, build APIs, and reinforce with practice'}
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

      {!isEdit && <p className="form-hint">{copy.createHint}</p>}
      {formError && <p className="form-error">{formError}</p>}

      {isEdit ? (
        <div className="sheet-actions">
          <button className="ghost-button" type="button" onClick={onCancel}>{copy.cancel}</button>
          <button className="primary-button" type="submit">{copy.saveChanges}</button>
        </div>
      ) : (
        <button className="primary-button" type="submit">{copy.createGoal}</button>
      )}
    </form>
  )
}
