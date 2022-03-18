package everyday

import "time"

// StartEverydayJob起一个协程，每天在给定时间点（本地时间）跑给定的任务。
func StartEverydayJob(hour, minute int, job func()) {
	now := time.Now()
	firstSleep := calculateNextTimeDuration(now, hour, minute)
	go func() {
		time.Sleep(firstSleep)
		job()
		for {
			time.Sleep(time.Hour * 24)
			job()
		}
	}()
}

// calculateNextTimeDuration以一个给定的时间点（都是本地时间），计算距离下一次触发任务还需要多久。
func calculateNextTimeDuration(now time.Time, hour, minute int) time.Duration {
	year := now.Year()
	month := now.Month()
	day := now.Day()
	todayTrigger := time.Date(year, month, day, hour, minute, 0, 0, time.Local)
	if todayTrigger.Before(now) {
		return todayTrigger.Add(time.Hour * 24).Sub(now)
	}
	return todayTrigger.Sub(now)
}
