package services

// 状态转换合法性表
var allowedTransitions = map[ServerStatus]map[ServerStatus]bool{
	StatusNotInstalled: {StatusInstalling: true},
	StatusInstalling:   {StatusStopped: true, StatusNotInstalled: true},
	StatusStopped:      {StatusStarting: true, StatusUpdating: true},
	StatusStarting:     {StatusRunning: true, StatusError: true},
	StatusRunning:      {StatusStopping: true, StatusError: true, StatusStopped: true},
	StatusStopping:     {StatusStopped: true},
	StatusUpdating:     {StatusStopped: true},
	StatusError:        {StatusStarting: true},
}
