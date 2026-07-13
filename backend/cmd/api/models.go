package main

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"createdAt"`
}

type AuthRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Name string `json:"name"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type AuthResponse struct {
	User User `json:"user"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type Entry struct {
	ID       int    `json:"id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Minutes  int    `json:"minutes"`
	Note     string `json:"note"`
}

type Goal struct {
	ID                 int    `json:"id"`
	UserID             int    `json:"-"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	ActiveWeekdays     []int  `json:"activeWeekdays"`
	StartDate          string `json:"startDate"`
	CreatedAt          string `json:"createdAt"`
	Status             string `json:"status"`
}

type GoalSummary struct {
	Goal
	CurrentStreak       int `json:"currentStreak"`
	TodayMinutes        int `json:"todayMinutes"`
	TodayProgressPct    int `json:"todayProgressPct"`
	CurrentDay          int `json:"currentDay"`
	TotalProgressPct    int `json:"totalProgressPct"`
	TotalPracticeMinute int `json:"totalPracticeMinutes"`
}

type Session struct {
	ID              int      `json:"id"`
	GoalID          int      `json:"goalId"`
	StartedAt       string   `json:"startedAt"`
	EndedAt         string   `json:"endedAt"`
	DurationMinutes int      `json:"durationMinutes"`
	Notes           string   `json:"notes"`
	Tags            []string `json:"tags"`
	CreatedAt       string   `json:"createdAt"`
}

type GoalDetail struct {
	GoalSummary
	TodayRemainingMinutes int         `json:"todayRemainingMinutes"`
	RecentSessions        []Session   `json:"recentSessions"`
	Calendar              []DailyStat `json:"calendar"`
}

type CreateGoalRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	ActiveWeekdays     []int  `json:"activeWeekdays"`
	StartDate          string `json:"startDate"`
}

type UpdateGoalRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	TotalDays          int    `json:"totalDays"`
	DailyTargetMinutes int    `json:"dailyTargetMinutes"`
	Status             string `json:"status"`
}

type CreateSessionRequest struct {
	StartedAt       string   `json:"startedAt"`
	EndedAt         string   `json:"endedAt"`
	DurationMinutes int      `json:"durationMinutes"`
	Notes           string   `json:"notes"`
	Tags            []string `json:"tags"`
}

type UpdateSessionRequest struct {
	Notes string   `json:"notes"`
	Tags  []string `json:"tags"`
}

type Stats struct {
	TotalSessions        int                `json:"totalSessions"`
	TotalPracticeMinutes int                `json:"totalPracticeMinutes"`
	CurrentStreak        int                `json:"currentStreak"`
	LongestStreak        int                `json:"longestStreak"`
	CompletedDays        int                `json:"completedDays"`
	MissedDays           int                `json:"missedDays"`
	CompletionRate       int                `json:"completionRate"`
	WeeklyCompletionRate int                `json:"weeklyCompletionRate"`
	PreviousWeekMinutes  int                `json:"previousWeekMinutes"`
	WeekComparisonPct    int                `json:"weekComparisonPct"`
	TodayMinutes         int                `json:"todayMinutes"`
	DailyTargetMinutes   int                `json:"dailyTargetMinutes"`
	Weekly               []DailyStat        `json:"weekly"`
	Calendar             []DailyStat        `json:"calendar"`
	MonthlyTotalMinutes  int                `json:"monthlyTotalMinutes"`
	GoalDistribution     []GoalDistribution `json:"goalDistribution"`
	GoalID               int                `json:"goalId"`
	GoalTitle            string             `json:"goalTitle"`
}

type DailyStat struct {
	Date          string `json:"date"`
	Label         string `json:"label"`
	Minutes       int    `json:"minutes"`
	TargetMinutes int    `json:"targetMinutes"`
	IsCompleted   bool   `json:"isCompleted"`
}

type GoalDistribution struct {
	GoalID  int    `json:"goalId"`
	Title   string `json:"title"`
	Minutes int    `json:"minutes"`
	Percent int    `json:"percent"`
}
