package utils

import "github.com/robfig/cron/v3"

var CronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
