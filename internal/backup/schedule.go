package backup

import (
	"fmt"
	"sort"
)

// SlotTimes returns HH:MM strings for evenly spaced daily backup slots.
func SlotTimes(startHour, startMinute, perDay int) ([]string, error) {
	if perDay < 1 || perDay > 24 {
		return nil, fmt.Errorf("backups_per_day must be between 1 and 24")
	}
	if startHour < 0 || startHour > 23 || startMinute < 0 || startMinute > 59 {
		return nil, fmt.Errorf("invalid start time")
	}
	startMin := startHour*60 + startMinute
	interval := (24 * 60) / perDay
	slots := make([]string, 0, perDay)
	seen := make(map[int]struct{}, perDay)
	for i := 0; i < perDay; i++ {
		m := (startMin + i*interval) % (24 * 60)
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		slots = append(slots, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	sort.Strings(slots)
	return slots, nil
}

// CronExprs converts slot times to cron expressions (minute hour * * *).
func CronExprs(slots []string) ([]string, error) {
	out := make([]string, 0, len(slots))
	for _, slot := range slots {
		var hh, mm int
		if _, err := fmt.Sscanf(slot, "%d:%d", &hh, &mm); err != nil {
			return nil, fmt.Errorf("invalid slot %q", slot)
		}
		out = append(out, fmt.Sprintf("%d %d * * *", mm, hh))
	}
	return out, nil
}