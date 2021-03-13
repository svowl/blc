package conf

// Config is app config structure
type Config struct {
	Server struct {
		Addr string
	}
	Crawler
	Auth struct {
		Userslist string
	}
	SMTP
	Reports struct {
		MaxReportsToStore int
	}
	Schedule map[string]*ScheduleData
}

// Crawler config
type Crawler struct {
	Delay int
}

// SMTP config
type SMTP struct {
	Addr       string
	From       string
	ReplyTo    string
	To         string
	Username   string
	Password   string
	Encryption string
	Domain     string
}

// ScheduleData config
type ScheduleData struct {
	URL         []string
	Depth       int
	Cron        string
	SessionName string
	ExcludedURL []string
}
