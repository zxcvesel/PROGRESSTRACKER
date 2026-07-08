package main

import (
	"database/sql"
	"net/http"
	"time"
)

func statsHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUserFromRequest(w, r)
	if !ok {
		return
	}

	goals, err := loadGoals(user.ID)
	if err != nil {
		writeError(w, "failed to load goals", http.StatusInternalServerError)
		return
	}

	goalID, err := optionalGoalID(r)
	if err != nil {
		writeError(w, "invalid goal id", http.StatusBadRequest)
		return
	}

	if goalID != 0 {
		if _, err := loadGoal(goalID, user.ID); err == sql.ErrNoRows {
			writeError(w, "goal not found", http.StatusNotFound)
			return
		} else if err != nil {
			writeError(w, "failed to load goal", http.StatusInternalServerError)
			return
		}
	}

	totalSessions, totalMinutes, err := loadSessionTotals(goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load stats", http.StatusInternalServerError)
		return
	}

	today := todayString()
	todayMinutes, err := loadMinutesForDate(today, goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load today's stats", http.StatusInternalServerError)
		return
	}

	dailyTarget := 0
	currentStreak := 0
	longestStreak := 0
	goalTitle := ""
	for _, goal := range goals {
		if goalID != 0 && goal.ID != goalID {
			continue
		}

		if goalID == goal.ID {
			goalTitle = goal.Title
		}

		if goal.Status == "active" {
			dailyTarget += goal.DailyTargetMinutes
		}

		current, longest, err := calculateGoalStreaks(goal)
		if err != nil {
			writeError(w, "failed to load streaks", http.StatusInternalServerError)
			return
		}

		currentStreak = max(currentStreak, current)
		longestStreak = max(longestStreak, longest)
	}

	weekly, err := buildWeeklyStats(dailyTarget, goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load weekly stats", http.StatusInternalServerError)
		return
	}

	completedDays, missedDays, completionRate, err := loadCompletionSummary(goalID, user.ID, "", "")
	if err != nil {
		writeError(w, "failed to load completion stats", http.StatusInternalServerError)
		return
	}

	weekStart := dateOnly(time.Now()).AddDate(0, 0, -6).Format(time.DateOnly)
	weekEnd := todayString()
	_, _, weeklyCompletionRate, err := loadCompletionSummary(goalID, user.ID, weekStart, weekEnd)
	if err != nil {
		writeError(w, "failed to load weekly completion stats", http.StatusInternalServerError)
		return
	}

	currentWeekMinutes, err := loadMinutesBetween(weekStart, weekEnd, goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load current week stats", http.StatusInternalServerError)
		return
	}

	previousWeekStart := dateOnly(time.Now()).AddDate(0, 0, -13).Format(time.DateOnly)
	previousWeekEnd := dateOnly(time.Now()).AddDate(0, 0, -7).Format(time.DateOnly)
	previousWeekMinutes, err := loadMinutesBetween(previousWeekStart, previousWeekEnd, goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load previous week stats", http.StatusInternalServerError)
		return
	}

	monthlyTotal, err := loadMonthlyTotal(goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load monthly stats", http.StatusInternalServerError)
		return
	}

	distribution, err := loadGoalDistribution(totalMinutes, goalID, user.ID)
	if err != nil {
		writeError(w, "failed to load distribution", http.StatusInternalServerError)
		return
	}

	calendar, err := loadCalendarStats(goalID, user.ID, 42)
	if err != nil {
		writeError(w, "failed to load calendar stats", http.StatusInternalServerError)
		return
	}

	stats := Stats{
		TotalSessions:        totalSessions,
		TotalPracticeMinutes: totalMinutes,
		CurrentStreak:        currentStreak,
		LongestStreak:        longestStreak,
		CompletedDays:        completedDays,
		MissedDays:           missedDays,
		CompletionRate:       completionRate,
		WeeklyCompletionRate: weeklyCompletionRate,
		PreviousWeekMinutes:  previousWeekMinutes,
		WeekComparisonPct:    signedPercentChange(currentWeekMinutes, previousWeekMinutes),
		TodayMinutes:         todayMinutes,
		DailyTargetMinutes:   dailyTarget,
		Weekly:               weekly,
		Calendar:             calendar,
		MonthlyTotalMinutes:  monthlyTotal,
		GoalDistribution:     distribution,
		GoalID:               goalID,
		GoalTitle:            goalTitle,
	}

	writeJSON(w, stats, http.StatusOK)
}
