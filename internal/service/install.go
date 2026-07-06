package service

import "github.com/simplefrp/simplefrp/internal/app"

func Install(role app.Role) error { return installService(role) }
func Start(role app.Role) error   { return startService(role) }
func Stop(role app.Role) error    { return stopService(role) }
func Enable(role app.Role) error  { return enableService(role) }
func Disable(role app.Role) error { return disableService(role) }
